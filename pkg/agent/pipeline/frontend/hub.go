//
// Copyright © 2018 Aljabr, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package frontend

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// Hub distributes messages to frontends
type Hub struct {
	log zerolog.Logger

	// Registered clients.
	clients map[*client]struct{}

	// Queue of messages to be broadcasted to all clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *client

	// Unregister requests from clients.
	unregister chan *client
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// NewHub initializes a new hub.
func NewHub(log zerolog.Logger) *Hub {
	return &Hub{
		log:        log,
		clients:    make(map[*client]struct{}),
		broadcast:  make(chan []byte, 32),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

// Run the hub until the given context is canceled.
func (h *Hub) Run(ctx context.Context) {
	defer func() {
		close(h.register)
		close(h.unregister)
		close(h.broadcast)
	}()

	for {
		select {
		case client := <-h.register:
			// Register client
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			// Unregister client
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			// Broadcast message to all clients
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		case <-ctx.Done():
			// Context canceled, close all clients
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// CreateHandler creates a websocket handler for the  hub.
func (h *Hub) CreateHandler() func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			h.log.Error().Err(err).Msg("Failed to upgrade websocket connection")
			return
		}
		// Create and register client
		c := &client{hub: h, conn: conn, send: make(chan []byte, 256)}
		h.register <- c

		// read & write in go routines.
		go c.writePump()
		go c.readPump()
	}
}

// StatisticsChanged sends a message to all clients notifying a statistics change.
func (h *Hub) StatisticsChanged() {
	h.broadcast <- []byte(`{"type":"statistics-changed"}`)
}
