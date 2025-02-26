// https://github.com/coder/coder/blob/main/LICENSE
// Extracted and modified from github.com/coder/coder
package ui

import (
	"fmt"
	"strings"

	"github.com/coder/pretty"
)

// cliMessage provides a human-readable message for CLI errors and messages.
type cliMessage struct {
	Style  pretty.Style
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

	pretty.Fprint(&str, m.Style, m.Header)
	_, _ = str.WriteString("\r\n")
	for _, line := range m.Lines {
		_, _ = fmt.Fprintf(&str, "  %s %s\r\n", pretty.Sprint(m.Style, "|"), line)
	}
	return str.String()
}
