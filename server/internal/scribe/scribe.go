package scribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const sttURL = "https://api.elevenlabs.io/v1/speech-to-text"
const model = "scribe_v1"

type Client struct {
	apiKey string
	http   *http.Client
}

type Transcript struct {
	Text         string  `json:"text"`
	LanguageCode string  `json:"language_code"`
	LanguageProb float64 `json:"language_probability"`
}

func New(apiKey string) *Client {
	return &Client{apiKey: apiKey, http: &http.Client{}}
}

func (c *Client) Transcribe(ctx context.Context, filename string, audio io.Reader) (Transcript, error) {
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)

	part, err := form.CreateFormFile("file", filename)
	if err != nil {
		return Transcript{}, fmt.Errorf("[scribe] form file: %w", err)
	}
	if _, err := io.Copy(part, audio); err != nil {
		return Transcript{}, fmt.Errorf("[scribe] copy audio: %w", err)
	}
	if err := form.WriteField("model_id", model); err != nil {
		return Transcript{}, fmt.Errorf("[scribe] write model_id: %w", err)
	}
	if err := form.Close(); err != nil {
		return Transcript{}, fmt.Errorf("[scribe] on close form: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sttURL, &buf)
	if err != nil {
		return Transcript{}, fmt.Errorf("[scribe] create request: %w", err)
	}
	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", form.FormDataContentType())

	res, err := c.http.Do(req)
	if err != nil {
		return Transcript{}, fmt.Errorf("[scribe] send request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return Transcript{}, fmt.Errorf("[scribe] http: %d, %s", res.StatusCode, body)
	}

	var out Transcript
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return Transcript{}, fmt.Errorf("[scribe] decode response: %w", err)
	}

	return out, nil
}
