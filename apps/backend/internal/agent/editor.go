package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"

	"github.com/fabula-studio/backend/internal/schema"
)

// ChiefEditorAgent reviews and revises the complete screenplay.
type ChiefEditorAgent struct {
	agent *llmagent.LLMAgent
}

// EditResult contains the revised screenplay and a report of changes.
type EditResult struct {
	Screenplay *schema.Screenplay `json:"-"`
	Issues     []string           `json:"issues"`
	Changes    []string           `json:"changes"`
}

const chiefEditorDesc = "Reviews and revises a complete screenplay for consistency, pacing, and quality"
const chiefEditorPrompt = `You are a chief screenplay editor. You review a complete screenplay and fix issues.

You will receive the full screenplay in YAML format. Review for:
1. Story continuity: Are events in logical order?
2. Character consistency: Do characters speak and act consistently?
3. Relationship consistency: Do relationships match across scenes?
4. Information leaks: Does any scene reveal future information it shouldn't?
5. Scene pacing: Are scenes too sparse or too dense?
6. Content quality: Do actions and dialogue serve the dramatic purpose?
7. YAML structure: Are all required fields present and valid?

Output a JSON object with:
- screenplay: The complete revised screenplay (full YAML structure)
- issues: Array of issues you found (even if fixed)
- changes: Array of descriptions of what you changed

If no changes needed, return the original screenplay unchanged.

Rules:
- You may only base revisions on information already in the screenplay.
- You may NOT add new plot points not derived from the source material.
- You may adjust dialogue for consistency but not change the story.

Output ONLY valid JSON. No markdown fences, no commentary.`

// NewChiefEditorAgent creates the chief editor agent.
func NewChiefEditorAgent(modelName, apiKey, baseURL string) *ChiefEditorAgent {
	opts := []openai.Option{}
	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelName, opts...)
	genConfig := model.GenerationConfig{
		MaxTokens:   intPtr(8000),
		Temperature: floatPtr(0.4),
	}

	agt := llmagent.New("chief-editor",
		llmagent.WithModel(m),
		llmagent.WithDescription(chiefEditorDesc),
		llmagent.WithInstruction(chiefEditorPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &ChiefEditorAgent{agent: agt}
}

// editorOutput is the JSON structure returned by the editor agent.
type editorOutput struct {
	ScreenplayRaw json.RawMessage `json:"screenplay"`
	Issues        []string        `json:"issues"`
	Changes       []string        `json:"changes"`
}

// ReviewAndRevise reviews the complete screenplay and returns a revised version.
func (a *ChiefEditorAgent) ReviewAndRevise(ctx context.Context, sp *schema.Screenplay) (*EditResult, error) {
	spYAML, err := yaml.Marshal(sp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal screenplay: %w", err)
	}

	prompt := fmt.Sprintf("Review and revise this screenplay:\n```yaml\n%s\n```", string(spYAML))

	raw, err := RunAgent(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	var output editorOutput
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		return nil, fmt.Errorf("failed to parse editor output JSON: %w\nraw: %s", err, raw)
	}

	// The editor returns the screenplay as a nested JSON/YAML structure.
	// Try to parse it from the raw JSON.
	result := &EditResult{
		Issues:  output.Issues,
		Changes: output.Changes,
	}

	if len(output.ScreenplayRaw) > 0 {
		var revised schema.Screenplay
		// Try JSON first
		if err := json.Unmarshal(output.ScreenplayRaw, &revised); err == nil {
			result.Screenplay = &revised
		} else {
			// Try YAML
			if err := yaml.Unmarshal(output.ScreenplayRaw, &revised); err == nil {
				result.Screenplay = &revised
			}
		}
	}

	if result.Screenplay == nil {
		// Fallback: return original if parsing failed
		result.Screenplay = sp
		result.Issues = append(result.Issues, "editor output could not be parsed, original screenplay retained")
	}

	return result, nil
}
