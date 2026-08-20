package ttsapi

import "net/http"

type ConfigTTSAPI struct {
	TTS_API_HOST string `envconfig:"TTS_API_HOST"`
}

type Client struct {
	host   string
	client *http.Client
}

type Endpoint string

const (
	TTS Endpoint = "/tts"
)
