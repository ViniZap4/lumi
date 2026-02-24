package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinizap/lumi/tui-client/theme"
)

// Inline class constants for per-character styling within a line.
const (
	clsNormal = iota
	clsBold
	clsItalic
	clsBoldItalic
	clsCode
	clsLinkText
	clsDim // delimiters, URLs, brackets
	clsListMarker
	clsStrike
)

// shouldClassifyInline returns true if the line should receive inline highlighting.
// Headings, blockquotes, separators, and code block lines are styled at the line
// level and skip inline classification.
func shouldClassifyInline(line string, inCode bool) bool {
	if inCode {
		return false
	}
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "# "),
		strings.HasPrefix(trimmed, "## "),
		strings.HasPrefix(trimmed, "### "),
		strings.HasPrefix(trimmed, "#### "),
		strings.HasPrefix(trimmed, "##### "),
		strings.HasPrefix(trimmed, "###### "):
		return false
	case strings.HasPrefix(trimmed, "> "):
		return false
	case trimmed == "---" || trimmed == "***" || trimmed == "___":
		return false
	}
	// Table lines get inline classification for pipe delimiters
	return true
}

// classifyInline returns a per-rune inline class array for a line.
// Detection order (higher priority first): list markers, code spans,
// bold-italic, bold, italic, strikethrough, wikilinks, standard links.
// Delimiter characters (**, `, [, ], etc.) are classified as clsDim.
func classifyInline(line string) []int {
	runes := []rune(line)
	n := len(runes)
	if n == 0 {
		return nil
	}
	cls := make([]int, n)
	used := make([]bool, n)

	// --- Table lines ---
	trimmedLine := strings.TrimSpace(string(runes))
	if strings.HasPrefix(trimmedLine, "|") && strings.HasSuffix(trimmedLine, "|") && n > 2 {
		// Check if separator row (| --- | --- |)
		isSepRow := true
		inner := trimmedLine[1 : len(trimmedLine)-1]
		for _, cell := range strings.Split(inner, "|") {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}
			cleaned := strings.Trim(cell, ":-")
			if cleaned != "" {
				isSepRow = false
				break
			}
		}
		if isSepRow {
			// Entire separator row is dim
			for i := range cls {
				cls[i] = clsDim
			}
			return cls
		}
		// Regular table row: style pipe characters as dim
		for i, r := range runes {
			if r == '|' {
				cls[i] = clsDim
				used[i] = true
			}
		}
	}

	// --- List markers at line start ---
	ls := 0
	for ls < n && (runes[ls] == ' ' || runes[ls] == '\t') {
		ls++
	}
	if ls < n {
		rest := runes[ls:]
		ml := 0
		if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '+') && rest[1] == ' ' {
			ml = 2
		} else if len(rest) >= 2 && rest[0] == '*' && rest[1] == ' ' {
			ml = 2
		} else if len(rest) >= 3 && rest[0] >= '0' && rest[0] <= '9' {
			j := 1
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			if j < len(rest)-1 && rest[j] == '.' && rest[j+1] == ' ' {
				ml = j + 2
			}
		}
		for k := ls; k < ls+ml && k < n; k++ {
			cls[k] = clsListMarker
			used[k] = true
		}
	}

	// --- Code spans (`...`) ---
	for i := 0; i < n; {
		if used[i] {
			i++
			continue
		}
		if runes[i] == '`' {
			j := i + 1
			for j < n && runes[j] != '`' {
				j++
			}
			if j < n && j > i+1 {
				cls[i] = clsDim
				cls[j] = clsDim
				used[i] = true
				used[j] = true
				for k := i + 1; k < j; k++ {
					cls[k] = clsCode
					used[k] = true
				}
				i = j + 1
				continue
			}
		}
		i++
	}

	// --- Bold italic (***...***) ---
	for i := 0; i < n; {
		if used[i] {
			i++
			continue
		}
		if i+2 < n && runes[i] == '*' && runes[i+1] == '*' && runes[i+2] == '*' {
			j := i + 3
			for j+2 < n {
				if !used[j] && runes[j] == '*' && runes[j+1] == '*' && runes[j+2] == '*' {
					break
				}
				j++
			}
			if j+2 < n && j > i+3 {
				for k := i; k < i+3; k++ {
					cls[k] = clsDim
					used[k] = true
				}
				for k := j; k < j+3; k++ {
					cls[k] = clsDim
					used[k] = true
				}
				for k := i + 3; k < j; k++ {
					if !used[k] {
						cls[k] = clsBoldItalic
						used[k] = true
					}
				}
				i = j + 3
				continue
			}
		}
		i++
	}

	// --- Bold (**...**) ---
	for i := 0; i < n; {
		if used[i] {
			i++
			continue
		}
		if i+1 < n && runes[i] == '*' && runes[i+1] == '*' {
			if i+2 < n && runes[i+2] == '*' {
				i++
				continue
			}
			j := i + 2
			for j+1 < n {
				if !used[j] && runes[j] == '*' && runes[j+1] == '*' {
					if j+2 < n && runes[j+2] == '*' {
						j++
						continue
					}
					break
				}
				j++
			}
			if j+1 < n && j > i+2 {
				cls[i] = clsDim
				cls[i+1] = clsDim
				used[i] = true
				used[i+1] = true
				cls[j] = clsDim
				cls[j+1] = clsDim
				used[j] = true
				used[j+1] = true
				for k := i + 2; k < j; k++ {
					if !used[k] {
						cls[k] = clsBold
						used[k] = true
					}
				}
				i = j + 2
				continue
			}
		}
		i++
	}

	// --- Italic (*...*) ---
	for i := 0; i < n; {
		if used[i] {
			i++
			continue
		}
		if runes[i] == '*' {
			if i+1 < n && runes[i+1] == '*' {
				i++
				continue
			}
			j := i + 1
			for j < n {
				if !used[j] && runes[j] == '*' && (j+1 >= n || runes[j+1] != '*') {
					break
				}
				j++
			}
			if j < n && j > i+1 {
				cls[i] = clsDim
				cls[j] = clsDim
				used[i] = true
				used[j] = true
				for k := i + 1; k < j; k++ {
					if !used[k] {
						cls[k] = clsItalic
						used[k] = true
					}
				}
				i = j + 1
				continue
			}
		}
		i++
	}

	// --- Strikethrough (~~...~~) ---
	for i := 0; i < n; {
		if used[i] {
			i++
			continue
		}
		if i+1 < n && runes[i] == '~' && runes[i+1] == '~' {
			j := i + 2
			for j+1 < n {
				if !used[j] && runes[j] == '~' && runes[j+1] == '~' {
					break
				}
				j++
			}
			if j+1 < n && j > i+2 {
				cls[i] = clsDim
				cls[i+1] = clsDim
				used[i] = true
				used[i+1] = true
				cls[j] = clsDim
				cls[j+1] = clsDim
				used[j] = true
				used[j+1] = true
				for k := i + 2; k < j; k++ {
					if !used[k] {
						cls[k] = clsStrike
						used[k] = true
					}
				}
				i = j + 2
				continue
			}
		}
		i++
	}

	// --- Wikilinks ([[...]]) ---
	for i := 0; i < n; {
		if used[i] {
			i++
			continue
		}
		if i+1 < n && runes[i] == '[' && runes[i+1] == '[' {
			j := i + 2
			for j+1 < n {
				if runes[j] == ']' && runes[j+1] == ']' {
					break
				}
				j++
			}
			if j+1 < n && j > i+2 {
				cls[i] = clsDim
				cls[i+1] = clsDim
				used[i] = true
				used[i+1] = true
				cls[j] = clsDim
				cls[j+1] = clsDim
				used[j] = true
				used[j+1] = true
				for k := i + 2; k < j; k++ {
					if !used[k] {
						cls[k] = clsLinkText
						used[k] = true
					}
				}
				i = j + 2
				continue
			}
		}
		i++
	}

	// --- Standard links ([text](url)) ---
	for i := 0; i < n; {
		if used[i] {
			i++
			continue
		}
		if runes[i] == '[' {
			j := i + 1
			for j < n && runes[j] != ']' {
				j++
			}
			if j < n && j+1 < n && runes[j+1] == '(' {
				k := j + 2
				for k < n && runes[k] != ')' {
					k++
				}
				if k < n && j > i+1 {
					cls[i] = clsDim
					used[i] = true
					for t := i + 1; t < j; t++ {
						if !used[t] {
							cls[t] = clsLinkText
							used[t] = true
						}
					}
					for t := j; t <= k; t++ {
						if !used[t] {
							cls[t] = clsDim
							used[t] = true
						}
					}
					i = k + 1
					continue
				}
			}
		}
		i++
	}

	return cls
}

// resolveInlineStyle maps an inline class to a lipgloss style, using baseStyle
// as the foundation for bold/italic (so they inherit the line-level foreground).
func resolveInlineStyle(cls int, baseStyle lipgloss.Style) lipgloss.Style {
	t := theme.Current
	switch cls {
	case clsBold:
		return baseStyle.Bold(true)
	case clsItalic:
		return baseStyle.Italic(true)
	case clsBoldItalic:
		return baseStyle.Bold(true).Italic(true)
	case clsCode:
		return lipgloss.NewStyle().Foreground(t.Accent)
	case clsLinkText:
		return lipgloss.NewStyle().Foreground(t.Info).Underline(true)
	case clsDim:
		return lipgloss.NewStyle().Foreground(t.TextDim)
	case clsListMarker:
		return lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	case clsStrike:
		return lipgloss.NewStyle().Foreground(t.TextDim).Strikethrough(true)
	default:
		return baseStyle
	}
}
