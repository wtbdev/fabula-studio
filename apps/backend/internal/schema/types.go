// Package schema defines the structured data types for screenplay representation
// and novel analysis. These types serialize to YAML as the output format.
package schema

// Screenplay is the root document produced by the conversion.
// It contains metadata, character profiles, and a sequence of scenes.
type Screenplay struct {
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Characters []Character `yaml:"characters" json:"characters"`
	Scenes     []Scene     `yaml:"scenes" json:"scenes"`
}

// Metadata describes the screenplay itself and its provenance.
type Metadata struct {
	Title         string   `yaml:"title" json:"title"`
	Author        string   `yaml:"author" json:"author"`
	Version       string   `yaml:"version" json:"version"`
	CreatedAt     string   `yaml:"created_at" json:"created_at"`
	OriginalNovel string   `yaml:"original_novel" json:"original_novel"`
	Logline       string   `yaml:"logline" json:"logline"`
	Genre         []string `yaml:"genre" json:"genre"`
	SourceChapters []int   `yaml:"source_chapters" json:"source_chapters"`
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
	ID               string        `yaml:"id" json:"id"`
	Sequence         int           `yaml:"sequence" json:"sequence"`
	Heading          string        `yaml:"heading" json:"heading"`
	Setting          SceneSetting  `yaml:"setting" json:"setting"`
	Synopsis         string        `yaml:"synopsis" json:"synopsis"`
	CharactersPresent []string     `yaml:"characters_present" json:"characters_present"`
	Content          []SceneElement `yaml:"content" json:"content"`
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
	ElementAction       SceneElementType = "action"
	ElementDialogue     SceneElementType = "dialogue"
	ElementTransition   SceneElementType = "transition"
	ElementShot         SceneElementType = "shot"
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
	Index    int      `json:"index"`
	Title    string   `json:"title"`
	Synopsis string   `json:"synopsis"`
	Settings []string `json:"settings"`
	Characters []string `json:"characters"`
}

// ConversionRequest is the input to the /api/convert endpoint.
type ConversionRequest struct {
	Title    string   `json:"title"`
	Author   string   `json:"author"`
	Chapters []string `json:"chapters"`
}

// ConversionResponse is the output from the /api/convert endpoint.
type ConversionResponse struct {
	Screenplay *Screenplay `json:"screenplay,omitempty"`
	YAML       string      `json:"yaml,omitempty"`
	Error      string      `json:"error,omitempty"`
}
