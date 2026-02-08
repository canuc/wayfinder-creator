package main

import "sync"

type LogHub struct {
	mu       sync.Mutex
	channels map[int64]chan struct{}
}

func NewLogHub() *LogHub {
	return &LogHub{
		channels: make(map[int64]chan struct{}),
	}
}

func (h *LogHub) Notify(serverID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.channels[serverID]
	if !ok {
		return
	}
	close(ch)
	h.channels[serverID] = make(chan struct{})
}

func (h *LogHub) WaitChan(serverID int64) <-chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.channels[serverID]
	if !ok {
		ch = make(chan struct{})
		h.channels[serverID] = ch
	}
	return ch
}

func (h *LogHub) Remove(serverID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.channels[serverID]
	if ok {
		close(ch)
		delete(h.channels, serverID)
	}
}
