package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"

	"github.com/fabula-studio/backend/internal/tree"
)

// NodeAnalyzerAgent analyzes individual story nodes.
type NodeAnalyzerAgent struct {
	agent *llmagent.LLMAgent
}

// NodeAnalysisResult is the structured output from node analysis.
type NodeAnalysisResult struct {
	Summary      string       `json:"summary"`
	MainConflict string       `json:"main_conflict"`
	Characters   []string     `json:"characters"`
	Events       []tree.Event `json:"events"`
	Location     string       `json:"location"`
	TimeFrame    string       `json:"time_frame"`
	IsComplete   bool         `json:"is_complete"`
	Decision     string       `json:"decision"`
	SplitReason  string       `json:"split_reason,omitempty"`
}

const nodeAnalyzerDesc = "Analyzes a single story node and extracts structure, decides processing path"
const nodeAnalyzerPrompt = `You are a professional literary analyst specializing in novel-to-screenplay adaptation.

You will receive a text fragment from a novel. Analyze it and output a JSON object with these fields:

1. summary: One-paragraph summary of what happens in this fragment.
2. main_conflict: The central dramatic conflict or tension (one sentence).
3. characters: List of character names that appear.
4. events: List of events, each with:
   - description: What happens
   - characters: Who is involved
   - impact: How it affects the story
5. location: Where the action takes place.
6. time_frame: When the action takes place.
7. is_complete: true if this fragment tells a complete mini-story (has setup, action, and resolution/consequence). false if it feels cut off mid-action or mid-dialogue.
8. decision: One of:
   - "keep": Fragment is complete and manageable in length, ready for scene adaptation.
   - "split": Fragment is too long or complex, needs to be divided further.
   - "merge_right": Fragment is cut off and needs the next fragment to complete the current event.
   - "summarize_only": Fragment has informational value but is not suitable for direct scene adaptation (e.g., exposition, backstory, internal monologue).
   - "discard": Fragment has no significant value for the screenplay.
9. split_reason: If decision is "split", explain why (e.g., "multiple location changes", "too many characters").

Important:
- A fragment ending mid-dialogue or mid-action should be "merge_right".
- Background exposition, worldbuilding, or long internal monologue should be "summarize_only".
- Only use "discard" for truly redundant content.

Output ONLY valid JSON. No markdown fences, no commentary.`

// NewNodeAnalyzerAgent creates the node analysis agent.
func NewNodeAnalyzerAgent(modelName, apiKey, baseURL string) *NodeAnalyzerAgent {
	opts := []openai.Option{}
	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelName, opts...)

	genConfig := model.GenerationConfig{
		MaxTokens:   intPtr(2000),
		Temperature: floatPtr(0.3),
	}

	agt := llmagent.New("node-analyzer",
		llmagent.WithModel(m),
		llmagent.WithDescription(nodeAnalyzerDesc),
		llmagent.WithInstruction(nodeAnalyzerPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &NodeAnalyzerAgent{agent: agt}
}

// Analyze sends a story node for analysis and returns structured results.
func (a *NodeAnalyzerAgent) Analyze(ctx context.Context, node *tree.StoryNode, leftNeighborSummary string) (*NodeAnalysisResult, error) {
	prompt := fmt.Sprintf("Analyze this novel fragment:\n\n---\n%s\n---", node.TextContent)
	if leftNeighborSummary != "" {
		prompt = fmt.Sprintf("Context: The previous fragment ended with: %q\n\n%s", leftNeighborSummary, prompt)
	}

	raw, err := RunAgent(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	var result NodeAnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse node analysis JSON: %w\nraw: %s", err, raw)
	}
	return &result, nil
}
