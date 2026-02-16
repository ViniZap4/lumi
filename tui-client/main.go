// tui-client/main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vinizap/lumi/tui-client/ui"
)

func main() {
	rootDir := os.Getenv("LUMI_NOTES_DIR")
	if rootDir == "" {
		rootDir = "."
	}

	p := tea.NewProgram(
		ui.NewModel(rootDir),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
