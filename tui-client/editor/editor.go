// tui-client/editor/editor.go
package editor

import (
	"os"
	"os/exec"

	"github.com/vinizap/lumi/tui-client/config"
	"github.com/vinizap/lumi/tui-client/domain"
)

func OpenCmd(note *domain.Note) *exec.Cmd {
	cfg := config.Load()
	
	// Build args: editor args + note path
	args := append(cfg.EditorArgs, note.Path)
	
	// Check if EDITOR env var is set, use it instead
	if envEditor := os.Getenv("EDITOR"); envEditor != "" {
		cfg.Editor = envEditor
		args = []string{note.Path}
	}
	
	cmd := exec.Command(cfg.Editor, args...)
	return cmd
}
