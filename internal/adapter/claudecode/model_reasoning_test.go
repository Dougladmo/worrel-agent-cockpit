package claudecode

import (
	"strings"
	"testing"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/adapter"
)

func TestBuildInteractiveModel(t *testing.T) {
	spec, err := New().BuildInteractive(adapter.SpawnOpts{
		WorkingDir: "/tmp/p", Model: "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(spec.Args, " ")
	if !strings.Contains(args, "--model claude-sonnet-4-6") {
		t.Errorf("args sem --model: %q", args)
	}
}

func TestBuildInteractiveReasoningEnv(t *testing.T) {
	cases := map[string]string{
		"":       "",                     // default: não emite env
		"off":    "MAX_THINKING_TOKENS=0", // desliga explicitamente
		"low":    "MAX_THINKING_TOKENS=4096",
		"medium": "MAX_THINKING_TOKENS=10000",
		"high":   "MAX_THINKING_TOKENS=32000",
	}
	for level, want := range cases {
		spec, err := New().BuildInteractive(adapter.SpawnOpts{WorkingDir: "/tmp/p", Reasoning: level})
		if err != nil {
			t.Fatal(err)
		}
		env := strings.Join(spec.Env, " ")
		if want == "" {
			if strings.Contains(env, "MAX_THINKING_TOKENS") {
				t.Errorf("reasoning %q: env deveria ser vazio, veio %q", level, env)
			}
			continue
		}
		if !strings.Contains(env, want) {
			t.Errorf("reasoning %q: env %q não contém %q", level, env, want)
		}
	}
}
