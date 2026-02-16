// tui-client/ui/home.go
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/domain"
)

const asciiArt = `
  ██╗     ██╗   ██╗███╗   ███╗██╗
  ██║     ██║   ██║████╗ ████║██║
  ██║     ██║   ██║██╔████╔██║██║
  ██║     ██║   ██║██║╚██╔╝██║██║
  ███████╗╚██████╔╝██║ ╚═╝ ██║██║
  ╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚═╝
`

func (m Model) renderHome() string {
	var content strings.Builder

	// ASCII art centered
	artStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		MarginTop(2).
		MarginBottom(2)
	content.WriteString(artStyle.Render(asciiArt))
	content.WriteString("\n")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(m.width).
		Align(lipgloss.Center).
		MarginBottom(3)
	content.WriteString(subtitleStyle.Render("Local-first Markdown notes with vim motions"))
	content.WriteString("\n\n")

	// Recent notes section
	recentTitle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		MarginBottom(1).
		Render("📝 Recent Notes")
	content.WriteString(recentTitle)
	content.WriteString("\n\n")

	// Show recent notes (last 5)
	recentNotes := m.getRecentNotes(5)
	if len(recentNotes) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(m.width).
			Align(lipgloss.Center)
		content.WriteString(emptyStyle.Render("No notes yet. Press 't' to browse or 'n' to create."))
	} else {
		for i, note := range recentNotes {
			noteStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Width(m.width).
				Align(lipgloss.Center)
			
			timeAgo := formatTimeAgo(note.UpdatedAt)
			line := fmt.Sprintf("%d. %s  •  %s", i+1, note.Title, timeAgo)
			content.WriteString(noteStyle.Render(line))
			content.WriteString("\n")
		}
	}

	content.WriteString("\n\n")

	// Commands section
	commandsTitle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		MarginBottom(1).
		Render("⌨️  Commands")
	content.WriteString(commandsTitle)
	content.WriteString("\n\n")

	commands := []string{
		HelpKeyStyle.Render("t") + " - Open tree browser",
		HelpKeyStyle.Render("n") + " - Create new note",
		HelpKeyStyle.Render("q") + " - Quit",
	}

	for _, cmd := range commands {
		cmdStyle := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center)
		content.WriteString(cmdStyle.Render(cmd))
		content.WriteString("\n")
	}

	return content.String()
}

func (m Model) getRecentNotes(limit int) []*domain.Note {
	var recent []*domain.Note
	
	for _, item := range m.notes {
		recent = append(recent, item.note)
	}

	// Sort by UpdatedAt (most recent first)
	for i := 0; i < len(recent)-1; i++ {
		for j := i + 1; j < len(recent); j++ {
			if recent[j].UpdatedAt.After(recent[i].UpdatedAt) {
				recent[i], recent[j] = recent[j], recent[i]
			}
		}
	}

	if len(recent) > limit {
		recent = recent[:limit]
	}

	return recent
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	} else {
		return t.Format("Jan 2")
	}
}
