package n8n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type TriggerPayload struct {
	ProjectID     string         `json:"project_id"`
	EpisodeID     string         `json:"episode_id"`
	EpisodeNum    int            `json:"episode_num"`
	Title         string         `json:"title,omitempty"`
	WorldSetting  string         `json:"world_setting"`
	StoryText     string         `json:"story_text"`
	Characters    []CharacterRef `json:"characters"`
	CallbackURL   string         `json:"callback_url"`
	ImageProvider string         `json:"image_provider"`
	VideoProvider string         `json:"video_provider"`
	TTSProvider   string         `json:"tts_provider"`
	OutputFormat  string         `json:"output_format"`
}

type CharacterRef struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	RefImageURL string   `json:"ref_image_url,omitempty"`
	FeatureTags []string `json:"feature_tags,omitempty"`
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) TriggerEpisode(ctx context.Context, payload TriggerPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := c.baseURL + "/webhook/episode-pipeline"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("n8n returned status %d", resp.StatusCode)
	}
	return nil
}
