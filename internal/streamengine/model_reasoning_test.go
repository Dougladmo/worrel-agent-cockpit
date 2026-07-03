package streamengine

import (
	"strings"
	"testing"
)

func TestClaudeArgsModel(t *testing.T) {
	args := strings.Join(claudeArgs(Opts{Model: "claude-opus-4-8"}), " ")
	if !strings.Contains(args, "--model claude-opus-4-8") {
		t.Errorf("claudeArgs sem --model: %q", args)
	}
	if strings.Contains(strings.Join(claudeArgs(Opts{}), " "), "--model") {
		t.Error("claudeArgs sem Model não deveria emitir --model")
	}
}

func TestThinkingTokens(t *testing.T) {
	cases := []struct {
		level string
		tok   int
		ok    bool
	}{
		{"", 0, false}, {"off", 0, true}, {"low", 4096, true},
		{"medium", 10000, true}, {"high", 32000, true},
	}
	for _, c := range cases {
		tok, ok := thinkingTokens(c.level)
		if ok != c.ok || tok != c.tok {
			t.Errorf("thinkingTokens(%q) = (%d,%v), quer (%d,%v)", c.level, tok, ok, c.tok, c.ok)
		}
	}
}

func TestCodexReasoningEffort(t *testing.T) {
	cases := map[string]string{
		"": "", "off": "minimal", "low": "low", "medium": "medium", "high": "high",
	}
	for level, want := range cases {
		if got := codexReasoningEffort(level); got != want {
			t.Errorf("codexReasoningEffort(%q) = %q, quer %q", level, got, want)
		}
	}
}
