package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"github.com/fabula-studio/backend/internal/tree"
	"github.com/fabula-studio/backend/internal/util"
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

const nodeAnalyzerDesc = "分析单个故事节点，提取结构，决定处理路径"
const nodeAnalyzerPrompt = `你是一名专业文学分析师，专精于小说到剧本的改编。

你将收到一段小说文本片段。分析它并输出一个 JSON 对象，包含以下字段：

1. summary: 该片段内容的一段话摘要
2. main_conflict: 核心戏剧冲突或张力（一句话）
3. characters: 出现的人物名称列表
4. events: 事件列表，每个事件包含：
   - description: 发生了什么
   - characters: 涉及的人物
   - impact: 对故事的影响
5. location: 故事发生地点
6. time_frame: 故事发生时间
7. is_complete: true 表示该片段讲述了一个完整的微故事（有起承转合），false 表示感觉被截断了
8. decision: 以下之一：
   - "keep": 片段完整且长度适中，可直接用于场景改编
   - "split": 片段太长或太复杂，需要进一步拆分
   - "merge_right": 片段被截断，需要与下一片段合并才能构成完整事件
   - "summarize_only": 片段有价值但不适合直接改编为场景（如背景交代、内心独白）
   - "discard": 片段对剧本没有重要价值
9. split_reason: 如果 decision 为 "split"，说明原因（如"多次换景"、"角色太多"）

重要：
- 对话中途或动作中途结束的片段应标记为 "merge_right"
- 背景交代、世界观构建或长篇内心独白应标记为 "summarize_only"
- 只有真正冗余的内容才标记为 "discard"

只输出合法 JSON，不要 markdown 代码块，不要额外注释。`

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

	raw, err := Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}
	raw, err = util.PrepareJSON(raw, "node analysis output")
	if err != nil {
		return nil, err
	}

	var result NodeAnalysisResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse node analysis JSON: %w\nraw: %s", err, raw)
	}
	return &result, nil
}
