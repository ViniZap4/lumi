// server/http/handlers.go
package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/vinizap/lumi/server/domain"
	"github.com/vinizap/lumi/server/filesystem"
	"github.com/vinizap/lumi/server/ws"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Server struct {
	rootDir string
	hub     *ws.Hub
}

func NewServer(rootDir string, hub *ws.Hub) *Server {
	return &Server{rootDir: rootDir, hub: hub}
}

func (s *Server) HandleFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := filesystem.ListFolders(s.rootDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

func (s *Server) HandleNotes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = s.rootDir
	} else {
		path = filepath.Join(s.rootDir, path)
	}

	notes, err := filesystem.ListNotes(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func (s *Server) HandleGetNote(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	if id == "" {
		http.Error(w, "Note ID required", http.StatusBadRequest)
		return
	}

	notePath := filepath.Join(s.rootDir, id+".md")
	note, err := filesystem.ReadNote(notePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (s *Server) HandleCreateNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string   `json:"id"`
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	note, err := filesystem.CreateNote(s.rootDir, req.ID, req.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	note.Content = req.Content
	note.Tags = req.Tags

	if err := filesystem.WriteNote(note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("note_created", note)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func (s *Server) HandleUpdateNote(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	if id == "" {
		http.Error(w, "Note ID required", http.StatusBadRequest)
		return
	}

	notePath := filepath.Join(s.rootDir, id+".md")
	note, err := filesystem.ReadNote(notePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var req struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	note.Title = req.Title
	note.Content = req.Content
	note.Tags = req.Tags

	if err := filesystem.WriteNote(note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("note_updated", note)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (s *Server) HandleDeleteNote(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	if id == "" {
		http.Error(w, "Note ID required", http.StatusBadRequest)
		return
	}

	notePath := filepath.Join(s.rootDir, id+".md")
	note, err := filesystem.ReadNote(notePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err := filesystem.DeleteNote(notePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("note_deleted", note)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.hub.Register(conn)
	s.hub.HandleConnection(conn)
}
