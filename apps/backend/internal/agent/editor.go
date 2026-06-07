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

// ChiefEditorAgent reviews the complete screenplay without rewriting it.
type ChiefEditorAgent struct {
	agent *llmagent.LLMAgent
}

// EditResult contains the editor's review findings and recommended changes.
type EditResult struct {
	Issues  []editorIssue  `json:"issues"`
	Changes []editorChange `json:"changes"`
}

const chiefEditorDesc = "审查完整剧本，提出一致性、节奏和质量建议"
const chiefEditorPrompt = `你是一名首席剧本编辑。你只审查完整剧本并提出问题与修改建议，不直接改写剧本。

	你将收到完整的 YAML 格式剧本，以及可选的 generation artifacts（source index、story beats、scene plan、graph snapshot）。审查以下方面：
	1. 故事连贯性：事件是否按逻辑顺序排列？
	2. 角色一致性：角色的对白和行为是否一致？
	3. 关系一致性：角色关系在不同场景间是否一致？
	4. 信息泄露：是否有场景提前揭示了 artifacts 中还不应出现的信息？
	5. 场景节奏：场景是否过于稀疏或过于密集？
	6. 内容质量：动作和对白是否服务于戏剧目的？
	7. YAML 结构：所有必需字段是否存在且合法？

输出一个 JSON 对象，只包含：
- issues: 发现的问题数组
- changes: 建议的修改或改进数组

如果无需修改建议，返回空数组。

规则：
	- 只做审查和建议，不输出 screenplay、revised_screenplay 或任何替换剧本字段
	- 你的建议只能基于剧本和 generation artifacts 中已有的信息
	- 不得添加未从原始素材或 artifacts 中推导出的新情节点
	- 可以建议为一致性调整对白，但不得改变故事或 source grounding

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

// editorIssue is a single review issue returned by the editor agent.
type editorIssue struct {
	Severity         string   `json:"severity"`
	Description      string   `json:"description"`
	AffectedLocations []string `json:"affectedLocations,omitempty"`
}

// editorChange is a single suggested change returned by the editor agent.
type editorChange struct {
	Type             string   `json:"type"`
	Description      string   `json:"description"`
	AffectedLocations []string `json:"affectedLocations,omitempty"`
}

// editorOutput is the JSON structure returned by the editor agent.
// We decode issues/changes as RawMessage to handle both string and object shapes.
type editorOutput struct {
	Issues  json.RawMessage `json:"issues"`
	Changes json.RawMessage `json:"changes"`
}
// decodeIssues converts issues raw json into []editorIssue, accepting both
// plain strings and objects.
func decodeIssues(raw json.RawMessage) []editorIssue {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	// Try objects first
	var objs []editorIssue
	if err := json.Unmarshal(raw, &objs); err == nil {
		return objs
	}
	// Fall back: plain strings
	var strs []string
	if err := json.Unmarshal(raw, &strs); err != nil {
		// Return as a single issue with the raw string
		return []editorIssue{{Description: string(raw)}}
	}
	issues := make([]editorIssue, len(strs))
	for i, s := range strs {
		issues[i] = editorIssue{Description: s}
	}
	return issues
}
// decodeChanges converts changes raw json into []editorChange.
func decodeChanges(raw json.RawMessage) []editorChange {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var objs []editorChange
	if err := json.Unmarshal(raw, &objs); err == nil {
		return objs
	}
	var strs []string
	if err := json.Unmarshal(raw, &strs); err != nil {
		return []editorChange{{Description: string(raw)}}
	}
	changes := make([]editorChange, len(strs))
	for i, s := range strs {
		changes[i] = editorChange{Description: s}
	}
	return changes
}
// ReviewAndRevise reviews the complete screenplay and returns findings only.
func (a *ChiefEditorAgent) ReviewAndRevise(ctx context.Context, sp *schema.Screenplay, artifacts *schema.GenerationArtifacts) (*EditResult, error) {
	spYAML, err := yaml.Marshal(sp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal screenplay: %w", err)
	}
	artifactsJSON, err := json.Marshal(artifacts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal generation artifacts: %w", err)
	}
	prompt := fmt.Sprintf("Review this screenplay using the generation artifacts as source grounding. Return only review issues and recommended changes; do not return a replacement screenplay.\n\nGeneration artifacts:\n```json\n%s\n```\n\nScreenplay:\n```yaml\n%s\n```", string(artifactsJSON), string(spYAML))

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

	result := &EditResult{
		Issues:  decodeIssues(output.Issues),
		Changes: decodeChanges(output.Changes),
	}

	return result, nil
}
