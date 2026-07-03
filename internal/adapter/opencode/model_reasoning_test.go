package opencode

import (
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

func TestBuildInteractiveModelVariant(t *testing.T) {
	spec, err := New().BuildInteractive(adapter.SpawnOpts{
		WorkingDir: "/tmp/p", Model: "anthropic/claude-sonnet-4-6", Reasoning: "high",
	})
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(spec.Args, " ")
	if !strings.Contains(args, "--model anthropic/claude-sonnet-4-6") {
		t.Errorf("args sem --model: %q", args)
	}
	if !strings.Contains(args, "--variant high") {
		t.Errorf("args sem --variant: %q", args)
	}
}

func TestOpencodeVariant(t *testing.T) {
	cases := map[string]string{
		"": "", "off": "", "low": "low", "medium": "medium", "high": "high",
	}
	for level, want := range cases {
		if got := opencodeVariant(level); got != want {
			t.Errorf("opencodeVariant(%q) = %q, quer %q", level, got, want)
		}
	}
}
