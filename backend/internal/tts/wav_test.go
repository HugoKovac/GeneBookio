package tts

import (
	"bytes"
	"testing"
)

// makeTestWAV builds a minimal WAV file (like silentWAV, but with real
// content instead of silence) for round-trip testing.
func makeTestWAV(t *testing.T, fmtChunk, data []byte) []byte {
	t.Helper()
	return buildWAV(fmtChunk, data)
}

func TestParseWAV_RoundTrip(t *testing.T) {
	fmtChunk := []byte{1, 0, 1, 0, 0x40, 0x1f, 0, 0, 0x80, 0x3e, 0, 0, 2, 0, 16, 0}
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	wavBytes := makeTestWAV(t, fmtChunk, data)

	parsed, err := parseWAV(wavBytes)
	if err != nil {
		t.Fatalf("parseWAV returned an error: %v", err)
	}
	if !bytes.Equal(parsed.fmtChunk, fmtChunk) {
		t.Errorf("fmtChunk = %v, want %v", parsed.fmtChunk, fmtChunk)
	}
	if !bytes.Equal(parsed.data, data) {
		t.Errorf("data = %v, want %v", parsed.data, data)
	}
}

func TestParseWAV_RejectsNonWAV(t *testing.T) {
	if _, err := parseWAV([]byte("not a wav file at all")); err == nil {
		t.Error("expected an error for non-WAV input")
	}
}

func TestParseWAV_OddSizedChunkIsPadded(t *testing.T) {
	// An odd-length "data" chunk must be followed by one pad byte that's
	// not part of the chunk's declared size — parseWAV must skip it rather
	// than reading it as the start of the next chunk.
	fmtChunk := []byte{1, 2, 3, 4}
	data := []byte{1, 2, 3} // odd length: 3 bytes

	var buf bytes.Buffer
	buf.WriteString("RIFF")
	buf.Write([]byte{0, 0, 0, 0}) // placeholder RIFF size, unused by parseWAV
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	buf.Write([]byte{4, 0, 0, 0})
	buf.Write(fmtChunk)
	buf.WriteString("data")
	buf.Write([]byte{3, 0, 0, 0})
	buf.Write(data)
	buf.WriteByte(0) // pad byte
	// A trailing chunk after the padded one, to prove the pad byte was skipped.
	buf.WriteString("LIST")
	buf.Write([]byte{0, 0, 0, 0})

	parsed, err := parseWAV(buf.Bytes())
	if err != nil {
		t.Fatalf("parseWAV returned an error: %v", err)
	}
	if !bytes.Equal(parsed.data, data) {
		t.Errorf("data = %v, want %v", parsed.data, data)
	}
}

func TestConcatenateWAVSegments(t *testing.T) {
	fmtChunk := []byte{1, 0, 1, 0, 0x40, 0x1f, 0, 0, 0x80, 0x3e, 0, 0, 2, 0, 16, 0}

	seg1 := makeTestWAV(t, fmtChunk, []byte{1, 2, 3, 4})
	seg2 := makeTestWAV(t, fmtChunk, []byte{5, 6, 7, 8})

	w1, err := parseWAV(seg1)
	if err != nil {
		t.Fatalf("parseWAV(seg1): %v", err)
	}
	w2, err := parseWAV(seg2)
	if err != nil {
		t.Fatalf("parseWAV(seg2): %v", err)
	}

	var combinedData bytes.Buffer
	combinedData.Write(w1.data)
	combinedData.Write(w2.data)
	combined := buildWAV(w1.fmtChunk, combinedData.Bytes())

	reparsed, err := parseWAV(combined)
	if err != nil {
		t.Fatalf("parseWAV(combined): %v", err)
	}
	want := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	if !bytes.Equal(reparsed.data, want) {
		t.Errorf("combined data = %v, want %v", reparsed.data, want)
	}
}
