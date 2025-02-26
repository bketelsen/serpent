package main

import (
	"fmt"

	"github.com/bketelsen/serpent"
	"github.com/bketelsen/serpent/completion"
	"github.com/bketelsen/serpent/ui"
)

// installCommand returns a serpent command that helps
// a user configure their shell to use serpent's completion.
func installCommand() *serpent.Command {
	var shell string
	return &serpent.Command{
		Use:   "completion [--shell <shell>]",
		Short: "Generate completion scripts for the given shell.",
		Handler: func(inv *serpent.Invocation) error {
			defaultShell, err := completion.DetectUserShell(inv.Command.Parent.Name())
			if err != nil {
				return fmt.Errorf("Could not detect user shell, please specify a shell using `--shell`")
			}
			return defaultShell.WriteCompletion(inv.Stdout)
		},
		Options: serpent.OptionSet{
			{
				Flag:          "shell",
				FlagShorthand: "s",
				Description:   "The shell to generate a completion script for.",
				Value:         completion.ShellOptions(&shell),
			},
		},
	}
}

func main() {
	var (
		print    bool
		upper    bool
		fileType string
		fileArr  []string
		types    []string
	)
	cmd := serpent.Command{
		Use:   "completetest <text>",
		Short: "Prints a table.",
		ContactInfo: &serpent.ContactInfo{
			Issues: "https://github.com/bketelsen/serpent/issues/new",
		},

		Handler: func(inv *serpent.Invocation) error {

			people := []Person{
				{
					Name:       "Bill",
					Occupation: "Adventurer",
					Age:        21,
				},
				{
					Name:       "Ted",
					Occupation: "Adventurer",
					Age:        21,
				},
			}
			output, err := ui.DisplayTable(people, "", nil)
			if err != nil {
				return err
			}

			inv.Println(output)
			return nil
		},
		Children: []*serpent.Command{
			{
				Use:   "sub",
				Short: "A subcommand",
				Handler: func(inv *serpent.Invocation) error {
					inv.Println("subcommand")
					return nil
				},
				Options: serpent.OptionSet{
					{
						Name:        "upper",
						Value:       serpent.BoolOf(&upper),
						Flag:        "upper",
						Description: "Prints the text in upper case.",
					},
				},
			},
			{
				Use: "file <file>",
				Handler: func(inv *serpent.Invocation) error {
					return nil
				},
				Options: serpent.OptionSet{
					{
						Name:        "print",
						Value:       serpent.BoolOf(&print),
						Flag:        "print",
						Description: "Print the file.",
					},
					{
						Name:        "type",
						Value:       serpent.EnumOf(&fileType, "binary", "text"),
						Flag:        "type",
						Description: "The type of file.",
					},
					{
						Name:        "extra",
						Flag:        "extra",
						Description: "Extra files.",
						Value:       serpent.StringArrayOf(&fileArr),
					},
					{
						Name:  "types",
						Flag:  "types",
						Value: serpent.EnumArrayOf(&types, "binary", "text"),
					},
				},
				CompletionHandler: completion.FileHandler(nil),
				Middleware:        serpent.RequireNArgs(1),
			},
			installCommand(),
		},
	}

	inv := cmd.Invoke().WithOS()

	err := inv.Run()
	if err != nil {
		panic(err)
	}
}

type Person struct {
	Name       string `table:"name,default_sort"`
	Age        int    `table:"age"`
	Occupation string `table:"occupation"`
}
