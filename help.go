package serpent

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"

	"github.com/mitchellh/go-wordwrap"
	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/coder/pretty"
)

//go:embed help.tpl
var helpTemplateRaw string

type optionGroup struct {
	Name        string
	Description string
	Options     OptionSet
}

func ttyWidth() int {
	width, _, err := term.GetSize(0)
	if err != nil {
		return 80
	}
	return width
}

// wrapTTY wraps a string to the width of the terminal, or 80 no terminal
// is detected.
func wrapTTY(s string) string {
	return wordwrap.WrapString(s, uint(ttyWidth()))
}

var (
	helpColorProfile termenv.Profile
	helpColorOnce    sync.Once
)

// Color returns a color for the given string.
func helpColor(s string) termenv.Color {
	helpColorOnce.Do(func() {
		helpColorProfile = termenv.NewOutput(os.Stdout).ColorProfile()
		if flag.Lookup("test.v") != nil {
			// Use a consistent colorless profile in tests so that results
			// are deterministic.
			helpColorProfile = termenv.Ascii
		}
	})
	return helpColorProfile.Color(s)
}

// prettyHeader formats a header string with consistent styling.
// It uppercases the text, adds a colon, and applies the header color.
func prettyHeader(s string) string {
	headerFg := pretty.FgColor(helpColor("#337CA0"))
	s = strings.ToUpper(s)
	txt := pretty.String(s, ":")
	headerFg.Format(txt)
	return txt.String()
}

var defaultHelpTemplate = func() *template.Template {
	var (
		optionFg = pretty.FgColor(
			helpColor("#04A777"),
		)
	)
	return template.Must(
		template.New("usage").Funcs(
			template.FuncMap{
				"wrapTTY": func(s string) string {
					return wrapTTY(s)
				},
				"trimNewline": func(s string) string {
					return strings.TrimSuffix(s, "\n")
				},
				"keyword": func(s string) string {
					txt := pretty.String(s)
					optionFg.Format(txt)
					return txt.String()
				},
				"prettyHeader": prettyHeader,
				"typeHelper": func(opt *Option) string {
					switch v := opt.Value.(type) {
					case *Enum:
						return strings.Join(v.Choices, "|")
					case *EnumArray:
						return fmt.Sprintf("[%s]", strings.Join(v.Choices, "|"))
					default:
						return v.Type()
					}
				},
				"joinStrings": func(s []string) string {
					return strings.Join(s, ", ")
				},
				"indent": func(body string, spaces int) string {
					twidth := ttyWidth()

					spacing := strings.Repeat(" ", spaces)

					wrapLim := twidth - len(spacing)
					body = wordwrap.WrapString(body, uint(wrapLim))

					sc := bufio.NewScanner(strings.NewReader(body))

					var sb strings.Builder
					for sc.Scan() {
						// Remove existing indent, if any.
						// line = strings.TrimSpace(line)
						// Use spaces so we can easily calculate wrapping.
						_, _ = sb.WriteString(spacing)
						_, _ = sb.Write(sc.Bytes())
						_, _ = sb.WriteString("\n")
					}
					return sb.String()
				},
				"rootCommandName": func(cmd *Command) string {
					return strings.Split(cmd.FullName(), " ")[0]
				},
				"formatSubcommand": func(cmd *Command) string {
					// Minimize padding by finding the longest neighboring name.
					maxNameLength := len(cmd.Name())
					if parent := cmd.Parent; parent != nil {
						for _, c := range parent.Children {
							if len(c.Name()) > maxNameLength {
								maxNameLength = len(c.Name())
							}
						}
					}

					var sb strings.Builder
					_, _ = fmt.Fprintf(
						&sb, "%s%s%s",
						strings.Repeat(" ", 4), cmd.Name(), strings.Repeat(" ", maxNameLength-len(cmd.Name())+4),
					)

					// This is the point at which indentation begins if there's a
					// next line.
					descStart := sb.Len()

					twidth := ttyWidth()

					for i, line := range strings.Split(
						wordwrap.WrapString(cmd.Short, uint(twidth-descStart)), "\n",
					) {
						if i > 0 {
							_, _ = sb.WriteString(strings.Repeat(" ", descStart))
						}
						_, _ = sb.WriteString(line)
						_, _ = sb.WriteString("\n")
					}

					return sb.String()
				},
				"envName": func(opt Option) string {
					if opt.Env == "" {
						return ""
					}
					return opt.Env
				},
				"flagName": func(opt Option) string {
					return opt.Flag
				},

				"isDeprecated": func(opt Option) bool {
					return len(opt.UseInstead) > 0
				},
				"useInstead": func(opt Option) string {
					var sb strings.Builder
					for i, s := range opt.UseInstead {
						if i > 0 {
							if i == len(opt.UseInstead)-1 {
								_, _ = sb.WriteString(" and ")
							} else {
								_, _ = sb.WriteString(", ")
							}
						}
						if s.Flag != "" {
							_, _ = sb.WriteString("--")
							_, _ = sb.WriteString(s.Flag)
						} else if s.FlagShorthand != "" {
							_, _ = sb.WriteString("-")
							_, _ = sb.WriteString(s.FlagShorthand)
						} else if s.Env != "" {
							_, _ = sb.WriteString("$")
							_, _ = sb.WriteString(s.Env)
						} else {
							_, _ = sb.WriteString(s.Name)
						}
					}
					return sb.String()
				},
				"formatGroupDescription": func(s string) string {
					s = strings.ReplaceAll(s, "\n", "")
					s = s + "\n"
					s = wrapTTY(s)
					return s
				},
				"visibleChildren": func(cmd *Command) []*Command {
					return filterSlice(cmd.Children, func(c *Command) bool {
						return !c.Hidden
					})
				},
				"optionGroups": func(cmd *Command) []optionGroup {
					groups := []optionGroup{{
						// Default group.
						Name:        "",
						Description: "",
					}}

					// Sort options lexicographically.
					sort.Slice(cmd.Options, func(i, j int) bool {
						return cmd.Options[i].Name < cmd.Options[j].Name
					})

				optionLoop:
					for _, opt := range cmd.Options {
						if opt.Hidden {
							continue
						}

						if len(opt.Group.Ancestry()) == 0 {
							// Just add option to default group.
							groups[0].Options = append(groups[0].Options, opt)
							continue
						}

						groupName := opt.Group.FullName()

						for i, foundGroup := range groups {
							if foundGroup.Name != groupName {
								continue
							}
							groups[i].Options = append(groups[i].Options, opt)
							continue optionLoop
						}

						groups = append(groups, optionGroup{
							Name:        groupName,
							Description: opt.Group.Description,
							Options:     OptionSet{opt},
						})
					}
					sort.Slice(groups, func(i, j int) bool {
						// Sort groups lexicographically.
						return groups[i].Name < groups[j].Name
					})

					return filterSlice(groups, func(g optionGroup) bool {
						return len(g.Options) > 0
					})
				},
			},
		).Parse(helpTemplateRaw),
	)
}()

func filterSlice[T any](s []T, f func(T) bool) []T {
	var r []T
	for _, v := range s {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

// newLineLimiter makes working with Go templates more bearable. Without this,
// modifying the template is a slow toil of counting newlines and constantly
// checking that a change to one command's help doesn't break another.
type newlineLimiter struct {
	// w is not an interface since we call WriteRune byte-wise,
	// and the devirtualization overhead is significant.
	w     *bufio.Writer
	limit int

	newLineCounter int
}

// isSpace is a based on unicode.IsSpace, but only checks ASCII characters.
func isSpace(b byte) bool {
	switch b {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0:
		return true
	}
	return false
}

func (lm *newlineLimiter) Write(p []byte) (int, error) {
	for _, b := range p {
		switch {
		case b == '\r':
			// Carriage returns can sneak into `help.tpl` when `git clone`
			// is configured to automatically convert line endings.
			continue
		case b == '\n':
			lm.newLineCounter++
			if lm.newLineCounter > lm.limit {
				continue
			}
		case !isSpace(b):
			lm.newLineCounter = 0
		}
		err := lm.w.WriteByte(b)
		if err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

var usageWantsArgRe = regexp.MustCompile(`<.*>`)

type UnknownSubcommandError struct {
	Args []string
}

func (e *UnknownSubcommandError) Error() string {
	return fmt.Sprintf("unknown subcommand %q", strings.Join(e.Args, " "))
}

// DefaultHelpFn returns a function that generates usage (help)
// output for a given command.
func DefaultHelpFn() HandlerFunc {
	return func(inv *Invocation) error {
		// We use stdout for help and not stderr since there's no straightforward
		// way to distinguish between a user error and a help request.
		//
		// We buffer writes to stdout because the newlineLimiter writes one
		// rune at a time.
		outBuf := bufio.NewWriter(inv.Stdout)
		out := newlineLimiter{w: outBuf, limit: 2}
		tabwriter := tabwriter.NewWriter(&out, 0, 0, 2, ' ', 0)
		err := defaultHelpTemplate.Execute(tabwriter, inv.Command)
		if err != nil {
			return fmt.Errorf("execute template: %w", err)
		}
		err = tabwriter.Flush()
		if err != nil {
			return err
		}
		err = outBuf.Flush()
		if err != nil {
			return err
		}
		if len(inv.Args) > 0 && !usageWantsArgRe.MatchString(inv.Command.Use) {
			_, _ = fmt.Fprintf(inv.Stderr, "---\nerror: unknown subcommand %q\n", inv.Args[0])
		}
		if len(inv.Args) > 0 {
			// Return an error so that exit status is non-zero when
			// a subcommand is not found.
			return &UnknownSubcommandError{Args: inv.Args}
		}
		return nil
	}
}
