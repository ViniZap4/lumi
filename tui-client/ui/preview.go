// tui-client/ui/preview.go
package ui

import (
	"regexp"
	"strings"

	"github.com/vinizap/lumi/tui-client/domain"
)

type PreviewMode int

const (
	PreviewOff PreviewMode = iota
	PreviewPartial
	PreviewFull
)

func renderPreview(note *domain.Note, mode PreviewMode, width, height int) string {
	if note == nil || mode == PreviewOff {
		return DimItemStyle.Render("No note selected")
	}

	var content strings.Builder

	// Title
	content.WriteString(PreviewTitleStyle.Render(note.Title))
	content.WriteString("\n\n")

	// Metadata
	meta := PreviewMetaStyle.Render(
		"ID: " + note.ID + " | " +
			"Created: " + note.CreatedAt.Format("2006-01-02") + " | " +
			"Tags: " + strings.Join(note.Tags, ", "),
	)
	content.WriteString(meta)
	content.WriteString("\n\n")

	// Content
	noteContent := note.Content
	if mode == PreviewPartial {
		lines := strings.Split(noteContent, "\n")
		maxLines := height - 10
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			noteContent = strings.Join(lines, "\n") + "\n\n" + DimItemStyle.Render("... (press v for full view)")
		} else {
			noteContent = strings.Join(lines, "\n")
		}
	}

	// Highlight links
	noteContent = highlightLinks(noteContent)

	content.WriteString(PreviewContentStyle.Width(width - 4).Render(noteContent))

	return content.String()
}

func highlightLinks(content string) string {
	// Highlight [[wiki-links]]
	wikiLinkRe := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	content = wikiLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		return PreviewLinkStyle.Render(match)
	})

	// Highlight [markdown](links)
	mdLinkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	content = mdLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		return PreviewLinkStyle.Render(match)
	})

	return content
}

func extractLinks(content string) []string {
	var links []string

	// Extract [[wiki-links]]
	wikiLinkRe := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches := wikiLinkRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			links = append(links, match[1])
		}
	}

	// Extract [markdown](links)
	mdLinkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	matches = mdLinkRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 2 {
			links = append(links, match[2])
		}
	}

	return links
}
