package image

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var imageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

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

func Render(imagePath string, width int) string {
	_, err := os.Stat(imagePath)
	if err != nil {
		return fmt.Sprintf("[Image error: %s]", err)
	}
	
	// Try timg for inline rendering (works in iTerm2/Kitty/tmux)
	if output, err := exec.Command("timg", "-g", fmt.Sprintf("%dx20", width), imagePath).Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	
	// Try chafa for Unicode/ASCII art
	if output, err := exec.Command("chafa", "-s", fmt.Sprintf("%dx20", width), imagePath).Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	
	// Try viu
	if output, err := exec.Command("viu", "-w", fmt.Sprintf("%d", width), imagePath).Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	
	return fmt.Sprintf("🖼️  [Image: %s]\n   Install 'timg', 'chafa', or 'viu' for inline preview", filepath.Base(imagePath))
}

func renderITerm2(imagePath string, inTmux bool) string {
	return fmt.Sprintf("🖼️  [Image: %s]", filepath.Base(imagePath))
}

func renderKitty(imagePath string, inTmux bool) string {
	return fmt.Sprintf("🖼️  [Image: %s]", filepath.Base(imagePath))
}
