package test

import (
	"strings"
	"testing"

	"github.com/settixx/claude-code-go/internal/query"
	"github.com/settixx/claude-code-go/internal/tools/bash"
	"github.com/settixx/claude-code-go/internal/types"
)

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkTokenEstimation(b *testing.B) {
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.EstimateTokens(text)
	}
}

func BenchmarkTokenEstimation_CJK(b *testing.B) {
	text := strings.Repeat("这是一段中文测试文本，用于估算令牌数量。", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.EstimateTokens(text)
	}
}

func BenchmarkSecurityValidation_SafeCommand(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bash.ValidateCommand("ls -la /tmp")
	}
}

func BenchmarkSecurityValidation_DangerousCommand(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bash.ValidateCommand("rm -rf /")
	}
}

func BenchmarkSecurityValidation_ComplexCommand(b *testing.B) {
	cmd := `git log --oneline --graph --decorate --all 2>&1`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bash.ValidateCommand(cmd)
	}
}

func BenchmarkConversationTokenEstimation(b *testing.B) {
	msgs := makeBenchMessages(50, 500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query.EstimateConversationTokens(msgs)
	}
}

func BenchmarkQuoteExtraction(b *testing.B) {
	cmd := `echo "hello world" | grep 'pattern' | awk '{print $1}'`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bash.ExtractQuotedContent(cmd)
	}
}

func makeBenchMessages(n int, textLen int) []types.Message {
	msgs := make([]types.Message, n)
	text := strings.Repeat("benchmark text ", textLen/15)
	for i := range msgs {
		role := "user"
		mt := types.MsgUser
		if i%2 == 1 {
			role = "assistant"
			mt = types.MsgAssistant
		}
		msgs[i] = types.Message{
			Type: mt,
			Role: role,
			Content: []types.ContentBlock{{
				Type: types.ContentText,
				Text: text,
			}},
		}
	}
	return msgs
}
