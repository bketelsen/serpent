package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bketelsen/serpent"
)

var Version = "1.0.0-dev"

func main() {
	var upper bool
	cmd := serpent.Command{
		Use:     "echo <text>",
		Short:   "Prints the given text to the console.",
		Long:    "Prints the given text to the console.",
		Version: Version,
		Options: serpent.OptionSet{
			{

				Value:       serpent.BoolOf(&upper),
				Flag:        "upper",
				Description: "Prints the text in upper case.",
			},
		},
		ContactInfo: &serpent.ContactInfo{
			Issues: "https://github.com/bketelsen/serpent/issues/new",
		},

		Handler: func(inv *serpent.Invocation) error {
			if len(inv.Args) == 0 {
				inv.PrintErrln("error: missing text")
				os.Exit(1)
			}

			text := inv.Args[0]
			if upper {
				text = strings.ToUpper(text)

			}
			inv.Warn("Header", "a line")
			inv.Info("Header", "a line")
			inv.Error("Header", "a line")
			inv.Logger.Info("hello")

			inv.Println(text)

			return nil
		},
	}

	err := cmd.Invoke().WithOS().Run()
	if err != nil {
		fmt.Println("Error:", err.Error())

	}
}
