package util

import (
	"fmt"
	"strings"
)


// PrepareYAML strips markdown wrappers and rejects empty model outputs before
// callers unmarshal.
func PrepareYAML(raw, label string) (string, error) {
	prepared := RepairYAML(raw)
	if strings.TrimSpace(prepared) == "" {
		return "", fmt.Errorf("%s: empty model response", label)
	}
	return prepared, nil
}
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

	// Fix bare heading: LLM sometimes outputs heading value without the key,
	// e.g. starting with "外景 草原 - 日间" instead of `heading: "外景 草原 - 日间"`.
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// If first non-empty line doesn't start with a YAML mapping key (word: ...),
		// treat the whole line as a bare heading value.
		if !looksLikeYAMLKey(trimmed) {
			escaped := strings.ReplaceAll(trimmed, `"`, `\"`)
			lines[i] = `heading: "` + escaped + `"`
		}
		break
	}
	raw = strings.Join(lines, "\n")
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

// looksLikeYAMLKey reports whether s starts with a YAML mapping key (word: ...).
func looksLikeYAMLKey(s string) bool {
	colon := strings.Index(s, ":")
	if colon <= 0 {
		return false
	}
	key := s[:colon]
	if len(key) == 0 {
		return false
	}
	for _, c := range key {
		if !isYAMLKeyRune(c) {
			return false
		}
	}
	return true
}

func isYAMLKeyRune(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-'
}
