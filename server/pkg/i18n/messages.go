package i18n

// Messages is the centralized message map for all server i18n.
// Keys follow a semantic dot-notation convention.
//
// Each language has its own file (zh.go, en.go, ja.go, fr.go, de.go,
// es.go, ko.go, ru.go) that registers translations via init() +
// registerLang().
var Messages = map[string]map[Lang]string{}

// registerLang registers all translations for a single language.
// Called from init() in each language file.
func registerLang(lang Lang, entries map[string]string) {
	for key, text := range entries {
		if Messages[key] == nil {
			Messages[key] = map[Lang]string{}
		}
		Messages[key][lang] = text
	}
}
