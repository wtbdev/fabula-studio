package validator

import (
	"testing"

	"github.com/fabula-studio/backend/internal/schema"
)

func TestValidate_ValidScreenplay(t *testing.T) {
	sp := &schema.Screenplay{
		Metadata: schema.Metadata{
			Title:          "测试剧本",
			Author:         "作者",
			SourceChapters: []int{1, 2, 3},
		},
		Characters: []schema.Character{
			{ID: "char_001", Name: "林浩"},
		},
		Scenes: []schema.Scene{
			{
				ID:       "scene_001",
				Sequence: 1,
				Heading:  "INT. 办公室 - 夜",
				CharactersPresent: []string{"char_001"},
				Content: []schema.SceneElement{
					{Type: schema.ElementAction, Text: "林浩坐在桌前。"},
				},
			},
		},
	}

	v := &Validator{}
	result := v.Validate(sp, 3)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_MissingTitle(t *testing.T) {
	sp := &schema.Screenplay{
		Characters: []schema.Character{{ID: "c1", Name: "A"}},
		Scenes:     []schema.Scene{{ID: "s1", Sequence: 1, Content: []schema.SceneElement{{Type: schema.ElementAction, Text: "x"}}}},
	}
	v := &Validator{}
	result := v.Validate(sp, 0)
	if result.Valid {
		t.Error("expected invalid due to missing title")
	}
}

func TestValidate_WrongSequence(t *testing.T) {
	sp := &schema.Screenplay{
		Metadata:   schema.Metadata{Title: "T"},
		Characters: []schema.Character{{ID: "c1", Name: "A"}},
		Scenes: []schema.Scene{
			{ID: "s1", Sequence: 1, Content: []schema.SceneElement{{Type: schema.ElementAction, Text: "x"}}},
			{ID: "s2", Sequence: 3, Content: []schema.SceneElement{{Type: schema.ElementAction, Text: "y"}}},
		},
	}
	v := &Validator{}
	result := v.Validate(sp, 0)
	if result.Valid {
		t.Error("expected invalid due to wrong sequence")
	}
}

func TestValidate_UnknownCharacterRef(t *testing.T) {
	sp := &schema.Screenplay{
		Metadata:   schema.Metadata{Title: "T"},
		Characters: []schema.Character{{ID: "c1", Name: "A"}},
		Scenes: []schema.Scene{
			{
				ID: "s1", Sequence: 1,
				CharactersPresent: []string{"c1", "c_unknown"},
				Content:           []schema.SceneElement{{Type: schema.ElementAction, Text: "x"}},
			},
		},
	}
	v := &Validator{}
	result := v.Validate(sp, 0)
	if result.Valid {
		t.Error("expected invalid due to unknown character ref")
	}
}

func TestValidate_EmptySceneContent(t *testing.T) {
	sp := &schema.Screenplay{
		Metadata:   schema.Metadata{Title: "T"},
		Characters: []schema.Character{{ID: "c1", Name: "A"}},
		Scenes:     []schema.Scene{{ID: "s1", Sequence: 1}},
	}
	v := &Validator{}
	result := v.Validate(sp, 0)
	if result.Valid {
		t.Error("expected invalid due to empty scene content")
	}
}

func TestValidate_ChapterCoverage(t *testing.T) {
	sp := &schema.Screenplay{
		Metadata: schema.Metadata{
			Title:          "T",
			SourceChapters: []int{1},
		},
		Characters: []schema.Character{{ID: "c1", Name: "A"}},
		Scenes: []schema.Scene{
			{ID: "s1", Sequence: 1, Content: []schema.SceneElement{{Type: schema.ElementAction, Text: "x"}}},
		},
	}
	v := &Validator{}
	result := v.Validate(sp, 3)
	if result.Valid {
		t.Error("expected invalid due to insufficient chapter coverage")
	}
}
