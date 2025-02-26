package serpent

import (
	"fmt"
	"strings"

	"github.com/mitchellh/go-wordwrap"
)

type Example struct {
	Description string
	Command     string
}

// FormatExamples formats the examples as width wrapped bulletpoint
// descriptions with the command underneath.
func FormatExamples(examples ...Example) string {
	var sb strings.Builder

	padStyle := DefaultStyles.Wrap
	for i, e := range examples {
		if len(e.Description) > 0 {
			wordwrap.WrapString(e.Description, 80)
			_, _ = sb.WriteString(
				"  - " + padStyle.Render(e.Description+":") + "\n\n    ",
			)
		}
		// We add 1 space here because `cliui.DefaultStyles.Code` adds an extra
		// space. This makes the code block align at an even 2 or 6
		// spaces for symmetry.
		_, _ = sb.WriteString(" " + DefaultStyles.Code.Render(fmt.Sprintf("$ %s", e.Command)))
		if i < len(examples)-1 {
			_, _ = sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

func Long(description string, examples ...Example) string {
	var sb strings.Builder

	padStyle := DefaultStyles.Wrap
	if len(description) > 0 {
		wordwrap.WrapString(description, 80)
		_, _ = sb.WriteString(
			padStyle.Render(description) + "\n\n",
		)
	}
	sb.WriteString(FormatExamples(examples...))
	return sb.String()

}
