package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"

	"github.com/fabula-studio/backend/internal/scene"
	"github.com/fabula-studio/backend/internal/tree"
	"github.com/fabula-studio/backend/internal/util"
)

// ScenePlannerAgent decides how leaf nodes map to screenplay scenes.
type ScenePlannerAgent struct {
	agent *llmagent.LLMAgent
}

const scenePlannerDesc = "Plans how story leaf nodes map to screenplay scenes"
const scenePlannerPrompt = `You are a screenplay structure specialist. You decide how story fragments should become scenes.

You will receive a list of leaf nodes (story fragments that have been analyzed and deemed ready for adaptation).
For each node, decide how it maps to scenes:

- scene_count: 0 = summary only (background/exposition, not shown on screen)
                1 = single scene
                2+ = multiple scenes (if location/time/conflict changes significantly within the fragment)
- purpose: The dramatic purpose of this scene in the overall story
- key_plot_points: Essential plot points that MUST be preserved
- omit_details: Novel details that can be cut for pacing

You may also recommend merging adjacent short nodes into a single scene if they share:
- Same location
- Same time
- Continuous action
- Same dramatic goal

Output a JSON array of scene plans, each with:
- id: "plan_NNN"
- source_node_ids: Array of leaf node IDs this plan covers
- scene_count: Number of scenes to generate
- purpose: Dramatic purpose
- location: Primary location
- time_frame: Primary time
- characters: Array of character names/IDs present
- key_plot_points: Must-keep story beats
- omit_details: Can-cut details
- sequence: Order in the full screenplay (1, 2, 3...)
- summary_only: If scene_count=0, provide the summary text here

Output ONLY valid JSON. No markdown fences, no commentary.`

// NewScenePlannerAgent creates the scene planning agent.
func NewScenePlannerAgent(modelName, apiKey, baseURL string) *ScenePlannerAgent {
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

	agt := llmagent.New("scene-planner",
		llmagent.WithModel(m),
		llmagent.WithDescription(scenePlannerDesc),
		llmagent.WithInstruction(scenePlannerPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &ScenePlannerAgent{agent: agt}
}
// PlanScenes generates scene plans for a set of leaf nodes.
func (a *ScenePlannerAgent) PlanScenes(ctx context.Context, leaves []*tree.StoryNode) ([]*scene.ScenePlan, error) {
	// Build a compact representation of leaves for the prompt
	type leafInfo struct {
		ID          string   `json:"id"`
		Summary     string   `json:"summary"`
		MainConflict string  `json:"main_conflict"`
		Characters  []string `json:"characters"`
		Location    string   `json:"location"`
		TimeFrame   string   `json:"time_frame"`
	}

	infos := make([]leafInfo, len(leaves))
	for i, l := range leaves {
		infos[i] = leafInfo{
			ID: l.ID, Summary: l.Summary, MainConflict: l.MainConflict,
			Characters: l.Characters, Location: l.Location, TimeFrame: l.TimeFrame,
		}
	}

	infosJSON, _ := json.Marshal(infos)
	prompt := fmt.Sprintf("Plan scenes for these leaf nodes:\n```json\n%s\n```", string(infosJSON))

	raw, err := Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	raw = util.RepairJSON(raw)

	var plans []*scene.ScenePlan
	if err := json.Unmarshal([]byte(raw), &plans); err != nil {
		return nil, fmt.Errorf("failed to parse scene plans JSON: %w\nraw: %s", err, raw)
	}
	return plans, nil
}
