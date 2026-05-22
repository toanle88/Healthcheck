package handler

import (
	"context"
	"log/slog"
	"sync"

	"github.com/toanle88/healthcheck/internal/store"
)

// Broker handles multiple SSE client connections and broadcasts check updates to them.
type Broker struct {
	clients    map[chan []store.Check]bool
	register   chan chan []store.Check
	unregister chan chan []store.Check
	broadcast  chan []store.Check
	mu         sync.Mutex
}

// NewBroker initializes a new Broker.
func NewBroker() *Broker {
	return &Broker{
		clients:    make(map[chan []store.Check]bool),
		register:   make(chan chan []store.Check),
		unregister: make(chan chan []store.Check),
		broadcast:  make(chan []store.Check),
	}
}

// Start runs the main event loop for the broker, managing client registrations and broadcasts.
func (b *Broker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Close all active client channels on shutdown
			b.mu.Lock()
			for client := range b.clients {
				close(client)
			}
			b.clients = make(map[chan []store.Check]bool)
			b.mu.Unlock()
			return
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()
			slog.Debug("SSE client connected")
		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
				slog.Debug("SSE client disconnected")
			}
			b.mu.Unlock()
		case checks := <-b.broadcast:
			b.mu.Lock()
			for client := range b.clients {
				select {
				case client <- checks:
				default:
					// Drop event if client channel is blocked
				}
			}
			b.mu.Unlock()
		}
	}
}

// Broadcast sends a new checks update list to all connected clients.
func (b *Broker) Broadcast(checks []store.Check) {
	select {
	case b.broadcast <- checks:
	default:
		// Drop message if main event loop is busy to prevent blocking
	}
}
