package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func FlagError(cmd *cobra.Command, err error) error {
	hint := fmt.Sprintf(
		"Run '%s --help' to see the available options.",
		cmd.CommandPath(),
	)

	// Parrot's packaged pflag predates structured flag errors. Its error
	// messages are stable across both versions, so keep one presentation path.
	if actionable := parseFlagError(cmd, err.Error(), hint); actionable != nil {
		return actionable
	}
	return NewError(err.Error(), hint)
}

func parseFlagError(cmd *cobra.Command, message, hint string) error {
	const (
		unknownLongPrefix  = "unknown flag: --"
		unknownShortPrefix = "unknown shorthand flag: "
		missingLongPrefix  = "flag needs an argument: --"
		missingShortPrefix = "flag needs an argument: "
		invalidPrefix      = "invalid argument "
		syntaxPrefix       = "bad flag syntax: "
	)

	switch {
	case strings.HasPrefix(message, unknownLongPrefix):
		option := "--" + strings.TrimPrefix(message, unknownLongPrefix)
		return NewError(fmt.Sprintf("unknown option %q for %q", option, cmd.CommandPath()), hint)
	case strings.HasPrefix(message, unknownShortPrefix):
		option, ok := parseShortOption(strings.TrimPrefix(message, unknownShortPrefix))
		if ok {
			return NewError(fmt.Sprintf("unknown option %q for %q", option, cmd.CommandPath()), hint)
		}
	case strings.HasPrefix(message, missingLongPrefix):
		option := "--" + strings.TrimPrefix(message, missingLongPrefix)
		return NewError(fmt.Sprintf("option %q requires a value", option), hint)
	case strings.HasPrefix(message, missingShortPrefix):
		option, ok := parseShortOption(strings.TrimPrefix(message, missingShortPrefix))
		if ok {
			return NewError(fmt.Sprintf("option %q requires a value", option), hint)
		}
	case strings.HasPrefix(message, invalidPrefix):
		return parseInvalidValue(cmd, strings.TrimPrefix(message, invalidPrefix), hint)
	case strings.HasPrefix(message, syntaxPrefix):
		return NewError(
			fmt.Sprintf("invalid option syntax %q", strings.TrimPrefix(message, syntaxPrefix)),
			hint,
		)
	}

	return nil
}

func parseInvalidValue(cmd *cobra.Command, message, hint string) error {
	const separator = " flag: "

	valueEnd := strings.Index(message, " for ")
	if valueEnd < 0 {
		return nil
	}
	value, err := strconv.Unquote(message[:valueEnd])
	if err != nil {
		return nil
	}

	message = message[valueEnd+len(" for "):]
	flagEnd := strings.Index(message, separator)
	if flagEnd < 0 {
		return nil
	}
	flagName, err := strconv.Unquote(message[:flagEnd])
	if err != nil {
		return nil
	}

	option := preferredOption(flagName)
	flag := cmd.Flag(strings.TrimPrefix(option, "--"))
	if flag != nil && flag.Value.Type() == "bool" {
		return NewError(
			fmt.Sprintf("invalid value %q for option %q", value, option),
			fmt.Sprintf(
				"Use %s without a value to enable it, or %s=false to disable it.",
				option,
				option,
			),
		)
	}

	cause := message[flagEnd+len(separator):]
	return NewError(
		fmt.Sprintf("invalid value %q for option %q: %s", value, option, cause),
		hint,
	)
}

func parseShortOption(message string) (string, bool) {
	quoted, _, ok := strings.Cut(message, " in -")
	if !ok {
		return "", false
	}
	name, err := strconv.Unquote(quoted)
	if err != nil || name == "" {
		return "", false
	}
	return "-" + name, true
}

func preferredOption(flagName string) string {
	if index := strings.LastIndex(flagName, "--"); index >= 0 {
		return strings.TrimSpace(flagName[index:])
	}
	return strings.TrimSpace(strings.SplitN(flagName, ",", 2)[0])
}
