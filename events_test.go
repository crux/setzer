package main

import "testing"

func TestEmitNeverBlocks(t *testing.T) {
	// nil channel (headless): no-op, no panic.
	(&server{}).emit(trayEvent{kind: "published"})

	// buffered with a listener: delivered.
	s := &server{events: make(chan trayEvent, 1)}
	s.emit(trayEvent{kind: "published", body: "x"})
	select {
	case ev := <-s.events:
		if ev.body != "x" {
			t.Fatalf("got %q, want x", ev.body)
		}
	default:
		t.Fatal("event was not delivered")
	}

	// full buffer: must drop and return, not block.
	full := &server{events: make(chan trayEvent, 1)}
	full.emit(trayEvent{}) // fills the buffer
	full.emit(trayEvent{}) // would deadlock if emit blocked
}
