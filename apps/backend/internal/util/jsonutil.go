// Package util provides shared utility functions for JSON/YAML repair.
package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PrepareJSON strips markdown wrappers, extracts the JSON payload, repairs common
// truncation, and rejects empty/non-JSON model outputs before callers unmarshal.
func PrepareJSON(raw, label string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s: empty model response", label)
	}

	extracted := ExtractJSON(trimmed)
	if extracted == "" {
		return "", fmt.Errorf("%s: no JSON payload found in model response", label)
	}
	if !strings.ContainsAny(extracted, "{[") {
		return "", fmt.Errorf("%s: no JSON payload found in model response: %s", label, preview(trimmed))
	}

	repaired := RepairJSON(extracted)
	if strings.TrimSpace(repaired) == "" {
		return "", fmt.Errorf("%s: JSON payload is empty after repair", label)
	}
	return repaired, nil
}

// ExtractJSON strips markdown code blocks and surrounding text, returning just the JSON.
func ExtractJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	// Strip markdown code blocks: ```json ... ``` or ``` ... ```
	if start := strings.Index(raw, "```"); start != -1 {
		inner := raw[start+3:]
		// Skip language tag (json, JSON, etc.)
		if idx := strings.Index(inner, "\n"); idx != -1 {
			inner = inner[idx+1:]
		}
		if end := strings.LastIndex(inner, "```"); end != -1 {
			raw = strings.TrimSpace(inner[:end])
		}
	}

	// Find first { or [ and last } or ]
	firstObj := strings.IndexAny(raw, "{[")
	lastObj := strings.LastIndexAny(raw, "}]")
	if firstObj != -1 && lastObj > firstObj {
		raw = raw[firstObj : lastObj+1]
	}

	return raw
}

// RepairJSON attempts to fix truncated JSON by closing open brackets/braces.
func RepairJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	// Try parsing as-is first
	if json.Valid([]byte(raw)) {
		return raw
	}

	// Find the last valid position by counting brackets
	openBraces := 0
	openBrackets := 0
	inString := false
	escaped := false

	for _, ch := range raw {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			openBraces++
		case '}':
			openBraces--
		case '[':
			openBrackets++
		case ']':
			openBrackets--
		}
	}

	// Remove trailing incomplete string if any
	if inString {
		// Find last unescaped quote and truncate there
		lastQuote := strings.LastIndex(raw, `"`)

		// Check if this quote is escaped
		slashCount := 0
		for i := lastQuote - 1; i >= 0 && raw[i] == '\\'; i-- {
			slashCount++
		}
		if slashCount%2 == 0 {
			// Not escaped, truncate at the quote
			raw = raw[:lastQuote]
		}
	}

	// Remove trailing comma or colon if present
	raw = strings.TrimRight(raw, " \t\n\r")
	if len(raw) > 0 {
		last := raw[len(raw)-1]
		if last == ',' || last == ':' {
			raw = raw[:len(raw)-1]
		}
	}

	// Close open brackets and braces
	var suffix strings.Builder
	for i := 0; i < openBrackets; i++ {
		suffix.WriteString("]")
	}
	for i := 0; i < openBraces; i++ {
		suffix.WriteString("}")
	}

	repaired := raw + suffix.String()

	// Verify the repair worked
	if json.Valid([]byte(repaired)) {
		return repaired
	}

	// If still invalid, return original (will fail parsing as before)
	return raw
}

func preview(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 240 {
		return s
	}
	return s[:240] + "..."
}
