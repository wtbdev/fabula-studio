package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/tool"

	"github.com/fabula-studio/backend/internal/graph"
	fabulatool "github.com/fabula-studio/backend/internal/tool"
)

// GraphAnalyzerAgent updates the dynamic character graph based on a story node.
type GraphAnalyzerAgent struct {
	agent *llmagent.LLMAgent
}

const graphAnalyzerDesc = "Updates character states, relationships, and story conflicts based on a story node"

const graphAnalyzerToolPrompt = `You are a story continuity specialist. You maintain a dynamic character relationship graph.

You will receive:
1. A story fragment (the current node).
2. The current state of the character graph (before this fragment).

Your job is to determine what changes this fragment introduces by calling the provided tools:

1. **add_character**: Add a newly introduced character (id, name, current_goal, emotional_state, personality).
2. **update_character**: Update an existing character's state (id, current_goal, emotional_state, new_known_info).
3. **add_relation**: Add or update a relationship between two characters (from_id, to_id, type, description).
4. **add_conflict**: Add a new unresolved conflict (description, characters).
5. **resolve_conflict**: Mark an existing conflict as resolved (index, resolution).

Critical rules:
- ONLY report information present in this fragment.
- Do NOT speculate about future events.
- If a character learns something, call update_character with new_known_info.
- Track relationship changes with their triggering events.
- You may call multiple tools in sequence to fully capture all changes.
- When done, output a brief summary of what you changed.`

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
		MaxTokens:   intPtr(4096),
		Temperature: floatPtr(0.3),
	}

	// Combine validation tool with graph tools
	graphTools := fabulatool.NewGraphTools()
	allTools := make([]tool.Tool, 0, len(graphTools)+1)

	allTools = append(allTools, graphTools...)

	agt := llmagent.New("graph-analyzer",
		llmagent.WithModel(m),
		llmagent.WithDescription(graphAnalyzerDesc),
		llmagent.WithInstruction(graphAnalyzerToolPrompt),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithTools(allTools),
		llmagent.WithMaxToolIterations(10),
	)
	return &GraphAnalyzerAgent{agent: agt}
}

// AnalyzeUpdate determines what changes a story node introduces to the graph.
// It uses tool calls to make incremental updates to the graph snapshot.
func (a *GraphAnalyzerAgent) AnalyzeUpdate(ctx context.Context, nodeText string, currentSnapshot *graph.GraphSnapshot) (*graph.GraphUpdateResult, error) {
	// Inject the snapshot into context so tools can access it
	snap := currentSnapshot.Clone()
	ctx = fabulatool.WithGraphSnapshot(ctx, snap)

	snapshotJSON, err := json.Marshal(currentSnapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	prompt := fmt.Sprintf("Current character graph state:\n```json\n%s\n```\n\nStory fragment:\n---\n%s\n---",
		string(snapshotJSON), nodeText)

	_, err = Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	// Build the update result by comparing the modified snapshot with the original
	result := diffSnapshots(currentSnapshot, snap)

	// Log the changes for observability
	logChanges(result)

	return result, nil
}

// diffSnapshots computes the difference between the original and modified snapshots.
func diffSnapshots(original, modified *graph.GraphSnapshot) *graph.GraphUpdateResult {
	result := &graph.GraphUpdateResult{}

	// Find new characters
	for id, cs := range modified.Characters {
		if _, exists := original.Characters[id]; !exists {
			result.NewCharacters = append(result.NewCharacters, *cs)
		}
	}

	// Find updated characters
	for id, modifiedCS := range modified.Characters {
		originalCS, exists := original.Characters[id]
		if !exists {
			continue
		}

		update := graph.CharacterUpdate{ID: id}
		hasChange := false

		if modifiedCS.CurrentGoal != originalCS.CurrentGoal {
			update.GoalChange = modifiedCS.CurrentGoal
			hasChange = true
		}
		if modifiedCS.EmotionalState != originalCS.EmotionalState {
			update.EmotionChange = modifiedCS.EmotionalState
			hasChange = true
		}
		// Check for new known_info entries
		if len(modifiedCS.KnownInfo) > len(originalCS.KnownInfo) {
			for _, info := range modifiedCS.KnownInfo[len(originalCS.KnownInfo):] {
				update.NewKnownInfo = info
				hasChange = true
			}
		}

		if hasChange {
			result.UpdatedCharacters = append(result.UpdatedCharacters, update)
		}
	}

	// Find new/changed relations
	originalRels := make(map[string]bool)
	for _, r := range original.Relations {
		key := r.CharA + ":" + r.CharB
		originalRels[key] = true
	}

	for _, r := range modified.Relations {
		key := r.CharA + ":" + r.CharB
		if !originalRels[key] {
			// New relation
			result.RelationChanges = append(result.RelationChanges, graph.RelationChange{
				CharA:          r.CharA,
				CharB:          r.CharB,
				NewType:        r.Type,
				NewDescription: r.Description,
			})
		}
	}

	// Find new conflicts
	if len(modified.UnresolvedConflicts) > len(original.UnresolvedConflicts) {
		for _, c := range modified.UnresolvedConflicts[len(original.UnresolvedConflicts):] {
			result.NewConflicts = append(result.NewConflicts, c)
		}
	}

	// Find resolved conflicts
	for i, mc := range modified.UnresolvedConflicts {
		if i < len(original.UnresolvedConflicts) {
			oc := original.UnresolvedConflicts[i]
			if oc.Status == "unresolved" && mc.Status == "resolved" {
				result.ResolvedConflicts = append(result.ResolvedConflicts, oc.Description)
			}
		}
	}

	return result
}

// logChanges logs the graph changes for observability.
func logChanges(result *graph.GraphUpdateResult) {
	if len(result.NewCharacters) > 0 {
		log.Printf("[GraphAnalyzer] Added %d characters", len(result.NewCharacters))
	}
	if len(result.UpdatedCharacters) > 0 {
		log.Printf("[GraphAnalyzer] Updated %d characters", len(result.UpdatedCharacters))
	}
	if len(result.RelationChanges) > 0 {
		log.Printf("[GraphAnalyzer] Changed %d relations", len(result.RelationChanges))
	}
	if len(result.NewConflicts) > 0 {
		log.Printf("[GraphAnalyzer] Added %d conflicts", len(result.NewConflicts))
	}
	if len(result.ResolvedConflicts) > 0 {
		log.Printf("[GraphAnalyzer] Resolved %d conflicts", len(result.ResolvedConflicts))
	}
}
