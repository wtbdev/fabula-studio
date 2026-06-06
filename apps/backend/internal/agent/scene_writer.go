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

const sceneWriterDesc = "根据场景上下文包，写出单个 YAML 格式的剧本场景"
const sceneWriterPrompt = `你是一名专业编剧。你将收到一个场景上下文包（SCENE CONTEXT PACKAGE），包含：
- 场景规划（目的、地点、时间、角色、关键情节点）
- 小说的原始文本
- 角色信息与关系
- 已知事实和未解决的冲突

你的任务：将其转化为单个 YAML 场景，遵循剧本格式。

规则：
1. 场景标题格式：内景/外景 地点 - 时间
2. 动作描述行：现在时，展示而非讲述，只描写可见/可闻的内容
3. 对白：忠实于每个角色的语气。括号提示（parenthetical）少用
4. 绝不包含 forbidden_info 中的信息——它们是未来的剧透
5. 保留所有 key_plot_points
6. 可以省略 omit_details 中列出的细节
7. 内心想法必须转化为可见的动作或对白

YAML 场景结构：
  id: "scene_NNN"
  sequence: N
  heading: "内景/外景 地点 - 时间"
  setting:
    location: "..."
    time: "..."
    interior: true/false
  synopsis: "一句话场景概要"
  characters_present: ["char_id_1", "char_id_2"]
  content:
    - type: action
      text: "..."
    - type: dialogue
      character: "角色名"
      parenthetical: "(可选)"
      text: "..."
    - type: transition
      text: "切至："

只输出该单个场景的 YAML，不要 markdown 代码块，不要额外注释。`

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
