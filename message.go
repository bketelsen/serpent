// https://github.com/coder/coder/blob/main/LICENSE
// Extracted and modified from github.com/coder/coder
package serpent

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// cliMessage provides a human-readable message for CLI errors and messages.
type cliMessage struct {
	Style  lipgloss.Style
	Header string
	Prefix string
	Lines  []string
}

// String formats the CLI message for consumption by a human.
func (m cliMessage) String() string {
	var str strings.Builder

	if m.Prefix != "" {
		_, _ = str.WriteString(Bold(m.Prefix))
	}

	str.WriteString(m.Style.Render(m.Header))
	_, _ = str.WriteString("\r\n")
	for _, line := range m.Lines {
		_, _ = fmt.Fprintf(&str, "  %s %s\r\n", m.Style.Render("|"), line)
	}
	return str.String()
}

// Warn writes a log to the writer provided.
func (inv *Invocation) Warn(header string, lines ...string) {
	_, _ = fmt.Fprint(inv.Stderr, cliMessage{
		Style:  DefaultStyles.Warn,
		Prefix: "WARNING: ",
		Header: header,
		Lines:  lines,
	}.String())
}

// Info writes a log to the writer provided.
func (inv *Invocation) Info(header string, lines ...string) {
	_, _ = fmt.Fprint(inv.Stderr, cliMessage{
		Header: header,
		Lines:  lines,
	}.String())
}

// Error writes a log to the writer provided.
func (inv *Invocation) Error(header string, lines ...string) {
	_, _ = fmt.Fprint(inv.Stderr, cliMessage{
		Style:  DefaultStyles.Error,
		Prefix: "ERROR: ",
		Header: header,
		Lines:  lines,
	}.String())
}
