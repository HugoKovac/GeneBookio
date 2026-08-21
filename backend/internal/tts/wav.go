package tts

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// wavAudio holds the two chunks of a WAV file that matter for
// concatenation: the format description and the raw PCM payload.
type wavAudio struct {
	fmtChunk []byte
	data     []byte
}

// parseWAV extracts the "fmt " and "data" chunks from a RIFF/WAVE file,
// skipping any other chunks (e.g. metadata) a producer may have included.
func parseWAV(b []byte) (*wavAudio, error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a valid WAV file")
	}

	var w wavAudio
	pos := 12
	for pos+8 <= len(b) {
		id := string(b[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))

		dataStart := pos + 8
		dataEnd := min(dataStart+size, len(b))

		switch id {
		case "fmt ":
			w.fmtChunk = b[dataStart:dataEnd]
		case "data":
			w.data = b[dataStart:dataEnd]
		}

		// Chunks are word-aligned: a chunk with an odd size is padded with
		// one extra byte that isn't reflected in its declared size.
		pos = dataEnd
		if size%2 == 1 {
			pos++
		}
	}

	if w.fmtChunk == nil || w.data == nil {
		return nil, fmt.Errorf("WAV file is missing its fmt or data chunk")
	}
	return &w, nil
}

// buildWAV assembles a single-fmt, single-data RIFF/WAVE file.
func buildWAV(fmtChunk, data []byte) []byte {
	buf := new(bytes.Buffer)

	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(4+8+len(fmtChunk)+8+len(data)))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(len(fmtChunk)))
	buf.Write(fmtChunk)

	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(len(data)))
	buf.Write(data)

	return buf.Bytes()
}
