// Package validator performs structural validation on the final screenplay.
package validator

import (
	"fmt"

	"github.com/fabula-studio/backend/internal/schema"
)

// ValidationResult holds the outcome of screenplay validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// Validator performs programmatic checks on a screenplay.
type Validator struct{}

// Validate runs all checks and returns the combined result.
func (v *Validator) Validate(sp *schema.Screenplay, minChapters int) *ValidationResult {
	result := &ValidationResult{Valid: true}

	v.validateFields(sp, result)
	v.validateSceneOrder(sp, result)
	v.validateCharacterRefs(sp, result)
	v.validateNonEmpty(sp, result)
	v.validateChapterCoverage(sp, minChapters, result)

	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

// validateFields checks that required top-level fields exist.
func (v *Validator) validateFields(sp *schema.Screenplay, r *ValidationResult) {
	if sp.Metadata.Title == "" {
		r.Errors = append(r.Errors, "metadata.title is required")
	}
	if sp.Metadata.Author == "" {
		r.Warnings = append(r.Warnings, "metadata.author is empty")
	}
	if len(sp.Characters) == 0 {
		r.Errors = append(r.Errors, "characters list is empty")
	}
	if len(sp.Scenes) == 0 {
		r.Errors = append(r.Errors, "scenes list is empty")
	}
}

// validateSceneOrder checks that scene sequences are sequential starting from 1.
func (v *Validator) validateSceneOrder(sp *schema.Screenplay, r *ValidationResult) {
	if len(sp.Scenes) == 0 {
		return
	}
	for i, sc := range sp.Scenes {
		expected := i + 1
		if sc.Sequence != expected {
			r.Errors = append(r.Errors, fmt.Sprintf("scene %q has sequence %d, expected %d", sc.ID, sc.Sequence, expected))
		}
	}
}

// validateCharacterRefs ensures dialogue characters and characters_present reference valid character IDs/names.
func (v *Validator) validateCharacterRefs(sp *schema.Screenplay, r *ValidationResult) {
	charNames := make(map[string]bool)
	charIDs := make(map[string]bool)
	for _, ch := range sp.Characters {
		charNames[ch.Name] = true
		charIDs[ch.ID] = true
	}

	for _, sc := range sp.Scenes {
		for _, cid := range sc.CharactersPresent {
			if !charIDs[cid] && !charNames[cid] {
				r.Errors = append(r.Errors, fmt.Sprintf("scene %q references unknown character %q", sc.ID, cid))
			}
		}
		for _, elem := range sc.Content {
			if elem.Type == schema.ElementDialogue && elem.Character != "" {
				if !charNames[elem.Character] && !charIDs[elem.Character] {
					r.Warnings = append(r.Warnings, fmt.Sprintf("scene %q dialogue references unknown character %q", sc.ID, elem.Character))
				}
			}
		}
	}
}

// validateNonEmpty checks that scenes have actual content.
func (v *Validator) validateNonEmpty(sp *schema.Screenplay, r *ValidationResult) {
	for _, sc := range sp.Scenes {
		if len(sc.Content) == 0 {
			r.Errors = append(r.Errors, fmt.Sprintf("scene %q has empty content", sc.ID))
		}
		if sc.Heading == "" {
			r.Warnings = append(r.Warnings, fmt.Sprintf("scene %q has empty heading", sc.ID))
		}
	}
}

// validateChapterCoverage checks that at least minChapters source chapters are covered.
func (v *Validator) validateChapterCoverage(sp *schema.Screenplay, minChapters int, r *ValidationResult) {
	if minChapters <= 0 {
		return
	}
	covered := len(sp.Metadata.SourceChapters)
	if covered < minChapters {
		r.Errors = append(r.Errors, fmt.Sprintf("only %d source chapters covered, minimum is %d", covered, minChapters))
	}
}
