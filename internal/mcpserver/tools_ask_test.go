package mcpserver

import (
	"context"
	"testing"
	"time"

	"github.com/eduardoworrel/worrel-agent-cockpit/internal/ask"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/bus"
	"github.com/eduardoworrel/worrel-agent-cockpit/internal/store"
)

func TestAskUserBlocksAndReturnsAnswer(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/t.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	sess, _ := s.CreateSession(&store.Session{Adapter: "claude-code", Mode: "wrapper"})

	b := ask.New()
	svc := New(s, bus.New()).WithAskBroker(b)

	resCh := make(chan string, 1)
	go func() {
		res := svc.handleAskUser(context.Background(), sess.ID, "", askUserArgs{
			Question: "A ou B?",
			Options:  []string{"A", "B"},
		})
		resCh <- textOf(res)
	}()

	var reqID string
	for i := 0; i < 100; i++ {
		if p := b.Pending(); len(p) == 1 {
			reqID = p[0].ID
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if reqID == "" {
		t.Fatal("ask_user never reached the broker")
	}
	b.Resolve(reqID, "B")

	select {
	case got := <-resCh:
		if got != "B" {
			t.Fatalf("answer = %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handleAskUser did not return")
	}
}
