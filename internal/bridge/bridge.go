package bridge

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// OnMessageFunc is the callback invoked for every inbound BridgeMessage.
type OnMessageFunc func(BridgeMessage)

// BridgeConfig holds the parameters needed to establish a bridge connection.
type BridgeConfig struct {
	URL       string        // WebSocket endpoint (ws:// or wss://)
	SessionID string        // session identifier sent in every frame
	AuthToken string        // bearer token for the Upgrade request
	OnMessage OnMessageFunc // callback for inbound messages
}

// Bridge manages the lifecycle of a single WebSocket-like connection
// to a remote controller. The transport uses a simplified raw-TCP
// upgrade so no third-party WebSocket library is required.
type Bridge struct {
	cfg  BridgeConfig
	conn net.Conn

	mu        sync.RWMutex
	connected bool
	done      chan struct{}
}

// NewBridge creates a Bridge ready to connect.
func NewBridge(cfg BridgeConfig) *Bridge {
	return &Bridge{cfg: cfg, done: make(chan struct{})}
}

// IsConnected reports whether the bridge has an active connection.
func (b *Bridge) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// Connect performs the HTTP/1.1 Upgrade handshake and starts a
// read loop that delivers inbound frames to cfg.OnMessage.
func (b *Bridge) Connect(ctx context.Context) error {
	u, err := url.Parse(b.cfg.URL)
	if err != nil {
		return fmt.Errorf("bridge: invalid url: %w", err)
	}

	host, port := u.Hostname(), u.Port()
	if port == "" {
		port = defaultPort(u.Scheme)
	}

	conn, err := dialConn(ctx, u.Scheme, net.JoinHostPort(host, port))
	if err != nil {
		return fmt.Errorf("bridge: dial: %w", err)
	}

	if err := b.handshake(conn, u, host); err != nil {
		conn.Close()
		return err
	}

	b.mu.Lock()
	b.conn = conn
	b.connected = true
	b.done = make(chan struct{})
	b.mu.Unlock()

	go b.readLoop()
	return nil
}

// Disconnect gracefully tears down the connection.
func (b *Bridge) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil
	}
	b.connected = false
	close(b.done)

	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}

// Send encodes a BridgeMessage and writes it as a simplified
// WebSocket text frame (opcode 0x1, no masking, <=64 KiB).
func (b *Bridge) Send(msg BridgeMessage) error {
	b.mu.RLock()
	conn := b.conn
	ok := b.connected
	b.mu.RUnlock()

	if !ok || conn == nil {
		return errors.New("bridge: not connected")
	}

	data, err := EncodeBridgeMessage(msg)
	if err != nil {
		return err
	}

	return writeTextFrame(conn, data)
}

// --- internal helpers ---------------------------------------------------

func defaultPort(scheme string) string {
	if scheme == "wss" || scheme == "https" {
		return "443"
	}
	return "80"
}

func dialConn(ctx context.Context, scheme, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	if scheme == "wss" || scheme == "https" {
		return tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{MinVersion: tls.VersionTLS12})
	}

	return dialer.DialContext(ctx, "tcp", addr)
}

func (b *Bridge) handshake(conn net.Conn, u *url.URL, host string) error {
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Key: dGlDb2RlQnJpZGdl\r\n"

	if b.cfg.AuthToken != "" {
		req += "Authorization: Bearer " + b.cfg.AuthToken + "\r\n"
	}
	req += "\r\n"

	if _, err := io.WriteString(conn, req); err != nil {
		return fmt.Errorf("bridge: write handshake: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return fmt.Errorf("bridge: read handshake response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("bridge: upgrade rejected (status %d)", resp.StatusCode)
	}
	return nil
}

// readLoop reads simplified WebSocket frames until the connection closes.
func (b *Bridge) readLoop() {
	defer b.Disconnect()

	reader := bufio.NewReader(b.conn)
	for {
		select {
		case <-b.done:
			return
		default:
		}

		payload, err := readTextFrame(reader)
		if err != nil {
			return
		}

		msg, err := DecodeBridgeMessage(payload)
		if err != nil {
			continue
		}

		if b.cfg.OnMessage != nil {
			b.cfg.OnMessage(*msg)
		}
	}
}

// writeTextFrame writes a simplified WebSocket text frame.
// Opcode 0x81 = FIN + text. Length encoded per RFC 6455.
func writeTextFrame(conn net.Conn, payload []byte) error {
	n := len(payload)
	var header []byte

	switch {
	case n <= 125:
		header = []byte{0x81, byte(n)}
	case n <= 65535:
		header = make([]byte, 4)
		header[0] = 0x81
		header[1] = 126
		binary.BigEndian.PutUint16(header[2:], uint16(n))
	default:
		header = make([]byte, 10)
		header[0] = 0x81
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:], uint64(n))
	}

	buf := make([]byte, 0, len(header)+n)
	buf = append(buf, header...)
	buf = append(buf, payload...)

	_, err := conn.Write(buf)
	return err
}

// readTextFrame reads one simplified (unmasked) WebSocket frame.
func readTextFrame(r *bufio.Reader) ([]byte, error) {
	first, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	_ = first // opcode byte; we accept any opcode

	second, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	masked := second&0x80 != 0
	length := uint64(second & 0x7F)

	switch length {
	case 126:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf))
	case 127:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		length = binary.BigEndian.Uint64(buf)
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(r, maskKey[:]); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}

	return payload, nil
}
