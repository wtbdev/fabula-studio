package tree

import (
	"strings"
	"testing"
)

func TestSplitChapters_ShortText(t *testing.T) {
	s := NewSplitter(100)
	tree := s.SplitChapters([]string{"这是一段很短的文本。"})

	// Should have root + 1 leaf
	if len(tree.Nodes) != 2 {
		t.Fatalf("expected 2 nodes (root + 1 leaf), got %d", len(tree.Nodes))
	}
	leaves := tree.CollectLeaves()
	if len(leaves) != 1 {
		t.Fatalf("expected 1 leaf, got %d", len(leaves))
	}
	if leaves[0].SourceChapter != 0 {
		t.Errorf("expected source chapter 0, got %d", leaves[0].SourceChapter)
	}
}

func TestSplitChapters_MultipleChapters(t *testing.T) {
	s := NewSplitter(100)
	chapters := []string{
		"第一章内容。",
		"第二章内容。",
		"第三章内容。",
	}
	tree := s.SplitChapters(chapters)
	leaves := tree.CollectLeaves()
	if len(leaves) != 3 {
		t.Fatalf("expected 3 leaves, got %d", len(leaves))
	}
	for i, leaf := range leaves {
		if leaf.SourceChapter != i {
			t.Errorf("leaf %d: expected source chapter %d, got %d", i, i, leaf.SourceChapter)
		}
	}
}

func TestSplitChapters_LongTextSplitsAtParagraphs(t *testing.T) {
	s := NewSplitter(50)
	// Create text with paragraphs that will exceed the limit
	text := strings.Repeat("这是一段测试文本。", 5) + "\n\n" + strings.Repeat("这是另一段测试文本。", 5)
	tree := s.SplitChapters([]string{text})
	leaves := tree.CollectLeaves()
	if len(leaves) < 2 {
		t.Fatalf("expected at least 2 leaves for long text, got %d", len(leaves))
	}
}

func TestSplitChapters_RightNeighbor(t *testing.T) {
	s := NewSplitter(10)
	text := "这是第一段文本内容。" + "\n\n" + "这是第二段文本内容。" + "\n\n" + "这是第三段文本内容。"
	tree := s.SplitChapters([]string{text})
	leaves := tree.CollectLeaves()
	if len(leaves) < 2 {
		t.Fatalf("expected at least 2 leaves, got %d", len(leaves))
	}
	// Verify right neighbor chain
	for i := 0; i < len(leaves)-1; i++ {
		if leaves[i].RightNeighbor != leaves[i+1].ID {
			t.Errorf("leaf %d right neighbor: expected %s, got %s", i, leaves[i+1].ID, leaves[i].RightNeighbor)
		}
	}
}

func TestSplitSentenceBoundaries(t *testing.T) {
	sentences := splitSentenceBoundaries("你好。世界！再见？")
	if len(sentences) != 3 {
		t.Fatalf("expected 3 sentences, got %d: %v", len(sentences), sentences)
	}
}

func TestForceSplit(t *testing.T) {
	chunks := forceSplit("abcdefghij", 3)
	expected := []string{"abc", "def", "ghi", "j"}
	if len(chunks) != len(expected) {
		t.Fatalf("expected %d chunks, got %d", len(expected), len(chunks))
	}
	for i, c := range chunks {
		if c != expected[i] {
			t.Errorf("chunk %d: expected %q, got %q", i, expected[i], c)
		}
	}
}
