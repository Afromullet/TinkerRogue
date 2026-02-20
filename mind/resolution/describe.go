package resolution

import "strings"

// FormatDescription joins non-empty reward description parts into a single string.
// Example: ["150 gold", "75 XP"] -> "150 gold, 75 XP"
func FormatDescription(parts []string) string {
	return strings.Join(parts, ", ")
}
