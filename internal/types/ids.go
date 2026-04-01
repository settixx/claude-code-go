package types

import "regexp"

// SessionId uniquely identifies a Ti Code session.
type SessionId string

// AgentId uniquely identifies a subagent within a session.
type AgentId string

// RequestId tracks individual API requests.
type RequestId string

var agentIDPattern = regexp.MustCompile(`^a(?:.+-)?[0-9a-f]{16}$`)

func ToAgentId(s string) (AgentId, bool) {
	if agentIDPattern.MatchString(s) {
		return AgentId(s), true
	}
	return "", false
}
