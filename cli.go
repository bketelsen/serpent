// https://github.com/coder/coder/blob/main/LICENSE
// Extracted and modified from github.com/coder/coder
package serpent

import (
	"errors"
	"sync"
	"time"

	"github.com/muesli/termenv"

	"github.com/charmbracelet/lipgloss"
)

var Canceled = errors.New("canceled")

// DefaultStyles compose visual elements of the UI.
var DefaultStyles Styles

var (
	BoldStyle = lipgloss.NewStyle().Bold(true)
)

type Styles struct {
	Code,
	DateTimeStamp,
	Error,
	Field,
	Hyperlink,
	Keyword,
	Placeholder,
	Prompt,
	FocusedPrompt,
	Fuchsia,
	Warn,
	Wrap lipgloss.Style
}

var (
	color     termenv.Profile
	colorOnce sync.Once
)

var (
	// ANSI color codes
	red           = lipgloss.Color("1")
	green         = lipgloss.Color("2")
	yellow        = lipgloss.Color("3")
	magenta       = lipgloss.Color("5")
	white         = lipgloss.Color("7")
	brightBlue    = lipgloss.Color("12")
	brightMagenta = lipgloss.Color("13")
)

func isTerm() bool {
	return color != termenv.Ascii
}

// Bold returns a formatter that renders text in bold
// if the terminal supports it.
func Bold(s string) string {
	if !isTerm() {
		return s
	}
	return BoldStyle.Render(s)
}

// Timestamp formats a timestamp for display.
func Timestamp(t time.Time) string {
	return DefaultStyles.DateTimeStamp.Render(t.Format(time.Stamp))
}

// Keyword formats a keyword for display.
func Keyword(s string) string {
	return DefaultStyles.Keyword.Render(s)
}

// Placeholder formats a placeholder for display.
func Placeholder(s string) string {
	return DefaultStyles.Placeholder.Render(s)
}

// Wrap prevents the text from overflowing the terminal.
func Wrap(s string) string {
	return DefaultStyles.Wrap.Render(s)
}

// Code formats code for display.
func Code(s string) string {
	return DefaultStyles.Code.Render(s)
}

// Field formats a field for display.
func Field(s string) string {
	return DefaultStyles.Field.Render(s)
}

// KeyValuePair formats a kvp for display.
func KeyValuePair(key, value string) string {
	k := Field(key)
	v := Keyword(value)
	return k + ":" + v
}

var (
	normalFg = lipgloss.AdaptiveColor{Light: "235", Dark: "252"}
	indigo   = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	cream    = lipgloss.AdaptiveColor{Light: "#FFFDF5", Dark: "#FFFDF5"}
	fuchsia  = lipgloss.Color("#F780E2")
)

func init() {
	// We do not adapt the color based on whether the terminal is light or dark.
	// Doing so would require a round-trip between the program and the terminal
	// due to the OSC query and response.
	DefaultStyles = Styles{
		Code: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			Foreground(lipgloss.Color("#ED567A")).
			Background(lipgloss.Color("#2C2C2C")),
		DateTimeStamp: lipgloss.NewStyle().
			Foreground(brightBlue),

		Error: lipgloss.NewStyle().
			Foreground(red),

		Field: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#2B2A2A")),

		Fuchsia: lipgloss.NewStyle().
			Foreground(brightMagenta),

		Hyperlink: lipgloss.NewStyle().
			Foreground(magenta).
			Underline(true),

		Keyword: lipgloss.NewStyle().
			Foreground(green),

		Placeholder: lipgloss.NewStyle().
			Foreground(magenta),

		Warn: lipgloss.NewStyle().
			Foreground(yellow),

		Wrap: lipgloss.NewStyle().
			Width(80),
	}
}

// ValidateNotEmpty is a helper function to disallow empty inputs!
func ValidateNotEmpty(s string) error {
	if s == "" {
		return errors.New("Must be provided!")
	}
	return nil
}
