package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"

	"github.com/fabula-studio/backend/internal/graph"
)

// GraphAnalyzerAgent updates the dynamic character graph based on a story node.
type GraphAnalyzerAgent struct {
	agent *llmagent.LLMAgent
}

const graphAnalyzerDesc = "Updates character states, relationships, and story conflicts based on a story node"
const graphAnalyzerPrompt = `You are a story continuity specialist. You maintain a dynamic character relationship graph.

You will receive:
1. A story fragment (the current node).
2. The current state of the character graph (before this fragment).

Your job is to determine what changes this fragment introduces. Output a JSON object:

1. new_characters: Array of newly introduced characters, each with:
   - id: Generate a char ID (e.g., "char_001")
   - name: Character name
   - current_goal: What this character wants right now
   - emotional_state: Current emotional state
   - personality: Array of personality traits

2. updated_characters: Array of changes to existing characters, each with:
   - id: Existing character ID
   - goal_change: New goal (if changed)
   - emotion_change: New emotional state (if changed)
   - new_known_info: New information the character learned

3. relation_changes: Array of relationship changes, each with:
   - char_a: Character ID or name
   - char_b: Character ID or name
   - new_type: New relationship type (if changed)
   - new_description: New relationship description (if changed)
   - trigger_event: What caused this change

4. new_conflicts: Array of new conflicts introduced, each with:
   - description: What the conflict is
   - involved: Array of character IDs/names involved
   - status: "unresolved"

5. resolved_conflicts: Array of conflict descriptions that were resolved in this fragment.

6. new_foreshadows: Array of new foreshadowing elements, each with:
   - description: What is foreshadowed
   - introduced_in: Leave empty (will be filled by the system)

Critical rules:
- ONLY report information present in this fragment.
- Do NOT speculate about future events.
- If a character learns something, record it in known_info.
- Track relationship changes with their triggering events.

Output ONLY valid JSON. No markdown fences, no commentary.`

// NewGraphAnalyzerAgent creates the graph analysis agent.
func NewGraphAnalyzerAgent(modelName, apiKey, baseURL string) *GraphAnalyzerAgent {
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

	agt := llmagent.New("graph-analyzer",
		llmagent.WithModel(m),
		llmagent.WithDescription(graphAnalyzerDesc),
		llmagent.WithInstruction(graphAnalyzerPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &GraphAnalyzerAgent{agent: agt}
}

// AnalyzeUpdate determines what changes a story node introduces to the graph.
func (a *GraphAnalyzerAgent) AnalyzeUpdate(ctx context.Context, nodeText string, currentSnapshot *graph.GraphSnapshot) (*graph.GraphUpdateResult, error) {
	snapshotJSON, err := json.Marshal(currentSnapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	prompt := fmt.Sprintf("Current character graph state:\n```json\n%s\n```\n\nStory fragment:\n---\n%s\n---",
		string(snapshotJSON), nodeText)

	raw, err := RunAgent(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	var result graph.GraphUpdateResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse graph update JSON: %w\nraw: %s", err, raw)
	}
	return &result, nil
}
