// Package converter implements the novel-to-screenplay conversion pipeline.
package converter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/runner"

	"github.com/fabula-studio/backend/internal/agent"
	"github.com/fabula-studio/backend/internal/schema"
	"github.com/fabula-studio/backend/internal/util"
)

// Converter orchestrates the pipeline: novel text → analysis → screenplay YAML.
type Converter struct {
	agent *agent.NovelAgent
}

// New creates a Converter backed by the given NovelAgent.
func New(a *agent.NovelAgent) *Converter {
	return &Converter{agent: a}
}

// Convert processes the input chapters end-to-end and returns the screenplay.
//
// Pipeline:
//
//	 1. Analyze: Run analysis agent on each chapter to extract structure.
//	 2. Assemble: Merge per-chapter analyses into a combined NovelAnalysis.
//	 3. Write: Pass the combined analysis to the screenplay writer agent.
//	 4. Parse: Parse the returned JSON into a Screenplay struct (validated).
func (c *Converter) Convert(ctx context.Context, title, author string, chapters []string) (*schema.Screenplay, string, error) {
	if len(chapters) < 3 {
		return nil, "", fmt.Errorf("至少需要3个章节，当前仅提供 %d 个", len(chapters))
	}

	// Step 1 — Analyze each chapter
	var allAnalyses []schema.NovelAnalysis
	for i, text := range chapters {
		analysis, err := c.analyzeChapter(ctx, i+1, text)
		if err != nil {
			return nil, "", fmt.Errorf("分析第 %d 章失败: %w", i+1, err)
		}
		allAnalyses = append(allAnalyses, *analysis)
	}

	// Step 2 — Merge into a single NovelAnalysis
	merged := mergeAnalyses(title, author, allAnalyses)

	// Step 3 — Generate screenplay (as JSON)
	respStr, err := c.generateScreenplay(ctx, &merged)
	if err != nil {
		return nil, "", fmt.Errorf("生成剧本失败: %w", err)
	}

	// Step 4 — Parse & validate
	scr := &schema.Screenplay{}
	if err := json.Unmarshal([]byte(respStr), scr); err != nil {
		return nil, respStr, fmt.Errorf("解析生成的 JSON 失败: %w\n\n原始 JSON 输出:\n%s", err, respStr)
	}

	return scr, respStr, nil
}

// analyzeChapter sends one chapter to the analysis agent and parses the result.
func (c *Converter) analyzeChapter(ctx context.Context, index int, text string) (*schema.NovelAnalysis, error) {
	prompt := fmt.Sprintf(`请分析以下第 %d 章的内容。

章节内容：
%s

请以JSON格式输出分析结果，包含：title、author、genre（数组）、logline、characters（数组，每个包含id/name/intro/gender/age/personality/relationships）、chapters（数组，每个包含index/title/synopsis/settings/characters）。`, index, text)

	response, err := c.runAgent(ctx, c.agent.Analyzer, prompt)
	if err != nil {
		return nil, err
	}

	response, err = util.PrepareJSON(response, "chapter analysis output")
	if err != nil {
		return nil, err
	}

	analysis := &schema.NovelAnalysis{}
	if err := json.Unmarshal([]byte(response), analysis); err != nil {
		return nil, fmt.Errorf("分析结果JSON解析失败: %w\n原始输出: %s", err, response)
	}

	return analysis, nil
}

// generateScreenplay sends the merged analysis to the writer agent.
func (c *Converter) generateScreenplay(ctx context.Context, analysis *schema.NovelAnalysis) (string, error) {
	analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")
	prompt := fmt.Sprintf(`基于以下小说分析结果，生成一个完整的剧本 JSON。

分析结果：
%s

生成要求：
1. 剧本必须包含至少 3 个以上场景（scene）
2. 每个场景包含完整的场景标题、动作描述和对白
3. 保持原作的叙事节奏和戏剧张力
4. 输出严格遵循 JSON 格式`, string(analysisJSON))

	raw, err := c.runAgent(ctx, c.agent.Writer, prompt)
	if err != nil {
		return "", err
	}
	return util.PrepareJSON(raw, "screenplay writer output")
}

// runAgent executes a single-turn agent run and returns the text response.
func (c *Converter) runAgent(ctx context.Context, agt *llmagent.LLMAgent, prompt string) (string, error) {
	r := runner.NewRunner("fabula-converter", agt)
	msg := model.NewUserMessage(prompt)
	eventChan, err := r.Run(ctx, "default-user", fmt.Sprintf("session-%d", time.Now().UnixNano()), msg)
	if err != nil {
		return "", fmt.Errorf("agent run failed: %w", err)
	}

	var sb strings.Builder
	for evt := range eventChan {
		if evt.Error != nil {
			return "", fmt.Errorf("agent error: %s", evt.Error.Message)
		}
		for _, choice := range evt.Response.Choices {
			content := choice.Message.Content
			if content == "" {
				content = choice.Delta.Content
			}
			if content != "" {
				sb.WriteString(content)
			}
		}
	}
	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("agent returned empty response")
	}
	return result, nil
}

// mergeAnalyses combines per-chapter analyses into one.
// The first non-empty title/author/logline wins; characters are deduplicated by name.
func mergeAnalyses(title, author string, analyses []schema.NovelAnalysis) schema.NovelAnalysis {
	merged := schema.NovelAnalysis{
		Title:    title,
		Author:   author,
		Chapters: make([]schema.ChapterInfo, 0, len(analyses)),
	}

	charMap := make(map[string]schema.Character) // name → Character
	genreSet := make(map[string]bool)

	for i, a := range analyses {
		if merged.Title == "" && a.Title != "" {
			merged.Title = a.Title
		}
		if merged.Author == "" && a.Author != "" {
			merged.Author = a.Author
		}
		if merged.Logline == "" && a.Logline != "" {
			merged.Logline = a.Logline
		}
		for _, g := range a.Genre {
			genreSet[g] = true
		}
		for _, ch := range a.Characters {
			if _, exists := charMap[ch.Name]; !exists {
				charMap[ch.Name] = ch
			}
		}

		if len(a.Chapters) > 0 {
			merged.Chapters = append(merged.Chapters, a.Chapters...)
		} else {
			merged.Chapters = append(merged.Chapters, schema.ChapterInfo{Index: i + 1, Synopsis: a.Logline})
		}
	}

	for g := range genreSet {
		merged.Genre = append(merged.Genre, g)
	}
	for _, ch := range charMap {
		merged.Characters = append(merged.Characters, ch)
	}
	merged.ChapterCount = len(analyses)

	return merged
}
