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
	"github.com/fabula-studio/backend/internal/util"
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

const chiefEditorDesc = "审查并修订完整剧本，确保一致性、节奏和质量"
const chiefEditorPrompt = `你是一名首席剧本编辑。你审查完整的剧本并修复问题。

	你将收到完整的 YAML 格式剧本，以及可选的 generation artifacts（source index、story beats、scene plan、graph snapshot）。审查以下方面：
	1. 故事连贯性：事件是否按逻辑顺序排列？
	2. 角色一致性：角色的对白和行为是否一致？
	3. 关系一致性：角色关系在不同场景间是否一致？
	4. 信息泄露：是否有场景提前揭示了 artifacts 中还不应出现的信息？
	5. 场景节奏：场景是否过于稀疏或过于密集？
	6. 内容质量：动作和对白是否服务于戏剧目的？
	7. YAML 结构：所有必需字段是否存在且合法？

输出一个 JSON 对象，包含：
- screenplay: 完整的修订后剧本（完整 YAML 结构）
- issues: 发现的问题数组（即使已修复）
- changes: 你所做修改的描述数组

如果无需修改，原样返回剧本。

规则：
	- 你的修订只能基于剧本和 generation artifacts 中已有的信息
	- 不得添加未从原始素材或 artifacts 中推导出的新情节点
	- 可以为一致性调整对白，但不得改变故事或 source grounding

只输出合法 JSON，不要 markdown 代码块，不要额外注释。`

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
func (a *ChiefEditorAgent) ReviewAndRevise(ctx context.Context, sp *schema.Screenplay, artifacts *schema.GenerationArtifacts) (*EditResult, error) {
	spYAML, err := yaml.Marshal(sp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal screenplay: %w", err)
	}
	artifactsJSON, err := json.Marshal(artifacts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal generation artifacts: %w", err)
	}

	prompt := fmt.Sprintf("Review and revise this screenplay using the generation artifacts as source grounding.\n\nGeneration artifacts:\n```json\n%s\n```\n\nScreenplay:\n```yaml\n%s\n```", string(artifactsJSON), string(spYAML))

	raw, err := Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	raw, err = util.PrepareJSON(raw, "editor output")
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
