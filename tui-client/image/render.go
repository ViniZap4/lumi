package image

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var imageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// Cache which renderer is available to avoid repeated exec.LookPath calls.
var (
	rendererOnce sync.Once
	rendererName string // "kitty", "iterm2", "timg", "chafa", "viu", ""
)

func detectRenderer() {
	rendererOnce.Do(func() {
		// Check terminal protocol support
		term := os.Getenv("TERM")
		termProgram := os.Getenv("TERM_PROGRAM")
		kittyWindowID := os.Getenv("KITTY_WINDOW_ID")

		if kittyWindowID != "" || strings.Contains(term, "kitty") {
			rendererName = "kitty"
			return
		}
		if termProgram == "iTerm.app" || termProgram == "WezTerm" {
			rendererName = "iterm2"
			return
		}

		// Fall back to external tools
		for _, tool := range []string{"timg", "chafa", "viu"} {
			if _, err := exec.LookPath(tool); err == nil {
				rendererName = tool
				return
			}
		}
	})
}

func HasImage(line string) bool {
	return imageRegex.MatchString(line)
}

func ExtractImagePath(line string) string {
	matches := imageRegex.FindStringSubmatch(line)
	if len(matches) > 2 {
		return matches[2]
	}
	return ""
}

func GetImagePath(line, notePath string) string {
	imgPath := ExtractImagePath(line)
	if imgPath == "" {
		return ""
	}
	if !filepath.IsAbs(imgPath) {
		imgPath = filepath.Join(filepath.Dir(notePath), imgPath)
	}
	return imgPath
}

// Render renders an image for terminal display at the given width.
// Height is computed proportionally, capped at a reasonable max.
func Render(imagePath string, width int) string {
	if _, err := os.Stat(imagePath); err != nil {
		return fmt.Sprintf("[Image not found: %s]", filepath.Base(imagePath))
	}

	if width < 10 {
		width = 10
	}

	// Cap height to keep images from dominating the view
	height := width / 3
	if height < 8 {
		height = 8
	}
	if height > 30 {
		height = 30
	}

	detectRenderer()

	switch rendererName {
	case "kitty":
		return renderKittyProtocol(imagePath)
	case "iterm2":
		return renderITerm2Protocol(imagePath)
	case "timg":
		if out, err := exec.Command("timg", "-g", fmt.Sprintf("%dx%d", width, height), imagePath).Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	case "chafa":
		if out, err := exec.Command("chafa", "-s", fmt.Sprintf("%dx%d", width, height), "--animate=off", imagePath).Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	case "viu":
		if out, err := exec.Command("viu", "-w", fmt.Sprintf("%d", width), "-h", fmt.Sprintf("%d", height), imagePath).Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}

	return fmt.Sprintf("[Image: %s] (install timg, chafa, or viu for preview)", filepath.Base(imagePath))
}

// renderKittyProtocol uses the Kitty graphics protocol to display images inline.
func renderKittyProtocol(imagePath string) string {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Sprintf("[Image error: %s]", filepath.Base(imagePath))
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	// Kitty graphics protocol: transmit image in chunks
	var sb strings.Builder
	const chunkSize = 4096

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		more := 1
		if end >= len(encoded) {
			end = len(encoded)
			more = 0
		}
		chunk := encoded[i:end]

		if i == 0 {
			// First chunk: include format and action
			sb.WriteString(fmt.Sprintf("\033_Ga=T,f=100,m=%d;%s\033\\", more, chunk))
		} else {
			sb.WriteString(fmt.Sprintf("\033_Gm=%d;%s\033\\", more, chunk))
		}
	}

	return sb.String()
}

// renderITerm2Protocol uses the iTerm2 inline image protocol.
func renderITerm2Protocol(imagePath string) string {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Sprintf("[Image error: %s]", filepath.Base(imagePath))
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	name := base64.StdEncoding.EncodeToString([]byte(filepath.Base(imagePath)))

	return fmt.Sprintf("\033]1337;File=name=%s;inline=1;preserveAspectRatio=1:%s\a",
		name, encoded)
}
