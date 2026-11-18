package csvexport

import "strings"

// Text keeps user-controlled text from being interpreted as a spreadsheet
// formula. The CSV writer still handles separators, quotes and newlines.
func Text(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
