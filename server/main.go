// server/main.go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/vinizap/lumi/server/auth"
	httphandlers "github.com/vinizap/lumi/server/http"
	"github.com/vinizap/lumi/server/ws"
)

func main() {
	rootDir := os.Getenv("LUMI_ROOT")
	if rootDir == "" {
		rootDir = "./notes"
	}

	port := os.Getenv("LUMI_PORT")
	if port == "" {
		port = "8080"
	}

	hub := ws.NewHub()
	go hub.Run()

	server := httphandlers.NewServer(rootDir, hub)

	mux := http.NewServeMux()

	// CORS middleware
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Lumi-Token")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next(w, r)
		}
	}

	mux.HandleFunc("/api/folders", corsMiddleware(auth.Middleware(server.HandleFolders)))
	mux.HandleFunc("/api/notes", corsMiddleware(auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			server.HandleNotes(w, r)
		case http.MethodPost:
			server.HandleCreateNote(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.HandleFunc("/api/notes/", corsMiddleware(auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			server.HandleGetNote(w, r)
		case http.MethodPut:
			server.HandleUpdateNote(w, r)
		case http.MethodDelete:
			server.HandleDeleteNote(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	mux.HandleFunc("/ws", corsMiddleware(server.HandleWebSocket))

	log.Printf("Server starting on :%s with root: %s", port, rootDir)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
