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
	"github.com/fabula-studio/backend/internal/scene"
	"github.com/fabula-studio/backend/internal/util"
)

// SceneWriterAgent generates a single YAML scene from a context package.
type SceneWriterAgent struct {
	agent *llmagent.LLMAgent
}

const sceneWriterDesc = "Writes a single screenplay scene in YAML format from a bounded context package"
const sceneWriterPrompt = `You are a professional screenwriter. You will receive a SCENE CONTEXT PACKAGE containing:
- Scene plan (purpose, location, time, characters, key plot points)
- Source text from the novel
- Character information and relationships
- Known facts and unresolved conflicts

Your task: Convert this into a single YAML scene following the screenplay format.

Rules:
1. Scene heading format: INT./EXT. LOCATION - TIME
2. Action lines: present tense, show-don't-tell, describe only what is visible/audible.
3. Dialogue: faithful to each character's voice. Use parentheticals sparingly.
4. NEVER include information from forbidden_info — these are future spoilers.
5. Preserve all key_plot_points.
6. You MAY omit details listed in omit_details.
7. Internal thoughts must become visible actions or dialogue.

The YAML scene structure:
  id: "scene_NNN"
  sequence: N
  heading: "INT./EXT. LOCATION - TIME"
  setting:
    location: "..."
    time: "..."
    interior: true/false
  synopsis: "One-sentence scene summary"
  characters_present: ["char_id_1", "char_id_2"]
  content:
    - type: action
      text: "..."
    - type: dialogue
      character: "Character Name"
      parenthetical: "(optional)"
      text: "..."
    - type: transition
      text: "CUT TO:"

Output ONLY the YAML for this single scene. No markdown fences, no extra commentary.`

// NewSceneWriterAgent creates the scene writing agent.
func NewSceneWriterAgent(modelName, apiKey, baseURL string) *SceneWriterAgent {
	opts := []openai.Option{}
	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelName, opts...)
	genConfig := model.GenerationConfig{
		MaxTokens:   intPtr(8192),
		Temperature: floatPtr(0.7),
	}

	agt := llmagent.New("scene-writer",
		llmagent.WithModel(m),
		llmagent.WithDescription(sceneWriterDesc),
		llmagent.WithInstruction(sceneWriterPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &SceneWriterAgent{agent: agt}
}

// WriteScene generates a single scene from a context package.
func (a *SceneWriterAgent) WriteScene(ctx context.Context, sceneCtx *scene.SceneContext) (*schema.Scene, error) {
	ctxJSON, err := json.Marshal(sceneCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scene context: %w", err)
	}

	prompt := fmt.Sprintf("Generate the YAML scene for this context package:\n```json\n%s\n```", string(ctxJSON))

	raw, err := Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	raw = util.RepairYAML(raw)

	var sc schema.Scene
	if err := yaml.Unmarshal([]byte(raw), &sc); err != nil {
		return nil, fmt.Errorf("failed to parse scene YAML: %w\nraw: %s", err, raw)
	}
	return &sc, nil
}
