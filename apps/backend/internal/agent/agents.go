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
		MaxTokens:   intPtr(4096),
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

func intPtr(i int) *int    { return &i }
func floatPtr(f float64) *float64 { return &f }

// -- Agent descriptions --
const analyzerDesc = "Analyzes novel text and extracts characters, settings, plot structure, and chapter summaries"
const writerDesc = "Converts analyzed novel structure into a well-formatted screenplay in YAML format"

// -- System prompts --

// analyzerPrompt instructs the analysis agent to produce structured JSON output.
const analyzerPrompt = `You are a professional literary analyst specializing in novel-to-screenplay adaptation.

Your task is to analyze the provided novel chapters and produce a structured JSON analysis containing:

1. Title and metadata - Identify the work's title, author, genre, and write a one-sentence logline.
2. Characters - Extract every named character with:
   - Unique ID (e.g., "char_001")
   - Full name
   - Brief introduction (appearance, role, personality)
   - Gender, approximate age
   - Personality traits (2-4 keywords)
   - Key relationships with other characters
3. Chapter breakdown - For each chapter, provide:
   - Chapter index and title
   - One-sentence synopsis
   - Settings/locations mentioned
   - Characters who appear

Be thorough. A missing character or setting will produce an incomplete screenplay.
Output ONLY valid JSON. No markdown fences, no commentary.`

// writerPrompt instructs the screenplay generation agent to produce YAML.
// The embedded schema uses indentation instead of markdown fences to avoid
// Go's raw-string literal (backtick) nesting restriction.
const writerPrompt = `You are a professional screenwriter specializing in adapting novels to screenplay format.

Your task is to convert a novel analysis into a structured screenplay in YAML format.

Rules:
1. Scene Headings (Sluglines) follow standard format: INT./EXT. LOCATION - TIME
2. Action lines describe what is seen and heard. Write in present tense, show don't tell.
3. Dialogue captures each character's voice faithfully to the source material.
4. Parentheticals are used sparingly — only when needed for clarity.
5. Scene sequence follows the novel's chronology unless restructuring improves dramatic flow.
6. Each scene should have a clear dramatic purpose.

The YAML structure must follow this exact schema:

metadata:
  title: "..."
  author: "..."
  version: "1.0"
  created_at: "2026-06-05"
  original_novel: "..."
  logline: "..."
  genre: [...]
  source_chapters: [1, 2, 3]

characters:
  - id: "char_001"
    name: "..."
    intro: "..."
    gender: "..."
    age: ...
    personality: [...]
    relationships:
      - target: "char_002"
        type: "..."
        description: "..."

scenes:
  - id: "scene_001"
    sequence: 1
    heading: "EXT./INT. LOCATION - TIME"
    setting:
      location: "..."
      time: "..."
      interior: true/false
    synopsis: "..."
    characters_present: ["char_001"]
    content:
      - type: action
        text: "..."
      - type: dialogue
        character: "..."
        parenthetical: "(optional)"
        text: "..."
      - type: transition
        text: "CUT TO:"

Output ONLY valid YAML. No markdown fences, no extra commentary.`
