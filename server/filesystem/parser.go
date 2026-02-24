// tui-client/filesystem/parser.go
package filesystem

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vinizap/lumi/server/domain"
	"gopkg.in/yaml.v3"
)

func ReadNote(path string) (*domain.Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	note := &domain.Note{Path: path}
	
	// Split frontmatter and content
	parts := bytes.SplitN(data, []byte("---"), 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid frontmatter format")
	}

	// Parse frontmatter
	if err := yaml.Unmarshal(parts[1], note); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Store content
	note.Content = string(bytes.TrimSpace(parts[2]))
	
	return note, nil
}

func WriteNote(note *domain.Note) error {
	var buf bytes.Buffer
	
	buf.WriteString("---\n")
	
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(note); err != nil {
		return fmt.Errorf("failed to encode frontmatter: %w", err)
	}
	encoder.Close()
	
	buf.WriteString("---\n\n")
	buf.WriteString(note.Content)
	
	return os.WriteFile(note.Path, buf.Bytes(), 0644)
}

func ListNotes(dir string) ([]*domain.Note, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var notes []*domain.Note
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		
		path := filepath.Join(dir, entry.Name())
		note, err := ReadNote(path)
		if err != nil {
			continue // Skip invalid notes
		}
		notes = append(notes, note)
	}
	
	return notes, nil
}

func ListFolders(root string) ([]*domain.Folder, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var folders []*domain.Folder
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		folders = append(folders, &domain.Folder{
			Name: entry.Name(),
			Path: entry.Name(),
		})
	}

	return folders, nil
}

// FindNotePath searches recursively under rootDir for a markdown file
// whose frontmatter ID matches the given id. Returns the full file path.
func FindNotePath(rootDir, id string) (string, error) {
	var found string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		note, err := ReadNote(path)
		if err == nil && note.ID == id {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("note not found: %s", id)
	}
	return found, nil
}

// MoveNote moves a note file to a different directory.
func MoveNote(oldPath, newDir string) (*domain.Note, error) {
	newPath := filepath.Join(newDir, filepath.Base(oldPath))
	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("failed to move note: %w", err)
	}
	note, err := ReadNote(newPath)
	if err != nil {
		return nil, err
	}
	note.Path = newPath
	note.UpdatedAt = time.Now()
	if err := WriteNote(note); err != nil {
		return nil, err
	}
	return note, nil
}

// CopyNote duplicates a note with a new ID and title into destDir.
func CopyNote(srcPath, destDir, newID, newTitle string) (*domain.Note, error) {
	src, err := ReadNote(srcPath)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	note := &domain.Note{
		ID:        newID,
		Title:     newTitle,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      src.Tags,
		Path:      filepath.Join(destDir, newID+".md"),
		Content:   src.Content,
	}
	if err := WriteNote(note); err != nil {
		return nil, err
	}
	return note, nil
}

// RenameNote changes a note's ID and title, moving the file accordingly.
func RenameNote(oldPath, newID, newTitle string) (*domain.Note, error) {
	note, err := ReadNote(oldPath)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, newID+".md")
	note.ID = newID
	note.Title = newTitle
	note.Path = newPath
	note.UpdatedAt = time.Now()
	if err := WriteNote(note); err != nil {
		return nil, err
	}
	if oldPath != newPath {
		os.Remove(oldPath)
	}
	return note, nil
}
