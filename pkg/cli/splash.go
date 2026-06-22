package cli

import (
	"fmt"
	"strings"

	"golang.org/x/term"
)

var logoLines = []string{
	"                 ▄",
	"█▀▀▄ ▐▌ █▀▀▄ █▀▀▀▄",
	"█▄▄▀ ▐▌ █▄▄▀ █▄▄▄▀",
	"▀  ▀  ▀ ▀▀▀  ▀  ▀▀",
}

var logoColors = []string{
	"\033[38;5;51m",
	"\033[38;5;45m",
	"\033[38;5;75m",
	"\033[38;5;117m",
}

const colorReset = "\033[0m"
const colorDim = "\033[38;5;241m"
const colorVersion = "\033[38;5;245m"
const colorTagline = "\033[38;5;87m"

func DetectTermSize() (width, height int) {
	width = 80
	height = 24
	if !term.IsTerminal(0) {
		return
	}
	w, h, err := term.GetSize(0)
	if err == nil && w > 0 && h > 0 {
		width = w
		height = h
	}
	return
}

func RenderSplash(width, height int, version string) string {
	if width < 50 || height < 10 {
		return RenderSplashMinimal(width, version)
	}

	versionDisplay := "v" + version
	if version == "dev" || version == "" {
		versionDisplay = ""
	}

	maxLineWidth := len(logoLines[0])
	for _, line := range logoLines {
		if len(line) > maxLineWidth {
			maxLineWidth = len(line)
		}
	}

	artHeight := len(logoLines) + 2
	vertPad := (height - artHeight) / 2
	if vertPad < 1 {
		vertPad = 1
	}
	horizPad := (width - maxLineWidth) / 2
	if horizPad < 0 {
		horizPad = 0
	}

	var buf strings.Builder

	for i := 0; i < vertPad; i++ {
		buf.WriteString("\n")
	}

	for i, line := range logoLines {
		color := logoColors[i]
		if i >= len(logoColors) {
			color = logoColors[len(logoColors)-1]
		}
		buf.WriteString(strings.Repeat(" ", horizPad))
		buf.WriteString(color)
		buf.WriteString(line)
		buf.WriteString(colorReset)
		buf.WriteString("\n")
	}

	buf.WriteString("\n")

	tagline := "k13d cli"
	taglineOffset := horizPad + (maxLineWidth-len(tagline))/2
	if taglineOffset < 0 {
		taglineOffset = 0
	}
	buf.WriteString(strings.Repeat(" ", taglineOffset))
	buf.WriteString(colorTagline)
	buf.WriteString(tagline)
	buf.WriteString(colorReset)
	buf.WriteString("\n")

	if versionDisplay != "" {
		versionOffset := horizPad + maxLineWidth - len(versionDisplay)
		if versionOffset > horizPad {
			buf.WriteString(strings.Repeat(" ", versionOffset))
			buf.WriteString(colorVersion)
			buf.WriteString(versionDisplay)
			buf.WriteString(colorReset)
			buf.WriteString("\n")
		}
	}

	for i := 0; i < vertPad-1; i++ {
		buf.WriteString("\n")
	}

	return buf.String()
}

func RenderSplashMinimal(width int, version string) string {
	title := "k13d CLI"
	padding := (width - len(title)) / 2
	if padding < 0 {
		padding = 0
	}
	line := fmt.Sprintf("%s%s%s%s",
		colorTagline,
		strings.Repeat(" ", padding)+title,
		colorReset,
		colorDim+" ["+version+"]"+colorReset,
	)
	return fmt.Sprintf("\n%s\n%s\n\n",
		line,
		strings.Repeat(" ", padding)+strings.Repeat("=", len(title)),
	)
}

func ClearScreen() string {
	return "\033[2J\033[H"
}

func RenderPrompt() string {
	return "\033[38;5;45m▶\033[0m "
}

func PrintSplash(version string) {
	w, h := DetectTermSize()
	fmt.Print(ClearScreen())
	fmt.Print(RenderSplash(w, h, version))
}
