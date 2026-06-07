// Package schema defines the structured data types for screenplay representation
// and novel analysis. These types serialize to YAML as the output format.
package schema

import "context"

// Screenplay is the root document produced by the conversion.
// It contains metadata, character profiles, and a sequence of scenes.
type Screenplay struct {
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Characters []Character `yaml:"characters" json:"characters"`
	Scenes     []Scene     `yaml:"scenes" json:"scenes"`
}

// Metadata describes the screenplay itself and its provenance.
type Metadata struct {
	Title          string   `yaml:"title" json:"title"`
	Author         string   `yaml:"author" json:"author"`
	Version        string   `yaml:"version" json:"version"`
	CreatedAt      string   `yaml:"created_at" json:"created_at"`
	OriginalNovel  string   `yaml:"original_novel" json:"original_novel"`
	Logline        string   `yaml:"logline" json:"logline"`
	Genre          []string `yaml:"genre" json:"genre"`
	SourceChapters []int    `yaml:"source_chapters" json:"source_chapters"`
}

// Character represents a character extracted from the novel.
type Character struct {
	ID            string              `yaml:"id" json:"id"`
	Name          string              `yaml:"name" json:"name"`
	Intro         string              `yaml:"intro" json:"intro"`
	Gender        string              `yaml:"gender,omitempty" json:"gender,omitempty"`
	Age           int                 `yaml:"age,omitempty" json:"age,omitempty"`
	Personality   []string            `yaml:"personality" json:"personality"`
	Relationships []CharacterRelation `yaml:"relationships,omitempty" json:"relationships,omitempty"`
}

// CharacterRelation describes a relationship between two characters.
type CharacterRelation struct {
	Target      string `yaml:"target" json:"target"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
}

// Scene represents a single scene in the screenplay.
// It maps to a standard slugline / scene heading in screenwriting.
type Scene struct {
	ID                string         `yaml:"id" json:"id"`
	Sequence          int            `yaml:"sequence" json:"sequence"`
	Heading           string         `yaml:"heading" json:"heading"`
	Setting           SceneSetting   `yaml:"setting" json:"setting"`
	Synopsis          string         `yaml:"synopsis" json:"synopsis"`
	CharactersPresent []string       `yaml:"characters_present" json:"characters_present"`
	Content           []SceneElement `yaml:"content" json:"content"`
}

// SceneSetting describes the physical and temporal setting of a scene.
type SceneSetting struct {
	Location string `yaml:"location" json:"location"`
	Time     string `yaml:"time" json:"time"`
	Interior bool   `yaml:"interior" json:"interior"`
}

// SceneElementType enumerates the kinds of content in a scene.
type SceneElementType string

const (
	ElementAction        SceneElementType = "action"
	ElementDialogue      SceneElementType = "dialogue"
	ElementTransition    SceneElementType = "transition"
	ElementShot          SceneElementType = "shot"
	ElementParenthetical SceneElementType = "parenthetical"
)

// SceneElement is one building block of a scene: action description, dialogue,
// transition, or shot direction.
type SceneElement struct {
	Type          SceneElementType `yaml:"type" json:"type"`
	Character     string           `yaml:"character,omitempty" json:"character,omitempty"`
	Parenthetical string           `yaml:"parenthetical,omitempty" json:"parenthetical,omitempty"`
	Text          string           `yaml:"text" json:"text"`
}

// ---------------------------------------------------------------------------
// Generation contract types
// ---------------------------------------------------------------------------

// AdaptationProfile is the normalized backend-facing adaptation configuration.
// It is derived from persisted project configuration and intentionally keeps
// UI form fields out of generation prompts.
type AdaptationProfile struct {
	Style            string `json:"style,omitempty"`
	DialogueLevel    string `json:"dialogueLevel,omitempty"`
	AdaptationMode   string `json:"adaptationMode,omitempty"`
	SceneGranularity string `json:"sceneGranularity,omitempty"`
	NarrationLevel   string `json:"narrationLevel,omitempty"`
	CustomGuidance   string `json:"customGuidance,omitempty"`
}

type adaptationProfileContextKey struct{}

// WithAdaptationProfile attaches the normalized adaptation profile to a context
// without changing existing pipeline call signatures.
func WithAdaptationProfile(ctx context.Context, profile *AdaptationProfile) context.Context {
	if profile == nil {
		return ctx
	}
	return context.WithValue(ctx, adaptationProfileContextKey{}, profile)
}

// AdaptationProfileFromContext returns the profile attached by
// WithAdaptationProfile, if any.
func AdaptationProfileFromContext(ctx context.Context) (*AdaptationProfile, bool) {
	profile, ok := ctx.Value(adaptationProfileContextKey{}).(*AdaptationProfile)
	return profile, ok && profile != nil
}

// SourceIndex is a stable index of source text sentences used to ground later
// generation artifacts without persisting offsets in scene rows.
type SourceIndex struct {
	SourceID  string           `json:"sourceId,omitempty"`
	Title     string           `json:"title,omitempty"`
	Sentences []SourceSentence `json:"sentences"`
}

// SourceSentence is one indexed source sentence with byte offsets into the
// original source text where available.
type SourceSentence struct {
	ID           string `json:"id"`
	Index        int    `json:"index"`
	Chapter      int    `json:"chapter,omitempty"`
	ChapterIndex int    `json:"chapterIndex,omitempty"`
	StartOffset  int    `json:"startOffset,omitempty"`
	EndOffset    int    `json:"endOffset,omitempty"`
	Text         string `json:"text"`
}

// SourceSentenceRef references a sentence or contiguous sentence range in a
// SourceIndex.
type SourceSentenceRef struct {
	SentenceID string `json:"sentenceId,omitempty"`
	StartIndex int    `json:"startIndex,omitempty"`
	EndIndex   int    `json:"endIndex,omitempty"`
}

// StoryBeat is a source-grounded narrative beat extracted before scene
// planning.
type StoryBeat struct {
	ID             string               `json:"id"`
	Sequence       int                  `json:"sequence"`
	Summary        string               `json:"summary"`
	Purpose        string               `json:"purpose,omitempty"`
	Characters     []string             `json:"characters,omitempty"`
	Locations      []string             `json:"locations,omitempty"`
	SourceRefs     []SourceSentenceRef  `json:"sourceRefs,omitempty"`
	ExpectedChange SceneExpectedChanges `json:"expectedChange,omitempty"`
}

// SceneExpectedChanges describes graph/state changes a planned beat or scene is
// expected to produce.
type SceneExpectedChanges struct {
	CharacterChanges    []string `json:"characterChanges,omitempty"`
	RelationshipChanges []string `json:"relationshipChanges,omitempty"`
	FactChanges         []string `json:"factChanges,omitempty"`
	OpenQuestions       []string `json:"openQuestions,omitempty"`
}

// ScenePlan is the public, read-only scene planning artifact. Internal planners
// may carry richer package-specific data; this type is the API contract.
type ScenePlan struct {
	ID              string               `json:"id"`
	Sequence        int                  `json:"sequence"`
	Title           string               `json:"title,omitempty"`
	Purpose         string               `json:"purpose,omitempty"`
	SourceBeatIDs   []string             `json:"sourceBeatIds,omitempty"`
	SourceRefs      []SourceSentenceRef  `json:"sourceRefs,omitempty"`
	Location        string               `json:"location,omitempty"`
	TimeFrame       string               `json:"timeFrame,omitempty"`
	Characters      []string             `json:"characters,omitempty"`
	KeyPlotPoints   []string             `json:"keyPlotPoints,omitempty"`
	ExpectedChanges SceneExpectedChanges `json:"expectedChanges,omitempty"`
	Scenes          []ScenePlan          `json:"scenes,omitempty"`
}

// GenerationWarning is a non-fatal issue collected while creating generation
// artifacts.
type GenerationWarning struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

// GenerationArtifacts groups read-only generation artifacts that can be safely
// returned by API responses without requiring schema persistence.
type GenerationArtifacts struct {
	SourceIndex   *SourceIndex        `json:"sourceIndex,omitempty"`
	StoryBeats    []StoryBeat         `json:"storyBeats,omitempty"`
	ScenePlan     *ScenePlan          `json:"scenePlan,omitempty"`
	Warnings      []GenerationWarning `json:"warnings,omitempty"`
	GraphSnapshot any                 `json:"graphSnapshot,omitempty"`
}

// ---------------------------------------------------------------------------
// Novel analysis types (internal, used during conversion)
// ---------------------------------------------------------------------------

// NovelAnalysis is the structured summary produced by the analysis agent.
type NovelAnalysis struct {
	Title        string        `json:"title"`
	Author       string        `json:"author"`
	Genre        []string      `json:"genre"`
	Logline      string        `json:"logline"`
	Characters   []Character   `json:"characters"`
	ChapterCount int           `json:"chapter_count"`
	Chapters     []ChapterInfo `json:"chapters"`
}

// ChapterInfo is a lightweight summary of one chapter.
type ChapterInfo struct {
	Index      int      `json:"index"`
	Title      string   `json:"title"`
	Synopsis   string   `json:"synopsis"`
	Settings   []string `json:"settings"`
	Characters []string `json:"characters"`
}

// ConversionRequest is the input to the /api/convert endpoint.
type ConversionRequest struct {
	Title             string             `json:"title"`
	Author            string             `json:"author"`
	Chapters          []string           `json:"chapters"`
	AdaptationProfile *AdaptationProfile `json:"adaptationProfile,omitempty"`
}

// ConversionResponse is the output from the /api/convert endpoint.
type ConversionResponse struct {
	Screenplay *Screenplay `json:"screenplay,omitempty"`
	YAML       string      `json:"yaml,omitempty"`
	Error      string      `json:"error,omitempty"`
}
