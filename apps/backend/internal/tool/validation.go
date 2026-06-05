// Package tool provides custom tools for the fabula-studio agents.
package tool

import (
	"context"
	"encoding/json"

	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"

	"github.com/fabula-studio/backend/internal/util"
)

// ValidateOutputRequest is the input for the validate_output tool.
type ValidateOutputRequest struct {
	Format  string `json:"format" jsonschema:"description=Output format: json or yaml"`
	Content string `json:"content" jsonschema:"description=The content to validate"`
}

// ValidateOutputResponse is the output from the validate_output tool.
type ValidateOutputResponse struct {
	Valid   bool   `json:"valid"`
	Repaired string `json:"repaired,omitempty"`
	Note    string `json:"note,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewValidateOutputTool creates a tool that validates and optionally repairs JSON/YAML output.
func NewValidateOutputTool() tool.CallableTool {
	return function.NewFunctionTool(
		validateOutput,
		function.WithName("validate_output"),
		function.WithDescription("Validates JSON or YAML content. Returns repaired content if the original has minor issues."),
	)
}

func validateOutput(_ context.Context, req ValidateOutputRequest) (ValidateOutputResponse, error) {
	switch req.Format {
	case "json":
		return validateJSON(req.Content), nil
	case "yaml":
		return validateYAML(req.Content), nil
	default:
		return ValidateOutputResponse{
			Valid: false,
			Error: "unsupported format: " + req.Format + ". Use 'json' or 'yaml'",
		}, nil
	}
}

func validateJSON(content string) ValidateOutputResponse {
	// Strip markdown fences if present
	content = stripMarkdownFences(content, "json")

	if json.Valid([]byte(content)) {
		return ValidateOutputResponse{Valid: true}
	}
	repaired := util.RepairJSON(content)
	if json.Valid([]byte(repaired)) {
		return ValidateOutputResponse{
			Valid:    true,
			Repaired: repaired,
			Note:     "auto-repaired truncated or malformed JSON",
		}
	}

	return ValidateOutputResponse{
		Valid: false,
		Error: "invalid JSON content",
	}
}

func validateYAML(content string) ValidateOutputResponse {
	// Strip markdown fences if present
	content = stripMarkdownFences(content, "yaml")

	var parsed interface{}
	if err := yaml.Unmarshal([]byte(content), &parsed); err == nil {
		return ValidateOutputResponse{Valid: true}
	}

	// Try repair
	repaired := util.RepairYAML(content)
	if err := yaml.Unmarshal([]byte(repaired), &parsed); err == nil {
		return ValidateOutputResponse{
			Valid:    true,
			Repaired: repaired,
			Note:     "auto-repaired YAML formatting issues",
		}
	}

	return ValidateOutputResponse{
		Valid: false,
		Error: "invalid YAML content",
	}
}

func stripMarkdownFences(content, lang string) string {
	prefix := "```" + lang
	if len(content) > len(prefix) && content[:len(prefix)] == prefix {
		content = content[len(prefix):]
		if idx := len(content) - 3; idx >= 0 && content[idx:] == "```" {
			content = content[:idx]
		}
	}
	return content
}
