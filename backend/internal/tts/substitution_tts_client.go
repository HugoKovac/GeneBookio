package tts

import (
	"bytes"
	"context"
	"encoding/binary"
	"hkorpo/book/internal/primitive"
	"io"
)

// SubstitutionTTSClient stands in for OpenAiTTSClient when ConfigAi.TEST_MODE
// is enabled, so the pipeline can be exercised without spending real money on
// audio synthesis. It returns a tiny silent WAV file instead of calling
// OpenAI.
type SubstitutionTTSClient struct{}

func NewSubstitutionTTSClient() *SubstitutionTTSClient {
	return &SubstitutionTTSClient{}
}

func (sc *SubstitutionTTSClient) ModelName() string {
	return "test-mode-tts"
}

func (sc *SubstitutionTTSClient) CreateAudioFromString(_ context.Context, content string, _ primitive.Language) (io.ReadCloser, int64, primitive.ModelUsage, error) {
	wav := silentWAV()
	usage := primitive.ModelUsage{InputTokens: int64(len([]rune(content)))}
	return io.NopCloser(bytes.NewReader(wav)), int64(len(wav)), usage, nil
}

// silentWAV builds a minimal, valid 16-bit mono PCM WAV file containing a
// tenth of a second of silence at 8kHz.
func silentWAV() []byte {
	const (
		sampleRate    = 8000
		bitsPerSample = 16
		channels      = 1
		numSamples    = sampleRate / 10
	)

	dataSize := numSamples * channels * (bitsPerSample / 8)
	buf := new(bytes.Buffer)

	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(channels))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate*channels*bitsPerSample/8))
	binary.Write(buf, binary.LittleEndian, uint16(channels*bitsPerSample/8))
	binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))

	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(make([]byte, dataSize))

	return buf.Bytes()
}
