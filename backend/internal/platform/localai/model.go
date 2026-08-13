package localai

import "net/http"

type ConfigLocalAI struct {
	LOCAL_AI_HOST string `envconfig:"LOCAL_AI_HOST"`
}

type Client struct {
	host   string
	client *http.Client
}

type Endpoint string

const (
	TTS Endpoint = "/tts"
)
