package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"

	"github.com/fabula-studio/backend/internal/segment"
	"github.com/fabula-studio/backend/internal/util"
)

const storyBeatExtractorDesc = "从源文本句子窗口中提取改编故事节拍"

const storyBeatExtractorPrompt = `你是一名小说改编结构分析师。你将收到一段连续的源文本句子窗口。

任务：把窗口内适合剧本改编的连续叙事变化提取为 story beats。每个 beat 必须：
- 使用窗口中真实存在的 start_sentence_id 和 end_sentence_id，且二者之间是连续源句范围
- 保持源句顺序，不重叠，不倒退
- summary 概括这段源句发生的具体事件/信息变化，不要写泛泛主题
- dramatic_purpose 说明这个节拍在剧本中的戏剧作用
- conflict 写清阻力/悬念/信息落差；没有显性冲突时填空字符串
- characters/location/time_frame 只填写源句能支持的信息，不要编造
- boundary_reason 说明为什么在 end_sentence_id 处收束该节拍

输出合法 JSON 对象：
{
  "beats": [
    {
      "start_sentence_id": "s_000001",
      "end_sentence_id": "s_000003",
      "summary": "...",
      "dramatic_purpose": "...",
      "conflict": "...",
      "characters": ["..."],
      "location": "...",
      "time_frame": "...",
      "boundary_reason": "..."
    }
  ]
}

只输出合法 JSON，不要 markdown 代码块，不要额外注释。`

// StoryBeatExtractorAgent extracts adaptation beats from sentence windows.
type StoryBeatExtractorAgent struct {
	agent *llmagent.LLMAgent
}

// NewStoryBeatExtractorAgent creates the story beat extraction agent.
func NewStoryBeatExtractorAgent(modelName, apiKey, baseURL string) *StoryBeatExtractorAgent {
	opts := []openai.Option{}
	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelName, opts...)
	genConfig := model.GenerationConfig{Temperature: floatPtr(0.2)}
	agt := llmagent.New("story-beat-extractor",
		llmagent.WithModel(m),
		llmagent.WithDescription(storyBeatExtractorDesc),
		llmagent.WithInstruction(storyBeatExtractorPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &StoryBeatExtractorAgent{agent: agt}
}

// Extract extracts and repairs story beats for the complete source index.
func (a *StoryBeatExtractorAgent) Extract(ctx context.Context, idx *segment.SourceIndex) ([]segment.StoryBeat, error) {
	if idx == nil || len(idx.Sentences) == 0 {
		return nil, fmt.Errorf("source index has no sentences")
	}
	windows := segment.NewBeatWindows(idx, segment.DefaultBeatWindowSize, segment.DefaultBeatWindowOverlap)
	rawBeats := make([]segment.StoryBeat, 0, len(windows))
	for _, window := range windows {
		beats, err := a.extractWindow(ctx, window)
		if err != nil {
			beats = []segment.StoryBeat{fallbackWindowBeat(window, fmt.Sprintf("LLM 节拍提取失败，使用确定性窗口回退: %v", err))}
		}
		rawBeats = append(rawBeats, beats...)
	}
	return segment.ReconcileStoryBeatBoundaries(idx, rawBeats), nil
}

func (a *StoryBeatExtractorAgent) extractWindow(ctx context.Context, window segment.BeatWindow) ([]segment.StoryBeat, error) {
	payload := struct {
		WindowID        string             `json:"window_id"`
		Sequence        int                `json:"sequence"`
		StartSentenceID string             `json:"start_sentence_id"`
		EndSentenceID   string             `json:"end_sentence_id"`
		Sentences       []segment.Sentence `json:"sentences"`
	}{WindowID: window.ID, Sequence: window.Sequence, StartSentenceID: window.StartSentenceID, EndSentenceID: window.EndSentenceID, Sentences: window.Sentences}
	payloadJSON, _ := json.Marshal(payload)
	prompt := fmt.Sprintf("Extract story beats from this source window:\n```json\n%s\n```", string(payloadJSON))
	raw, err := Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}
	raw, err = util.PrepareJSON(raw, "story beat extractor output")
	if err != nil {
		return nil, err
	}
	var response struct {
		Beats []segment.StoryBeat `json:"beats"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		var beats []segment.StoryBeat
		if err2 := json.Unmarshal([]byte(raw), &beats); err2 != nil {
			return nil, fmt.Errorf("failed to parse story beats JSON: %w", err)
		}
		response.Beats = beats
	}
	if len(response.Beats) == 0 {
		return nil, fmt.Errorf("story beat extractor returned no beats")
	}
	windowIndex := segment.NewSourceIndex(window.Sentences)
	return segment.ValidateAndRepairStoryBeats(windowIndex, response.Beats), nil
}

func fallbackWindowBeat(window segment.BeatWindow, reason string) segment.StoryBeat {
	return segment.StoryBeat{
		StartSentenceID: window.StartSentenceID,
		EndSentenceID:   window.EndSentenceID,
		Summary:         summarizeWindow(window.Sentences),
		DramaticPurpose: "保留源文本中的连续行动/信息变化",
		BoundaryReason:  reason,
	}
}

func summarizeWindow(sentences []segment.Sentence) string {
	if len(sentences) == 0 {
		return ""
	}
	text := sentences[0].Text
	if len(sentences) > 1 {
		text += " ... " + sentences[len(sentences)-1].Text
	}
	runes := []rune(text)
	if len(runes) <= 160 {
		return text
	}
	return string(runes[:157]) + "..."
}
