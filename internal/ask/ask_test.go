package ask

import (
	"context"
	"testing"
	"time"
)

func TestOpenResolveWait(t *testing.T) {
	b := New()
	req, ch := b.Open(Request{SessionID: "s1", Kind: "permission", Title: "Rodar comando"})
	if req.ID == "" {
		t.Fatal("Open should assign an ID")
	}
	if got := b.Pending(); len(got) != 1 || got[0].ID != req.ID {
		t.Fatalf("Pending = %+v", got)
	}
	go func() {
		if !b.Resolve(req.ID, "allow") {
			t.Errorf("Resolve returned false")
		}
	}()
	answer, ok := b.Wait(context.Background(), ch)
	if !ok || answer != "allow" {
		t.Fatalf("Wait = %q, %v", answer, ok)
	}
	if len(b.Pending()) != 0 {
		t.Fatalf("Pending should be empty after Resolve")
	}
}

func TestResolveUnknown(t *testing.T) {
	b := New()
	if b.Resolve("nope", "allow") {
		t.Fatal("Resolve of unknown id should be false")
	}
}

func TestWaitCancelled(t *testing.T) {
	b := New()
	_, ch := b.Open(Request{SessionID: "s1", Kind: "choice"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	answer, ok := b.Wait(ctx, ch)
	if ok || answer != "" {
		t.Fatalf("cancelled Wait = %q, %v", answer, ok)
	}
}

func TestRemove(t *testing.T) {
	b := New()
	req, _ := b.Open(Request{SessionID: "s1"})
	r, ok := b.Remove(req.ID)
	if !ok || r.ID != req.ID {
		t.Fatalf("Remove = %+v, %v", r, ok)
	}
	if _, ok := b.Remove(req.ID); ok {
		t.Fatal("second Remove should be false")
	}
}

func TestWaitThenResolveRace(t *testing.T) {
	b := New()
	req, ch := b.Open(Request{SessionID: "s1"})
	done := make(chan string, 1)
	go func() {
		a, _ := b.Wait(context.Background(), ch)
		done <- a
	}()
	time.Sleep(10 * time.Millisecond)
	b.Resolve(req.ID, "yes")
	select {
	case a := <-done:
		if a != "yes" {
			t.Fatalf("got %q", a)
		}
	case <-time.After(time.Second):
		t.Fatal("Wait did not return")
	}
}
