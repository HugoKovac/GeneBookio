package tts

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitForTTS_ShortTextIsOneChunk(t *testing.T) {
	chunks := splitForTTS("a short sentence")
	if len(chunks) != 1 || chunks[0] != "a short sentence" {
		t.Fatalf("expected a single unchanged chunk, got %v", chunks)
	}
}

func TestSplitForTTS_RespectsTheCharacterLimit(t *testing.T) {
	text := strings.Repeat("word ", 2000) // 10,000 chars, well past the limit
	chunks := splitForTTS(text)

	if len(chunks) < 2 {
		t.Fatalf("expected text longer than %d chars to be split, got %d chunk(s)", maxTTSInputChars, len(chunks))
	}
	for i, c := range chunks {
		if n := utf8.RuneCountInString(c); n > maxTTSInputChars {
			t.Errorf("chunk %d has %d runes, want <= %d", i, n, maxTTSInputChars)
		}
	}
}

func TestSplitForTTS_DoesNotSplitAWordInHalf(t *testing.T) {
	text := strings.Repeat("word ", 2000)
	chunks := splitForTTS(text)

	for i, c := range chunks {
		trimmed := strings.TrimSpace(c)
		if trimmed == "" {
			continue
		}
		for _, word := range strings.Fields(trimmed) {
			if word != "word" {
				t.Errorf("chunk %d contains a mangled word %q (expected only whole occurrences of %q)", i, word, "word")
			}
		}
	}
}

func TestSplitForTTS_PreservesAllNonSpaceContent(t *testing.T) {
	text := strings.Repeat("word ", 2000)
	chunks := splitForTTS(text)

	var rebuilt strings.Builder
	for _, c := range chunks {
		rebuilt.WriteString(c)
		rebuilt.WriteByte(' ')
	}

	wantWords := strings.Fields(text)
	gotWords := strings.Fields(rebuilt.String())
	if len(gotWords) != len(wantWords) {
		t.Fatalf("word count changed across the split: got %d, want %d", len(gotWords), len(wantWords))
	}
}

func TestSplitForTTS_HardSplitsAWordLongerThanTheLimit(t *testing.T) {
	text := strings.Repeat("x", maxTTSInputChars+100)
	chunks := splitForTTS(text)

	if len(chunks) != 2 {
		t.Fatalf("expected the oversized word to be hard-split into 2 chunks, got %d", len(chunks))
	}
	if utf8.RuneCountInString(chunks[0]) != maxTTSInputChars {
		t.Errorf("first chunk has %d runes, want exactly %d", utf8.RuneCountInString(chunks[0]), maxTTSInputChars)
	}
}

func TestSplitForTTS_EmptyText(t *testing.T) {
	chunks := splitForTTS("")
	if len(chunks) != 1 || chunks[0] != "" {
		t.Fatalf("expected a single empty chunk, got %v", chunks)
	}
}
