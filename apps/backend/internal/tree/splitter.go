package tree

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Splitter performs hard-cut decomposition of novel text into story nodes.
type Splitter struct {
	maxChunkSize int
}

// NewSplitter creates a splitter with the given max node size in characters.
func NewSplitter(maxChunkSize int) *Splitter {
	return &Splitter{maxChunkSize: maxChunkSize}
}

// SplitChapters performs the initial hard-cut of chapters into first-layer nodes.
// It preserves chapter boundaries and tries to split at paragraph boundaries.
func (s *Splitter) SplitChapters(chapters []string) *StoryTree {
	tree := NewTree()
	nodeSeq := 0
	var topLevelIDs []string

	for chIdx, text := range chapters {
		chunks := s.splitText(text)
		var prevNodeID string

		for _, chunk := range chunks {
			nodeSeq++
			nodeID := fmt.Sprintf("node_%03d", nodeSeq)
			node := &StoryNode{
				ID:            nodeID,
				Level:         0,
				TextContent:   chunk,
				SourceChapter: chIdx,
				Decision:      DecisionKeep,
			}
			if prevNodeID != "" {
				if prev := tree.GetNode(prevNodeID); prev != nil {
					prev.RightNeighbor = nodeID
				}
			}
			tree.AddNode(node)
			topLevelIDs = append(topLevelIDs, nodeID)
			prevNodeID = nodeID
		}
	}


	// Build a synthetic root that holds all top-level nodes as children.
	nodeSeq++
	rootID := fmt.Sprintf("node_%03d", nodeSeq)
	root := &StoryNode{
		ID:          rootID,
		Level:       -1,
		ChildrenIDs: topLevelIDs,
	}
	for _, nid := range topLevelIDs {
		if n := tree.GetNode(nid); n != nil {
			n.ParentID = rootID
		}
	}
	tree.AddNode(root)
	tree.RootNodeID = rootID

	tree.UpdateLeafIDs()
	return tree
}

// splitText breaks a single chapter's text into chunks that fit within maxChunkSize.
// It prefers splitting at paragraph boundaries, then at sentence boundaries.
func (s *Splitter) splitText(text string) []string {
	if utf8.RuneCountInString(text) <= s.maxChunkSize {
		return []string{text}
	}

	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		paraRunes := utf8.RuneCountInString(para)
		currentRunes := utf8.RuneCountInString(current.String())

		// If adding this paragraph exceeds the limit
		if currentRunes+paraRunes+2 > s.maxChunkSize {
			// Flush current chunk if non-empty
			if currentRunes > 0 {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
			}
			// If the paragraph itself is too large, split at sentence boundaries
			if paraRunes > s.maxChunkSize {
				sentChunks := s.splitBySentences(para)
				chunks = append(chunks, sentChunks...)
			} else {
				current.WriteString(para)
			}
		} else {
			if currentRunes > 0 {
				current.WriteString("\n\n")
			}
			current.WriteString(para)
		}
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}

// splitBySentences splits oversized text at sentence boundaries (Chinese/English).
func (s *Splitter) splitBySentences(text string) []string {
	// Split on common sentence-ending punctuation
	sentences := splitSentenceBoundaries(text)
	var chunks []string
	var current strings.Builder

	for _, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}
		currentRunes := utf8.RuneCountInString(current.String())
		sentRunes := utf8.RuneCountInString(sent)

		if currentRunes+sentRunes+1 > s.maxChunkSize {
			if currentRunes > 0 {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
			}
			// If a single sentence is still too large, force-split by rune count
			if sentRunes > s.maxChunkSize {
				forceChunks := forceSplit(sent, s.maxChunkSize)
				chunks = append(chunks, forceChunks...)
			} else {
				current.WriteString(sent)
			}
		} else {
			if currentRunes > 0 {
				current.WriteString(" ")
			}
			current.WriteString(sent)
		}
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}
	return chunks
}

// splitSentenceBoundaries splits text at sentence-ending punctuation.
func splitSentenceBoundaries(text string) []string {
	var sentences []string
	runes := []rune(text)
	start := 0
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '。', '！', '？', '…', '.', '!', '?':
			// Include the punctuation in the sentence
			if i+1 < len(runes) && runes[i+1] == '\n' {
				sentences = append(sentences, string(runes[start:i+2]))
				start = i + 2
			} else {
				sentences = append(sentences, string(runes[start:i+1]))
				start = i + 1
			}
		}
	}
	if start < len(runes) {
		sentences = append(sentences, string(runes[start:]))
	}
	return sentences
}

// forceSplit breaks text into fixed-size rune chunks as a last resort.
func forceSplit(text string, maxRunes int) []string {
	runes := []rune(text)
	var chunks []string
	for len(runes) > 0 {
		end := maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[:end]))
		runes = runes[end:]
	}
	return chunks
}
