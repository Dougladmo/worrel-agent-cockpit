package codex

import (
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

func TestBuildInteractiveModelReasoning(t *testing.T) {
	spec, err := New().BuildInteractive(adapter.SpawnOpts{
		WorkingDir: "/tmp/p", Model: "gpt-5-codex", Reasoning: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(spec.Args, " ")
	if !strings.Contains(args, "-m gpt-5-codex") {
		t.Errorf("args sem -m: %q", args)
	}
	if !strings.Contains(args, `model_reasoning_effort="high"`) {
		t.Errorf("args sem reasoning effort: %q", args)
	}
}

func TestReasoningEffortOverride(t *testing.T) {
	cases := map[string]string{
		"":       "",
		"off":    "minimal",
		"low":    "low",
		"medium": "medium",
		"high":   "high",
	}
	for level, want := range cases {
		got := reasoningEffortOverride(level)
		if want == "" {
			if got != nil {
				t.Errorf("nível %q: esperava nil, veio %v", level, got)
			}
			continue
		}
		joined := strings.Join(got, " ")
		if !strings.Contains(joined, `model_reasoning_effort="`+want+`"`) {
			t.Errorf("nível %q: %q não contém effort %q", level, joined, want)
		}
	}
}
