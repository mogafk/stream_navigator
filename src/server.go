package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type sseClient chan string

type sseBroker struct {
	mu      sync.Mutex
	clients map[sseClient]struct{}
}

var broker = &sseBroker{clients: make(map[sseClient]struct{})}

func (b *sseBroker) subscribe() sseClient {
	ch := make(sseClient, 8)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *sseBroker) unsubscribe(ch sseClient) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

func (b *sseBroker) publish(event string) {
	b.mu.Lock()
	for ch := range b.clients {
		select {
		case ch <- event:
		default: // skip slow/disconnected clients
		}
	}
	b.mu.Unlock()
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := broker.subscribe()
	defer broker.unsubscribe(ch)

	for {
		select {
		case event := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

var httpServer *http.Server

const staticDir = "src/client/build/client"

func startServer(port int) {
	mux := http.NewServeMux()

	mux.HandleFunc("/events", sseHandler)

	mux.HandleFunc("/active-config", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(rootCfg.ActiveConfig)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(data)
	})

	mux.HandleFunc("/obs-overlay", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})

	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	addr := fmt.Sprintf(":%d", port)
	httpServer = &http.Server{Addr: addr, Handler: mux}
	fmt.Printf("  overlay : http://localhost%s/obs-overlay\n", addr)
	go func() {
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Println("[server] error:", err)
		}
	}()
}

func stopServer() {
	if httpServer != nil {
		httpServer.Shutdown(context.Background())
	}
}
