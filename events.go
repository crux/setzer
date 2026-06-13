package main

// trayEvent is a UI event surfaced by the tray (a notification + a status
// update). Handlers emit them; the tray consumes them in GUI mode, and they are
// dropped when headless.
type trayEvent struct {
	kind   string // "published" | "conflict"
	title  string // notification title
	body   string // notification body
	branch string // conflict branch
	url    string // conflict URL
}

// emit delivers a tray event without ever blocking the caller: if nothing is
// listening (headless) or the buffer is full, it's dropped.
func (s *server) emit(ev trayEvent) {
	if s.events == nil {
		return
	}
	select {
	case s.events <- ev:
	default:
	}
}
