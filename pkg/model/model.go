package model

import "time"

type ProjectStatus string

const (
	StatusDraft     ProjectStatus = "draft"
	StatusProducing ProjectStatus = "producing"
	StatusComplete  ProjectStatus = "complete"
)

type Project struct {
	ID             string        `json:"id"`
	Title          string        `json:"title"`
	WorldSetting   string        `json:"world_setting"`
	StoryText      string        `json:"story_text"`
	CoverURL       string        `json:"cover_url,omitempty"`
	Tags           []string      `json:"tags,omitempty"`
	Status         ProjectStatus `json:"status"`
	CharacterCount int           `json:"character_count"`
	EpisodeCount   int           `json:"episode_count"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type SceneStatus string

const (
	ScenePending    SceneStatus = "pending"
	SceneScriptDone SceneStatus = "script_done"
	SceneImageGen   SceneStatus = "image_gen"
	SceneImageDone  SceneStatus = "image_done"
	SceneVideoGen   SceneStatus = "video_gen"
	SceneVideoDone  SceneStatus = "video_done"
	SceneTTSPending SceneStatus = "tts_pending"
	SceneComplete   SceneStatus = "complete"
	SceneFailed     SceneStatus = "failed"
)

type Character struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RefImageURL string    `json:"ref_image_url,omitempty"`
	LoraPath    string    `json:"lora_path,omitempty"`
	FeatureTags []string  `json:"feature_tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Episode struct {
	ID            string      `json:"id"`
	ProjectID     string      `json:"project_id"`
	EpisodeNum    int         `json:"episode_num"`
	Title         string      `json:"title,omitempty"`
	ScriptText    string      `json:"script_text,omitempty"`
	SceneCount    int         `json:"scene_count"`
	Status        SceneStatus `json:"status"`
	FinalVideoURL string      `json:"final_video_url,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type Scene struct {
	ID              string      `json:"id"`
	EpisodeID       string      `json:"episode_id"`
	SceneNum        int         `json:"scene_num"`
	Description     string      `json:"description"`
	Dialogue        string      `json:"dialogue,omitempty"`
	DurationMs      int         `json:"duration_ms,omitempty"`
	Status          SceneStatus `json:"status"`
	ImageCandidates []string    `json:"image_candidates,omitempty"`
	SelectedImage   string      `json:"selected_image,omitempty"`
	VideoURL        string      `json:"video_url,omitempty"`
	AudioURL        string      `json:"audio_url,omitempty"`
	ErrorMessage    string      `json:"error_message,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}
