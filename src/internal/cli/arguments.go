package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	noun := "argument"
	pronoun := "it"
	if len(args) > 1 {
		noun = "arguments"
		pronoun = "them"
	}
	return NewError(
		fmt.Sprintf("unexpected %s %s for %q", noun, quoteArgs(args), cmd.CommandPath()),
		fmt.Sprintf(
			"Remove %s and run '%s --help' to see the supported options.",
			pronoun,
			cmd.CommandPath(),
		),
	)
}

func Subcommands(noun string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}

		hint := fmt.Sprintf(
			"Run '%s --help' to see the available %ss.",
			cmd.CommandPath(),
			noun,
		)
		if cmd.SuggestionsMinimumDistance <= 0 {
			cmd.SuggestionsMinimumDistance = 2
		}
		if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
			hint = fmt.Sprintf(
				"Did you mean '%s %s'?",
				cmd.CommandPath(),
				suggestions[0],
			)
		}
		return NewError(
			fmt.Sprintf("unknown %s %q for %q", noun, args[0], cmd.CommandPath()),
			hint,
		)
	}
}

func quoteArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, strconv.Quote(arg))
	}
	return strings.Join(quoted, ", ")
}
