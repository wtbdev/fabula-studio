// Package agent provides trpc-agent-go based AI agents for novel analysis
// and screenplay generation.
package agent

import (
	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
)

// NovelAgent wraps a pair of trpc-agent-go LLMAgents for the conversion pipeline.
// Analyzer extracts structure from raw novel text; Writer produces screenplay YAML.
type NovelAgent struct {
	Analyzer *llmagent.LLMAgent
	Writer   *llmagent.LLMAgent
}

// NewNovelAgent creates the pair of specialist agents sharing the same model backend.
func NewNovelAgent(modelName, apiKey, baseURL string) *NovelAgent {
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
		Stream:      false,
	}

	analyzer := llmagent.New("novel-analyzer",
		llmagent.WithModel(m),
		llmagent.WithDescription(analyzerDesc),
		llmagent.WithInstruction(analyzerPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)

	writerGenConfig := genConfig
	writerGenConfig.Temperature = floatPtr(0.7)

	writer := llmagent.New("screenplay-writer",
		llmagent.WithModel(m),
		llmagent.WithDescription(writerDesc),
		llmagent.WithInstruction(writerPrompt),
		llmagent.WithGenerationConfig(writerGenConfig),
	)

	return &NovelAgent{Analyzer: analyzer, Writer: writer}
}

func floatPtr(f float64) *float64 { return &f }

// -- Agent descriptions --
const analyzerDesc = "分析小说文本，提取角色、场景、情节结构和章节摘要"
const writerDesc = "将分析好的小说结构转化为格式正确的 JSON 剧本"

// -- System prompts --

// analyzerPrompt instructs the analysis agent to produce structured JSON output.
const analyzerPrompt = `你是一名专业文学分析师，专精于小说到剧本的改编。

你的任务是分析提供的小说章节，输出结构化的 JSON 分析结果，包含：

1. 标题与元数据 — 识别作品的标题、作者、类型，写一句 logline
2. 角色 — 提取每个有名角色，包含：
   - 唯一 ID（如 "char_001"）
   - 全名
   - 简要介绍（外貌、作用、性格）
   - 性别、大致年龄
   - 性格特征（2-4 个关键词）
   - 与其他角色的关键关系
3. 章节分解 — 对每一章提供：
   - 章节索引和标题
   - 一句话概括
   - 涉及的场景/地点
   - 出现的角色

请全面分析，遗漏角色或场景会导致剧本不完整。
只输出合法 JSON，不要 markdown 代码块，不要额外注释。`

// writerPrompt instructs the screenplay generation agent to produce JSON.
// Note: the `scenes` array is the primary output; characters/metadata are also included.
const writerPrompt = `你是一名专业编剧，专精于将小说改编为剧本格式。

你的任务是将小说分析结果转化为结构化的 JSON 剧本。

规则：
1. 场景标题遵循标准格式：内景/外景 地点 - 时间
2. 动作描述行描写可见可闻的内容。用现在时，展示而非讲述。
3. 对白忠实于每个角色的语气，忠实于原始素材。
4. 括号提示（parenthetical）少用——仅在必要时使用。
5. 场景顺序遵循小说时间线，除非重新组织能改善戏剧节奏。
6. 每个场景应有明确的戏剧目的。

JSON 结构必须严格遵循以下格式：

{
  "metadata": {
    "title": "...",
    "author": "...",
    "version": "1.0",
    "created_at": "2026-06-05",
    "original_novel": "...",
    "logline": "...",
    "genre": ["..."],
    "source_chapters": [1, 2, 3]
  },
  "characters": [
    { "id": "char_001", "name": "...", "intro": "...", "gender": "...", "age": 30, "personality": ["..."], "relationships": [{ "target": "char_002", "type": "...", "description": "..." }] }
  ],
  "scenes": [
    {
      "id": "scene_001",
      "sequence": 1,
      "heading": "外景/内景 地点 - 时间",
      "setting": { "location": "...", "time": "...", "interior": true },
      "synopsis": "...",
      "characters_present": ["char_001"],
      "content": [
        { "type": "action", "text": "..." },
        { "type": "dialogue", "character": "...", "parenthetical": "(可选)", "text": "..." },
        { "type": "transition", "text": "切至：" }
      ]
    }
  ]
}
只输出合法 JSON，不要 markdown 代码块，不要额外注释。`
