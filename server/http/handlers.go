// server/http/handlers.go
package http

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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

	notePath, err := filesystem.FindNotePath(s.rootDir, id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	note, err := filesystem.ReadNote(notePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		Folder  string   `json:"folder"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dir := s.rootDir
	if req.Folder != "" {
		dir = filepath.Join(s.rootDir, req.Folder)
	}

	note, err := filesystem.CreateNote(dir, req.ID, req.Title)
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

	notePath, err := filesystem.FindNotePath(s.rootDir, id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	note, err := filesystem.ReadNote(notePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	note.UpdatedAt = time.Now()

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

	notePath, err := filesystem.FindNotePath(s.rootDir, id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	note, err := filesystem.ReadNote(notePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := filesystem.DeleteNote(notePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("note_deleted", note)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) HandleMoveNote(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	id := strings.TrimSuffix(path, "/move")

	var req struct {
		Folder string `json:"folder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notePath, err := filesystem.FindNotePath(s.rootDir, id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	destDir := s.rootDir
	if req.Folder != "" {
		destDir = filepath.Join(s.rootDir, req.Folder)
	}

	note, err := filesystem.MoveNote(notePath, destDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("note_updated", note)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (s *Server) HandleCopyNote(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	id := strings.TrimSuffix(path, "/copy")

	var req struct {
		NewID    string `json:"new_id"`
		NewTitle string `json:"new_title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notePath, err := filesystem.FindNotePath(s.rootDir, id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	destDir := filepath.Dir(notePath)
	note, err := filesystem.CopyNote(notePath, destDir, req.NewID, req.NewTitle)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("note_created", note)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func (s *Server) HandleRenameNote(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/notes/")
	id := strings.TrimSuffix(path, "/rename")

	var req struct {
		NewID    string `json:"new_id"`
		NewTitle string `json:"new_title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notePath, err := filesystem.FindNotePath(s.rootDir, id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	oldNote, err := filesystem.ReadNote(notePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	note, err := filesystem.RenameNote(notePath, req.NewID, req.NewTitle)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Broadcast("note_deleted", oldNote)
	s.hub.Broadcast("note_created", note)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (s *Server) HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Folder name required", http.StatusBadRequest)
		return
	}

	if err := filesystem.CreateFolder(s.rootDir, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&domain.Folder{Name: req.Name, Path: req.Name})
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.hub.Register(conn)
	s.hub.HandleConnection(conn)
}
