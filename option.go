package serpent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/pflag"
)

type ValueSource string

const (
	ValueSourceNone    ValueSource = ""
	ValueSourceFlag    ValueSource = "flag"
	ValueSourceEnv     ValueSource = "env"
	ValueSourceYAML    ValueSource = "yaml"
	ValueSourceDefault ValueSource = "default"
)

var valueSourcePriority = []ValueSource{
	ValueSourceFlag,
	ValueSourceEnv,
	ValueSourceYAML,
	ValueSourceDefault,
	ValueSourceNone,
}

// Option is a configuration option for a CLI application.
type Option struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// Required means this value must be set by some means. It requires
	// `ValueSource != ValueSourceNone`
	// If `Default` is set, then `Required` is ignored.
	Required bool `json:"required,omitempty"`

	// Flag is the long name of the flag used to configure this option. If unset,
	// flag configuring is disabled.
	Flag string `json:"flag,omitempty"`
	// FlagShorthand is the one-character shorthand for the flag. If unset, no
	// shorthand is used.
	FlagShorthand string `json:"flag_shorthand,omitempty"`

	// Env is the environment variable used to configure this option. If unset,
	// environment configuring is disabled.
	Env string `json:"env,omitempty"`

	// YAML is the YAML key used to configure this option. If unset, YAML
	// configuring is disabled.
	YAML string `json:"yaml,omitempty"`

	// Default is parsed into Value if set.
	Default string `json:"default,omitempty"`
	// Value includes the types listed in values.go.
	Value pflag.Value `json:"value,omitempty"`

	// Annotations enable extensions to serpent higher up in the stack. It's useful for
	// help formatting and documentation generation.
	Annotations Annotations `json:"annotations,omitempty"`

	// Group is a group hierarchy that helps organize this option in help, configs
	// and other documentation.
	Group *Group `json:"group,omitempty"`

	// UseInstead is a list of options that should be used instead of this one.
	// The field is used to generate a deprecation warning.
	UseInstead []Option `json:"use_instead,omitempty"`

	Hidden bool `json:"hidden,omitempty"`

	ValueSource ValueSource `json:"value_source,omitempty"`

	CompletionHandler CompletionHandlerFunc `json:"-"`
}

// optionNoMethods is just a wrapper around Option so we can defer to the
// default json.Unmarshaler behavior.
type optionNoMethods Option

func (o *Option) UnmarshalJSON(data []byte) error {
	// If an option has no values, we have no idea how to unmarshal it.
	// So just discard the json data.
	if o.Value == nil {
		o.Value = &DiscardValue
	}

	return json.Unmarshal(data, (*optionNoMethods)(o))
}

func (o Option) YAMLPath() string {
	if o.YAML == "" {
		return ""
	}
	var gs []string
	for _, g := range o.Group.Ancestry() {
		gs = append(gs, g.YAML)
	}
	return strings.Join(append(gs, o.YAML), ".")
}

// OptionSet is a group of options that can be applied to a command.
type OptionSet []Option

// UnmarshalJSON implements json.Unmarshaler for OptionSets. Options have an
// interface Value type that cannot handle unmarshalling because the types cannot
// be inferred. Since it is a slice, instantiating the Options first does not
// help.
//
// However, we typically do instantiate the slice to have the correct types.
// So this unmarshaller will attempt to find the named option in the existing
// set, if it cannot, the value is discarded. If the option exists, the value
// is unmarshalled into the existing option, and replaces the existing option.
//
// The value is discarded if it's type cannot be inferred. This behavior just
// feels "safer", although it should never happen if the correct option set
// is passed in. The situation where this could occur is if a client and server
// are on different versions with different options.
func (optSet *OptionSet) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	// Should be a json array, so consume the starting open bracket.
	t, err := dec.Token()
	if err != nil {
		return fmt.Errorf("read array open bracket: %w", err)
	}
	if t != json.Delim('[') {
		return fmt.Errorf("expected array open bracket, got %q", t)
	}

	// As long as json elements exist, consume them. The counter is used for
	// better errors.
	var i int
OptionSetDecodeLoop:
	for dec.More() {
		var opt Option
		// jValue is a placeholder value that allows us to capture the
		// raw json for the value to attempt to unmarshal later.
		var jValue jsonValue
		opt.Value = &jValue
		err := dec.Decode(&opt)
		if err != nil {
			return fmt.Errorf("decode %d option: %w", i, err)
		}
		// This counter is used to contextualize errors to show which element of
		// the array we failed to decode. It is only used in the error above, as
		// if the above works, we can instead use the Option.Name which is more
		// descriptive and useful. So increment here for the next decode.
		i++

		// Try to see if the option already exists in the option set.
		// If it does, just update the existing option.
		for optIndex, have := range *optSet {
			if have.Name == opt.Name {
				if jValue != nil {
					err := json.Unmarshal(jValue, &(*optSet)[optIndex].Value)
					if err != nil {
						return fmt.Errorf("decode option %q value: %w", have.Name, err)
					}
					// Set the opt's value
					opt.Value = (*optSet)[optIndex].Value
				} else {
					// Hopefully the user passed empty values in the option set. There is no easy way
					// to tell, and if we do not do this, it breaks json.Marshal if we do it again on
					// this new option set.
					opt.Value = (*optSet)[optIndex].Value
				}
				// Override the existing.
				(*optSet)[optIndex] = opt
				// Go to the next option to decode.
				continue OptionSetDecodeLoop
			}
		}

		// If the option doesn't exist, the value will be discarded.
		// We do this because we cannot infer the type of the value.
		opt.Value = DiscardValue
		*optSet = append(*optSet, opt)
	}

	t, err = dec.Token()
	if err != nil {
		return fmt.Errorf("read array close bracket: %w", err)
	}
	if t != json.Delim(']') {
		return fmt.Errorf("expected array close bracket, got %q", t)
	}

	return nil
}

// Add adds the given Options to the OptionSet.
func (optSet *OptionSet) Add(opts ...Option) {
	*optSet = append(*optSet, opts...)
}

// Filter will only return options that match the given filter. (return true)
func (optSet OptionSet) Filter(filter func(opt Option) bool) OptionSet {
	cpy := make(OptionSet, 0)
	for _, opt := range optSet {
		if filter(opt) {
			cpy = append(cpy, opt)
		}
	}
	return cpy
}

// FlagSet returns a pflag.FlagSet for the OptionSet.
func (optSet *OptionSet) FlagSet() *pflag.FlagSet {
	if optSet == nil {
		return &pflag.FlagSet{}
	}

	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	for _, opt := range *optSet {
		if opt.Flag == "" {
			continue
		}
		var noOptDefValue string
		{
			no, ok := opt.Value.(NoOptDefValuer)
			if ok {
				noOptDefValue = no.NoOptDefValue()
			}
		}

		val := opt.Value
		if val == nil {
			val = DiscardValue
		}

		fs.AddFlag(&pflag.Flag{
			Name:        opt.Flag,
			Shorthand:   opt.FlagShorthand,
			Usage:       opt.Description,
			Value:       val,
			DefValue:    "",
			Changed:     false,
			Deprecated:  "",
			NoOptDefVal: noOptDefValue,
			Hidden:      opt.Hidden,
		})
	}
	fs.Usage = func() {
		_, _ = os.Stderr.WriteString("Override (*FlagSet).Usage() to print help text.\n")
	}
	return fs
}

// ParseEnv parses the given environment variables into the OptionSet.
// Use EnvsWithPrefix to filter out prefixes.
func (optSet *OptionSet) ParseEnv(vs []EnvVar) error {
	if optSet == nil {
		return nil
	}

	var merr *multierror.Error

	// We parse environment variables first instead of using a nested loop to
	// avoid N*M complexity when there are a lot of options and environment
	// variables.
	envs := make(map[string]string)
	for _, v := range vs {
		envs[v.Name] = v.Value
	}

	for i, opt := range *optSet {
		if opt.Env == "" {
			continue
		}

		envVal, ok := envs[opt.Env]
		if !ok {
			// Homebrew strips all environment variables that do not start with `HOMEBREW_`.
			// This prevented using brew to invoke the Coder agent, because the environment
			// variables to not get passed down.
			//
			// A customer wanted to use their custom tap inside a workspace, which was failing
			// because the agent lacked the environment variables to authenticate with Git.
			envVal, ok = envs[`HOMEBREW_`+opt.Env]
		}
		// Currently, empty values are treated as if the environment variable is
		// unset. This behavior is technically not correct as there is now no
		// way for a user to change a Default value to an empty string from
		// the environment. Unfortunately, we have old configuration files
		// that rely on the faulty behavior.
		//
		// TODO: We should remove this hack in May 2023, when deployments
		// have had months to migrate to the new behavior.
		if !ok || envVal == "" {
			continue
		}

		(*optSet)[i].ValueSource = ValueSourceEnv
		if err := opt.Value.Set(envVal); err != nil {
			merr = multierror.Append(
				merr, fmt.Errorf("parse %q: %w", opt.Name, err),
			)
		}
	}

	return merr.ErrorOrNil()
}

// SetDefaults sets the default values for each Option, skipping values
// that already have a value source.
func (optSet *OptionSet) SetDefaults() error {
	if optSet == nil {
		return nil
	}

	var merr *multierror.Error

	// It's common to have multiple options with the same value to
	// handle deprecation. We group the options by value so that we
	// don't let other options overwrite user input.
	groupByValue := make(map[pflag.Value][]*Option)
	for i := range *optSet {
		opt := &(*optSet)[i]
		if opt.Value == nil {
			merr = multierror.Append(
				merr,
				fmt.Errorf(
					"parse %q: no Value field set\nFull opt: %+v",
					opt.Name, opt,
				),
			)
			continue
		}
		groupByValue[opt.Value] = append(groupByValue[opt.Value], opt)
	}

	// Sorts by value source, then a default value being set.
	sortOptionByValueSourcePriorityOrDefault := func(a, b *Option) int {
		if a.ValueSource != b.ValueSource {
			return slices.Index(valueSourcePriority, a.ValueSource) - slices.Index(valueSourcePriority, b.ValueSource)
		}
		if a.Default != b.Default {
			if a.Default == "" {
				return 1
			}
			if b.Default == "" {
				return -1
			}
		}
		return 0
	}
	for _, opts := range groupByValue {
		// Sort the options by priority and whether or not a default is
		// set. This won't affect the value but represents correctness
		// from whence the value originated.
		slices.SortFunc(opts, sortOptionByValueSourcePriorityOrDefault)

		// If the first option has a value source, then we don't need to
		// set the default, but mark the source for all options.
		if opts[0].ValueSource != ValueSourceNone {
			for _, opt := range opts[1:] {
				opt.ValueSource = opts[0].ValueSource
			}
			continue
		}

		var optWithDefault *Option
		for _, opt := range opts {
			if opt.Default == "" {
				continue
			}
			if optWithDefault != nil && optWithDefault.Default != opt.Default {
				merr = multierror.Append(
					merr,
					fmt.Errorf(
						"parse %q: multiple defaults set for the same value: %q and %q (%q)",
						opt.Name, opt.Default, optWithDefault.Default, optWithDefault.Name,
					),
				)
				continue
			}
			optWithDefault = opt
		}
		if optWithDefault == nil {
			continue
		}
		if err := optWithDefault.Value.Set(optWithDefault.Default); err != nil {
			merr = multierror.Append(
				merr, fmt.Errorf("parse %q: %w", optWithDefault.Name, err),
			)
		}
		for _, opt := range opts {
			opt.ValueSource = ValueSourceDefault
		}
	}

	return merr.ErrorOrNil()
}

// ByName returns the Option with the given name, or nil if no such option
// exists.
func (optSet OptionSet) ByName(name string) *Option {
	for i := range optSet {
		if optSet[i].Name == name {
			return &optSet[i]
		}
	}
	return nil
}

func (optSet OptionSet) ByFlag(flag string) *Option {
	if flag == "" {
		return nil
	}
	for i := range optSet {
		opt := &optSet[i]
		if opt.Flag == flag {
			return opt
		}
	}
	return nil
}
