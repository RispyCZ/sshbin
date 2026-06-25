package sftpd

import (
	"io"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"golang.org/x/crypto/ssh"
)

type StderrWriter struct {
	ch ssh.Channel
}

func NewStderrWriter(ch ssh.Channel) *StderrWriter {
	return &StderrWriter{ch: ch}
}

func (w *StderrWriter) Write(p []byte) (int, error) {
	return w.ch.Stderr().Write(p)
}

// bannerRenderer forces a color profile because the SSH channel is not a TTY;
// lipgloss would otherwise auto-detect "no color" and strip all styling.
var bannerRenderer = func() *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	return r
}()

var (
	bannerBox = bannerRenderer.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("6")).
			Padding(0, 1)
	bannerTitle  = bannerRenderer.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	bannerStatus = bannerRenderer.NewStyle().Foreground(lipgloss.Color("2"))
	bannerLink   = bannerRenderer.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
)

// setupBanner renders the post-upload notice as a colored box. Output goes to
// the client's stderr, which the local terminal renders with ANSI escapes.
// Line endings are CRLF since the SSH channel has no PTY to translate them.
func setupBanner(fileName, url string) string {
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		bannerTitle.Render("Sshbin"),
		bannerStatus.Render(sanitizeName(fileName)+" uploaded. Finish setup:"),
		bannerLink.Render(url),
	)
	out := "\n" + bannerBox.Render(content) + "\n"
	return strings.ReplaceAll(out, "\n", "\r\n")
}

// sanitizeName strips ANSI escape sequences and other control characters from a
// client-supplied filename so it can't spoof or break the rendered banner.
func sanitizeName(name string) string {
	name = ansi.Strip(name)
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
}
