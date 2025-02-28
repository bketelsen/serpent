package serpent

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"testing"
	"unicode"

	"github.com/charmbracelet/log"
	"github.com/spf13/pflag"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var logger = log.New(os.Stderr)

type ContactInfo struct {
	Repo   string
	Issues string
	Chat   string
	Email  string
}

func (c *ContactInfo) RepoLink() string {
	return c.Repo
}
func (c *ContactInfo) IssuesLink() string {
	return c.Issues
}
func (c *ContactInfo) ChatLink() string {
	return c.Chat
}
func (c *ContactInfo) EmailLink() string {
	return c.Email
}

// Command describes an executable command.
type Command struct {
	// Parent is the direct parent of the command.
	//
	// It is set automatically when an invokation runs.
	Parent *Command

	// Children is a list of direct descendants.
	Children []*Command

	// Use is provided in form "command [flags] [args...]".
	Use string

	// Aliases is a list of alternative names for the command.
	Aliases []string

	// Short is a one-line description of the command.
	Short string

	// Hidden determines whether the command should be hidden from help.
	Hidden bool

	// Deprecated indicates whether this command is deprecated.
	// If empty, the command is not deprecated.
	// If set, the value is used as the deprecation message.
	Deprecated string `json:"deprecated,omitempty"`

	// RawArgs determines whether the command should receive unparsed arguments.
	// No flags are parsed when set, and the command is responsible for parsing
	// its own flags.
	RawArgs bool

	// Long is a detailed description of the command,
	// presented on its help page. It may contain examples.
	Long        string
	Options     OptionSet
	Annotations Annotations

	// Middleware is called before the Handler.
	// Use Chain() to combine multiple middlewares.
	Middleware  MiddlewareFunc
	Handler     HandlerFunc
	HelpHandler HandlerFunc
	// CompletionHandler is called when the command is run in completion
	// mode. If nil, only the default completion handler is used.
	//
	// Flag and option parsing is best-effort in this mode, so even if an Option
	// is "required" it may not be set.
	CompletionHandler CompletionHandlerFunc

	ContactInfo *ContactInfo

	// Version defines the version for this command. If this value is non-empty and the command does not
	// define a "version" flag, a "version" boolean flag will be added to the command and, if specified,
	// will print content of the "Version" variable. A shorthand "v" flag will also be added if the
	// command does not define one.
	Version string
}

// AddSubcommands adds the given subcommands, setting their
// Parent field automatically.
func (c *Command) AddSubcommands(cmds ...*Command) {
	for _, cmd := range cmds {
		cmd.Parent = c
		c.Children = append(c.Children, cmd)
	}
}

// Walk calls fn for the command and all its children.
func (c *Command) Walk(fn func(*Command)) {
	fn(c)
	for _, child := range c.Children {
		child.Parent = c
		child.Walk(fn)
	}
}

func ascendingSortFn[T constraints.Ordered](a, b T) int {
	if a < b {
		return -1
	} else if a == b {
		return 0
	}
	return 1
}

// init performs initialization and linting on the command and all its children.
func (c *Command) init() error {
	if c.Use == "" {
		c.Use = "unnamed"
	}
	var merr error

	for i := range c.Options {
		opt := &c.Options[i]
		if opt.Name == "" {
			switch {
			case opt.Flag != "":
				opt.Name = opt.Flag
			case opt.Env != "":
				opt.Name = opt.Env
			case opt.YAML != "":
				opt.Name = opt.YAML
			default:
				merr = errors.Join(merr, fmt.Errorf("option must have a Name, Flag, Env or YAML field"))
			}
		}
		if opt.Description != "" {
			// Enforce that description uses sentence form.
			if unicode.IsLower(rune(opt.Description[0])) {
				merr = errors.Join(merr, fmt.Errorf("option %q description should start with a capital letter", opt.Name))
			}
			if !strings.HasSuffix(opt.Description, ".") {
				merr = errors.Join(merr, fmt.Errorf("option %q description should end with a period", opt.Name))
			}
		}
	}

	slices.SortFunc(c.Options, func(a, b Option) int {
		return ascendingSortFn(a.Name, b.Name)
	})
	slices.SortFunc(c.Children, func(a, b *Command) int {
		return ascendingSortFn(a.Name(), b.Name())
	})
	for _, child := range c.Children {
		child.Parent = c
		err := child.init()
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("command %v: %w", child.Name(), err))
		}
	}
	hasVersion := false
	if c.Parent == nil {
		if c.Version != "" {
			nameVersion := c.Options.ByName("version")
			if nameVersion != nil {
				hasVersion = true
			}
			flagVersion := c.Options.ByFlag("version")
			if flagVersion != nil {
				hasVersion = true
			}

			if !hasVersion {
				var val bool
				c.Options.Add(Option{
					Flag:  "version",
					Value: BoolOf(&val),
					Name:  "version",
				})
			}

		}
	}
	return merr
}

// Name returns the first word in the Use string.
func (c *Command) Name() string {
	return strings.Split(c.Use, " ")[0]
}

// FullName returns the full invocation name of the command,
// as seen on the command line.
func (c *Command) FullName() string {
	var names []string
	if c.Parent != nil {
		names = append(names, c.Parent.FullName())
	}
	names = append(names, c.Name())
	return strings.Join(names, " ")
}

// FullName returns usage of the command, preceded
// by the usage of its parents.
func (c *Command) FullUsage() string {
	var uses []string
	if c.Parent != nil {
		uses = append(uses, c.Parent.FullName())
	}
	uses = append(uses, c.Use)
	return strings.Join(uses, " ")
}

// FullOptions returns the options of the command and its parents.
func (c *Command) FullOptions() OptionSet {
	var opts OptionSet
	if c.Parent != nil {
		opts = append(opts, c.Parent.FullOptions()...)
	}
	opts = append(opts, c.Options...)
	return opts
}

// Invoke creates a new invocation of the command, with
// stdio discarded.
//
// The returned invocation is not live until Run() is called.
func (c *Command) Invoke(args ...string) *Invocation {
	return &Invocation{
		Command: c,
		Args:    args,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		Stdin:   strings.NewReader(""),
		Logger:  logger,
	}
}

// Invocation represents an instance of a command being executed.
type Invocation struct {
	ctx         context.Context
	Command     *Command
	parsedFlags *pflag.FlagSet

	// Args is reduced into the remaining arguments after parsing flags
	// during Run.
	Args []string

	// Environ is a list of environment variables. Use EnvsWithPrefix to parse
	// os.Environ.
	Environ Environ
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader

	Logger *log.Logger
	// Deprecated
	Net Net

	// testing
	signalNotifyContext func(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc)
}

// Print is a convenience method to Print to the defined output, fallback to Stderr if not set.
func (inv *Invocation) Print(i ...interface{}) {
	fmt.Fprint(inv.Stdout, i...)
}

// Println is a convenience method to Println to the defined output, fallback to Stderr if not set.
func (inv *Invocation) Println(i ...interface{}) {
	inv.Print(fmt.Sprintln(i...))
}

// Printf is a convenience method to Printf to the defined output, fallback to Stderr if not set.
func (inv *Invocation) Printf(format string, i ...interface{}) {
	inv.Print(fmt.Sprintf(format, i...))
}

// PrintErr is a convenience method to Print to the defined Err output, fallback to Stderr if not set.
func (inv *Invocation) PrintErr(i ...interface{}) {
	fmt.Fprint(inv.Stderr, i...)
}

// PrintErrln is a convenience method to Println to the defined Err output, fallback to Stderr if not set.
func (inv *Invocation) PrintErrln(i ...interface{}) {
	inv.PrintErr(fmt.Sprintln(i...))
}

// PrintErrf is a convenience method to Printf to the defined Err output, fallback to Stderr if not set.
func (inv *Invocation) PrintErrf(format string, i ...interface{}) {
	inv.PrintErr(fmt.Sprintf(format, i...))
}

// WithOS returns the invocation as a main package, filling in the invocation's unset
// fields with OS defaults.
func (inv *Invocation) WithOS() *Invocation {
	return inv.with(func(i *Invocation) {
		i.Stdout = os.Stdout
		i.Stderr = os.Stderr
		i.Stdin = os.Stdin
		i.Args = os.Args[1:]
		i.Environ = ParseEnviron(os.Environ(), "")
		i.Net = osNet{}
		log.SetOutput(i.Stderr)
	})
}

// WithTestSignalNotifyContext allows overriding the default implementation of SignalNotifyContext.
// This should only be used in testing.
func (inv *Invocation) WithTestSignalNotifyContext(
	_ testing.TB, // ensure we only call this from tests
	f func(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc),
) *Invocation {
	return inv.with(func(i *Invocation) {
		i.signalNotifyContext = f
	})
}

// SignalNotifyContext is equivalent to signal.NotifyContext, but supports being overridden in
// tests.
func (inv *Invocation) SignalNotifyContext(parent context.Context, signals ...os.Signal) (ctx context.Context, stop context.CancelFunc) {
	if inv.signalNotifyContext == nil {
		return signal.NotifyContext(parent, signals...)
	}
	return inv.signalNotifyContext(parent, signals...)
}

func (inv *Invocation) WithTestParsedFlags(
	_ testing.TB, // ensure we only call this from tests
	parsedFlags *pflag.FlagSet,
) *Invocation {
	return inv.with(func(i *Invocation) {
		i.parsedFlags = parsedFlags
	})
}

func (inv *Invocation) Context() context.Context {
	if inv.ctx == nil {
		return context.Background()
	}
	return inv.ctx
}

func (inv *Invocation) ParsedFlags() *pflag.FlagSet {
	if inv.parsedFlags == nil {
		panic("flags not parsed, has Run() been called?")
	}
	return inv.parsedFlags
}

type runState struct {
	allArgs      []string
	commandDepth int

	flagParseErr error
}

func copyFlagSetWithout(fs *pflag.FlagSet, without string) *pflag.FlagSet {
	fs2 := pflag.NewFlagSet("", pflag.ContinueOnError)
	fs2.Usage = func() {}
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Name == without {
			return
		}
		fs2.AddFlag(f)
	})
	return fs2
}

func (inv *Invocation) CurWords() (prev string, cur string) {
	switch len(inv.Args) {
	// All the shells we support will supply at least one argument (empty string),
	// but we don't want to panic.
	case 0:
		cur = ""
		prev = ""
	case 1:
		cur = inv.Args[0]
		prev = ""
	default:
		cur = inv.Args[len(inv.Args)-1]
		prev = inv.Args[len(inv.Args)-2]
	}
	return
}

// run recursively executes the command and its children.
// allArgs is wired through the stack so that global flags can be accepted
// anywhere in the command invocation.
func (inv *Invocation) run(state *runState) error {
	if inv.Command.Deprecated != "" {
		fmt.Fprintf(inv.Stderr, "%s %q is deprecated!. %s\n",
			prettyHeader("warning"),
			inv.Command.FullName(),
			inv.Command.Deprecated,
		)
	}

	err := inv.Command.Options.ParseEnv(inv.Environ)
	if err != nil {
		return fmt.Errorf("parsing env: %w", err)
	}

	// Now the fun part, argument parsing!

	children := make(map[string]*Command)
	for _, child := range inv.Command.Children {
		child.Parent = inv.Command
		for _, name := range append(child.Aliases, child.Name()) {
			if _, ok := children[name]; ok {
				return fmt.Errorf("duplicate command name: %s", name)
			}
			children[name] = child
		}
	}

	if inv.parsedFlags == nil {
		inv.parsedFlags = pflag.NewFlagSet(inv.Command.Name(), pflag.ContinueOnError)
		// We handle Usage ourselves.
		inv.parsedFlags.Usage = func() {}
	}

	// If we find a duplicate flag, we want the deeper command's flag to override
	// the shallow one. Unfortunately, pflag has no way to remove a flag, so we
	// have to create a copy of the flagset without a value.
	inv.Command.Options.FlagSet().VisitAll(func(f *pflag.Flag) {
		if inv.parsedFlags.Lookup(f.Name) != nil {
			inv.parsedFlags = copyFlagSetWithout(inv.parsedFlags, f.Name)
		}
		inv.parsedFlags.AddFlag(f)
	})

	var parsedArgs []string

	if !inv.Command.RawArgs {
		// Flag parsing will fail on intermediate commands in the command tree,
		// so we check the error after looking for a child command.
		state.flagParseErr = inv.parsedFlags.Parse(state.allArgs)
		parsedArgs = inv.parsedFlags.Args()
	}

	// Set value sources for flags.
	for i, opt := range inv.Command.Options {
		if fl := inv.parsedFlags.Lookup(opt.Flag); fl != nil && fl.Changed {
			inv.Command.Options[i].ValueSource = ValueSourceFlag
		}
	}

	// Read YAML configs, if any.
	for _, opt := range inv.Command.Options {
		path, ok := opt.Value.(*YAMLConfigPath)
		if !ok || path.String() == "" {
			continue
		}

		byt, err := os.ReadFile(path.String())
		if err != nil {
			return fmt.Errorf("reading yaml: %w", err)
		}

		var n yaml.Node
		err = yaml.Unmarshal(byt, &n)
		if err != nil {
			return fmt.Errorf("decoding yaml: %w", err)
		}

		err = inv.Command.Options.UnmarshalYAML(&n)
		if err != nil {
			return fmt.Errorf("applying yaml: %w", err)
		}
	}

	err = inv.Command.Options.SetDefaults()
	if err != nil {
		return fmt.Errorf("setting defaults: %w", err)
	}

	// Run child command if found (next child only)
	// We must do subcommand detection after flag parsing so we don't mistake flag
	// values for subcommand names.
	if len(parsedArgs) > state.commandDepth {
		nextArg := parsedArgs[state.commandDepth]
		if child, ok := children[nextArg]; ok {
			child.Parent = inv.Command
			inv.Command = child
			state.commandDepth++
			return inv.run(state)
		}
	}

	// Outputted completions are not filtered based on the word under the cursor, as every shell we support does this already.
	// We only look at the current word to figure out handler to run, or what directory to inspect.
	if inv.IsCompletionMode() {
		for _, e := range inv.complete() {
			fmt.Fprintln(inv.Stdout, e)
		}
		return nil
	}

	ignoreFlagParseErrors := inv.Command.RawArgs

	// Flag parse errors are irrelevant for raw args commands.
	if !ignoreFlagParseErrors && state.flagParseErr != nil && !errors.Is(state.flagParseErr, pflag.ErrHelp) {
		return fmt.Errorf(
			"parsing flags (%v) for %q: %w",
			state.allArgs,
			inv.Command.FullName(), state.flagParseErr,
		)
	}

	// All options should be set. Check all required options have sources,
	// meaning they were set by the user in some way (env, flag, etc).
	var missing []string
	for _, opt := range inv.Command.Options {
		if opt.Required && opt.ValueSource == ValueSourceNone {
			name := opt.Name
			// use flag as a fallback if name is empty
			if name == "" {
				name = opt.Flag
			}
			missing = append(missing, name)
		}
	}
	// Don't error for missing flags if `--help` was supplied.
	if len(missing) > 0 && !inv.IsCompletionMode() && !errors.Is(state.flagParseErr, pflag.ErrHelp) {
		return fmt.Errorf("missing values for the required flags: %s", strings.Join(missing, ", "))
	}

	if inv.Command.RawArgs {
		// If we're at the root command, then the name is omitted
		// from the arguments, so we can just use the entire slice.
		if state.commandDepth == 0 {
			inv.Args = state.allArgs
		} else {
			argPos, err := findArg(inv.Command.Name(), state.allArgs, inv.parsedFlags)
			if err != nil {
				panic(err)
			}
			inv.Args = state.allArgs[argPos+1:]
		}
	} else {
		// In non-raw-arg mode, we want to skip over flags.
		inv.Args = parsedArgs[state.commandDepth:]
	}
	if inv.Command.Version != "" {
		vflag := inv.Command.Options.ByFlag("version")
		if vflag != nil {
			fl := inv.parsedFlags.Lookup(vflag.Flag)
			if fl != nil && fl.Changed {
				inv.Println(inv.Command.Name() + " " + inv.Command.Version)
				return nil
			}

		}
	}
	mw := inv.Command.Middleware
	if mw == nil {
		mw = Chain()
	}

	ctx := inv.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	inv = inv.WithContext(ctx)

	if inv.Command.Handler == nil || errors.Is(state.flagParseErr, pflag.ErrHelp) {
		if inv.Command.HelpHandler == nil {
			return DefaultHelpFn()(inv)
		}
		return inv.Command.HelpHandler(inv)
	}

	err = mw(inv.Command.Handler)(inv)
	if err != nil {
		return &RunCommandError{
			Cmd: inv.Command,
			Err: err,
		}
	}
	return nil
}

type RunCommandError struct {
	Cmd *Command
	Err error
}

func (e *RunCommandError) Unwrap() error {
	return e.Err
}

func (e *RunCommandError) Error() string {
	return fmt.Sprintf("running command %q: %+v", e.Cmd.FullName(), e.Err)
}

// findArg returns the index of the first occurrence of arg in args, skipping
// over all flags.
func findArg(want string, args []string, fs *pflag.FlagSet) (int, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			if arg == want {
				return i, nil
			}
			continue
		}

		// This is a flag!
		if strings.Contains(arg, "=") {
			// The flag contains the value in the same arg, just skip.
			continue
		}

		// We need to check if NoOptValue is set, then we should not wait
		// for the next arg to be the value.
		f := fs.Lookup(strings.TrimLeft(arg, "-"))
		if f == nil {
			return -1, fmt.Errorf("unknown flag: %s", arg)
		}
		if f.NoOptDefVal != "" {
			continue
		}

		if i == len(args)-1 {
			return -1, fmt.Errorf("flag %s requires a value", arg)
		}

		// Skip the value.
		i++
	}

	return -1, fmt.Errorf("arg %s not found", want)
}

// Run executes the command.
// If two command share a flag name, the first command wins.
//
//nolint:revive
func (inv *Invocation) Run() (err error) {
	err = inv.Command.init()
	if err != nil {
		return fmt.Errorf("initializing command: %w", err)
	}

	defer func() {
		// Pflag is panicky, so additional context is helpful in tests.
		if flag.Lookup("test.v") == nil {
			return
		}
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered for %s: %v", inv.Command.FullName(), r)
			panic(err)
		}
	}()
	// We close Stdin to prevent deadlocks, e.g. when the command
	// has ended but an io.Copy is still reading from Stdin.
	defer func() {
		if inv.Stdin == nil {
			return
		}
		rc, ok := inv.Stdin.(io.ReadCloser)
		if !ok {
			return
		}
		e := rc.Close()
		err = errors.Join(err, e)
	}()
	err = inv.run(&runState{
		allArgs: inv.Args,
	})
	return err
}

// WithContext returns a copy of the Invocation with the given context.
func (inv *Invocation) WithContext(ctx context.Context) *Invocation {
	return inv.with(func(i *Invocation) {
		i.ctx = ctx
	})
}

// with returns a copy of the Invocation with the given function applied.
func (inv *Invocation) with(fn func(*Invocation)) *Invocation {
	i2 := *inv
	fn(&i2)
	return &i2
}

func (inv *Invocation) complete() []string {
	prev, cur := inv.CurWords()

	// If the current word is a flag
	if strings.HasPrefix(cur, "--") {
		flagParts := strings.Split(cur, "=")
		flagName := flagParts[0][2:]
		// If it's an equals flag
		if len(flagParts) == 2 {
			if out := inv.completeFlag(flagName); out != nil {
				for i, o := range out {
					out[i] = fmt.Sprintf("--%s=%s", flagName, o)
				}
				return out
			}
		} else if out := inv.Command.Options.ByFlag(flagName); out != nil {
			// If the current word is a valid flag, auto-complete it so the
			// shell moves the cursor
			return []string{cur}
		}
	}
	// If the previous word is a flag, then we're writing it's value
	// and we should check it's handler
	if strings.HasPrefix(prev, "--") {
		word := prev[2:]
		if out := inv.completeFlag(word); out != nil {
			return out
		}
	}
	// If the current word is the command, move the shell cursor
	if inv.Command.Name() == cur {
		return []string{inv.Command.Name()}
	}
	var completions []string

	if inv.Command.CompletionHandler != nil {
		completions = append(completions, inv.Command.CompletionHandler(inv)...)
	}

	completions = append(completions, DefaultCompletionHandler(inv)...)

	return completions
}

func (inv *Invocation) completeFlag(word string) []string {
	opt := inv.Command.Options.ByFlag(word)
	if opt == nil {
		return nil
	}
	if opt.CompletionHandler != nil {
		return opt.CompletionHandler(inv)
	}
	enum, ok := opt.Value.(*Enum)
	if ok {
		return enum.Choices
	}
	enumArr, ok := opt.Value.(*EnumArray)
	if ok {
		return enumArr.Choices
	}
	return nil
}

// MiddlewareFunc returns the next handler in the chain,
// or nil if there are no more.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

func chain(ms ...MiddlewareFunc) MiddlewareFunc {
	return MiddlewareFunc(func(next HandlerFunc) HandlerFunc {
		if len(ms) > 0 {
			return chain(ms[1:]...)(ms[0](next))
		}
		return next
	})
}

// Chain returns a Handler that first calls middleware in order.
//
//nolint:revive
func Chain(ms ...MiddlewareFunc) MiddlewareFunc {
	// We need to reverse the array to provide top-to-bottom execution
	// order when defining a command.
	reversed := make([]MiddlewareFunc, len(ms))
	for i := range ms {
		reversed[len(ms)-1-i] = ms[i]
	}
	return chain(reversed...)
}

func RequireNArgs(want int) MiddlewareFunc {
	return RequireRangeArgs(want, want)
}

// RequireRangeArgs returns a Middleware that requires the number of arguments
// to be between start and end (inclusive). If end is -1, then the number of
// arguments must be at least start.
func RequireRangeArgs(start, end int) MiddlewareFunc {
	if start < 0 {
		panic("start must be >= 0")
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(i *Invocation) error {
			got := len(i.Args)
			switch {
			case start == end && got != start:
				switch start {
				case 0:
					if len(i.Command.Children) > 0 {
						return fmt.Errorf("unrecognized subcommand %q", i.Args[0])
					}
					return fmt.Errorf("wanted no args but got %v %v", got, i.Args)
				default:
					return fmt.Errorf(
						"wanted %v args but got %v %v",
						start,
						got,
						i.Args,
					)
				}
			case start > 0 && end == -1:
				switch {
				case got < start:
					return fmt.Errorf(
						"wanted at least %v args but got %v",
						start,
						got,
					)
				default:
					return next(i)
				}
			case start > end:
				panic("start must be <= end")
			case got < start || got > end:
				return fmt.Errorf(
					"wanted between %v and %v args but got %v",
					start, end,
					got,
				)
			default:
				return next(i)
			}
		}
	}
}

// HandlerFunc handles an Invocation of a command.
type HandlerFunc func(i *Invocation) error

type CompletionHandlerFunc func(i *Invocation) []string
