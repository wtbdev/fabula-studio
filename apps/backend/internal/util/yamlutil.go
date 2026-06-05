package util

import (
	"strings"
)

// RepairYAML attempts to fix common YAML formatting issues from LLM output.
func RepairYAML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	// Remove markdown code fences if present
	if strings.HasPrefix(raw, "```yaml") {
		raw = strings.TrimPrefix(raw, "```yaml")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}

	lines := strings.Split(raw, "\n")
	var fixed []string

	for _, line := range lines {
		// Fix lines where text field has trailing unquoted content
		// e.g., text: "quoted" trailing text
		if idx := strings.Index(line, `text: "`); idx >= 0 {
			rest := line[idx+7:] // after 'text: "'
			// Find the closing quote
			closeIdx := strings.Index(rest, `"`)
			if closeIdx >= 0 {
				// Check if there's content after the closing quote
				afterQuote := rest[closeIdx+1:]
				trimmedAfter := strings.TrimSpace(afterQuote)
				if trimmedAfter != "" && !strings.HasPrefix(trimmedAfter, "#") {
					// There's extra content - remove it
					line = line[:idx+7] + rest[:closeIdx+1]
				}
			}
		}

		// Fix unclosed quotes in text fields
		if strings.Contains(line, `text: "`) && !strings.HasSuffix(strings.TrimSpace(line), `"`) {
			trimmed := strings.TrimSpace(line)
			if strings.Count(trimmed, `"`)%2 != 0 {
				// Odd number of quotes - add closing quote
				line = line + `"`
			}
		}

		fixed = append(fixed, line)
	}

	return strings.Join(fixed, "\n")
}
