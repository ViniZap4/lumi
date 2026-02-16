// tui-client/editor/editor.go
package editor

import (
	"os"
	"os/exec"

	"github.com/vinizap/lumi/tui-client/domain"
)

func OpenCmd(note *domain.Note) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	cmd := exec.Command(editor, note.Path)
	return cmd
}
