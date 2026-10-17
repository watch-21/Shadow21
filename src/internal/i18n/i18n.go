package i18n

import (
	_ "embed"
	"encoding/json"
)

//go:embed de.json
var data []byte
var language = "en"
var german = func() map[string]string { m := map[string]string{}; _ = json.Unmarshal(data, &m); return m }()

func Set(l string)     { language = l }
func Language() string { return language }
func T(s string) string {
	if language == "de" {
		if t, ok := german[s]; ok {
			return t
		}
	}
	return s
}
