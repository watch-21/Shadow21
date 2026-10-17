package main

import (
	"github.com/spf13/cobra"
	"gitlab.com/parrotsec/packages/anonsurf/internal/i18n"
	"strings"
)

func configureRootHelp(root *cobra.Command) {
	cobra.AddTemplateFunc("localizeFlags", func(s string) string {
		if i18n.Language() == "de" {
			return strings.ReplaceAll(s, "(default ", "(Standard ")
		}
		return s
	})
	template := helpTemplate
	if i18n.Language() == "de" {
		template = strings.NewReplacer("Usage:", "Verwendung:", "Commands:", "Befehle:", "Global Flags:", "Globale Optionen:", "Flags:", "Optionen:", "Examples:", "Beispiele:").Replace(template)
	}
	root.SetHelpTemplate(template)
}
