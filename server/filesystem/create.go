// tui-client/filesystem/create.go
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vinizap/lumi/server/domain"
)

func CreateNote(dir, id, title string) (*domain.Note, error) {
	now := time.Now()
	
	note := &domain.Note{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{},
		Path:      filepath.Join(dir, id+".md"),
		Content:   fmt.Sprintf("# %s\n\n", title),
	}

	if err := WriteNote(note); err != nil {
		return nil, err
	}

	return note, nil
}

func DeleteNote(path string) error {
	return os.Remove(path)
}

func CreateFolder(rootDir, name string) error {
	return os.MkdirAll(filepath.Join(rootDir, name), 0755)
}
