package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kiddyt00/comic-pipeline/pkg/model"
)

// CreateEpisode 插入新 episode
func (s *Store) CreateEpisode(ctx context.Context, projectID string, episodeNum int) (*model.Episode, error) {
	now := time.Now().UTC()
	ep := &model.Episode{
		ID:         uuid.New().String(),
		ProjectID:  projectID,
		EpisodeNum: episodeNum,
		Status:     model.ScenePending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO episodes (id, project_id, episode_num, title, script_text,
		 scene_count, status, final_video_url, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ep.ID, ep.ProjectID, ep.EpisodeNum, ep.Title, ep.ScriptText,
		ep.SceneCount, string(ep.Status), ep.FinalVideoURL, ep.CreatedAt, ep.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert episode: %w", err)
	}

	return ep, nil
}

// ListEpisodes 按 episode_num 排序返回某项目的所有集
func (s *Store) ListEpisodes(ctx context.Context, projectID string) ([]model.Episode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, episode_num, title, script_text,
		 scene_count, status, final_video_url, created_at, updated_at
		 FROM episodes WHERE project_id = ? ORDER BY episode_num ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query episodes: %w", err)
	}
	defer rows.Close()

	var episodes []model.Episode
	for rows.Next() {
		var ep model.Episode
		if err := rows.Scan(
			&ep.ID, &ep.ProjectID, &ep.EpisodeNum, &ep.Title, &ep.ScriptText,
			&ep.SceneCount, &ep.Status, &ep.FinalVideoURL, &ep.CreatedAt, &ep.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan episode: %w", err)
		}
		episodes = append(episodes, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if episodes == nil {
		episodes = []model.Episode{}
	}
	return episodes, nil
}

// GetEpisode 返回单个 episode（不存在返回 nil, nil）
func (s *Store) GetEpisode(ctx context.Context, id string) (*model.Episode, error) {
	var ep model.Episode
	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, episode_num, title, script_text,
		 scene_count, status, final_video_url, created_at, updated_at
		 FROM episodes WHERE id = ?`, id,
	).Scan(
		&ep.ID, &ep.ProjectID, &ep.EpisodeNum, &ep.Title, &ep.ScriptText,
		&ep.SceneCount, &ep.Status, &ep.FinalVideoURL, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get episode: %w", err)
	}
	return &ep, nil
}

// UpsertScenes 批量插入或更新 scenes（ON CONFLICT episode_id+scene_num DO UPDATE）
func (s *Store) UpsertScenes(ctx context.Context, episodeID string, scenes []model.Scene) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	for i := range scenes {
		if scenes[i].ID == "" {
			scenes[i].ID = uuid.New().String()
		}
		if scenes[i].EpisodeID == "" {
			scenes[i].EpisodeID = episodeID
		}
		if scenes[i].CreatedAt.IsZero() {
			scenes[i].CreatedAt = now
		}
		scenes[i].UpdatedAt = now

		candidatesJSON := "[]"
		if len(scenes[i].ImageCandidates) > 0 {
			b, err := json.Marshal(scenes[i].ImageCandidates)
			if err != nil {
				return fmt.Errorf("marshal image_candidates: %w", err)
			}
			candidatesJSON = string(b)
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO scenes (id, episode_id, scene_num, description, dialogue,
			 duration_ms, status, image_candidates, selected_image, video_url,
			 audio_url, error_message, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(episode_id, scene_num) DO UPDATE SET
			 description = excluded.description,
			 dialogue = excluded.dialogue,
			 duration_ms = excluded.duration_ms,
			 status = excluded.status,
			 image_candidates = excluded.image_candidates,
			 selected_image = excluded.selected_image,
			 video_url = excluded.video_url,
			 audio_url = excluded.audio_url,
			 error_message = excluded.error_message,
			 updated_at = excluded.updated_at`,
			scenes[i].ID, scenes[i].EpisodeID, scenes[i].SceneNum,
			scenes[i].Description, scenes[i].Dialogue, scenes[i].DurationMs,
			string(scenes[i].Status), candidatesJSON, scenes[i].SelectedImage,
			scenes[i].VideoURL, scenes[i].AudioURL, scenes[i].ErrorMessage,
			scenes[i].CreatedAt, scenes[i].UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("upsert scene %d: %w", scenes[i].SceneNum, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// ListScenes 按 scene_num 排序返回某集的所有镜头
func (s *Store) ListScenes(ctx context.Context, episodeID string) ([]model.Scene, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, episode_id, scene_num, description, dialogue,
		 duration_ms, status, image_candidates, selected_image, video_url,
		 audio_url, error_message, created_at, updated_at
		 FROM scenes WHERE episode_id = ? ORDER BY scene_num ASC`, episodeID)
	if err != nil {
		return nil, fmt.Errorf("query scenes: %w", err)
	}
	defer rows.Close()

	var scenes []model.Scene
	for rows.Next() {
		var sc model.Scene
		var candidatesJSON string
		if err := rows.Scan(
			&sc.ID, &sc.EpisodeID, &sc.SceneNum, &sc.Description, &sc.Dialogue,
			&sc.DurationMs, &sc.Status, &candidatesJSON, &sc.SelectedImage,
			&sc.VideoURL, &sc.AudioURL, &sc.ErrorMessage, &sc.CreatedAt, &sc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan scene: %w", err)
		}
		if candidatesJSON != "" && candidatesJSON != "[]" {
			if err := json.Unmarshal([]byte(candidatesJSON), &sc.ImageCandidates); err != nil {
				return nil, fmt.Errorf("unmarshal image_candidates: %w", err)
			}
		}
		if sc.ImageCandidates == nil {
			sc.ImageCandidates = []string{}
		}
		scenes = append(scenes, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if scenes == nil {
		scenes = []model.Scene{}
	}
	return scenes, nil
}

// UpdateSceneStatus 更新单个 scene 的状态和相关字段
func (s *Store) UpdateSceneStatus(ctx context.Context, sceneID string, status model.SceneStatus, imageCandidates []string, selectedImage, videoURL, errMsg string) error {
	candidatesJSON := "[]"
	if len(imageCandidates) > 0 {
		b, err := json.Marshal(imageCandidates)
		if err != nil {
			return fmt.Errorf("marshal image_candidates: %w", err)
		}
		candidatesJSON = string(b)
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE scenes SET status = ?, image_candidates = ?, selected_image = ?,
		 video_url = ?, error_message = ?, updated_at = ?
		 WHERE id = ?`,
		string(status), candidatesJSON, selectedImage, videoURL, errMsg,
		time.Now().UTC(), sceneID,
	)
	if err != nil {
		return fmt.Errorf("update scene status: %w", err)
	}
	return nil
}

// UpdateEpisodeScript 设置剧本和 scene 数量
func (s *Store) UpdateEpisodeScript(ctx context.Context, episodeID, scriptText string, sceneCount int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE episodes SET script_text = ?, scene_count = ?, updated_at = ?
		 WHERE id = ?`,
		scriptText, sceneCount, time.Now().UTC(), episodeID,
	)
	if err != nil {
		return fmt.Errorf("update episode script: %w", err)
	}
	return nil
}

// UpdateEpisodeStatus 更新 episode 状态和最终视频 URL
func (s *Store) UpdateEpisodeStatus(ctx context.Context, episodeID string, status model.SceneStatus, finalVideoURL string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE episodes SET status = ?, final_video_url = ?, updated_at = ?
		 WHERE id = ?`,
		string(status), finalVideoURL, time.Now().UTC(), episodeID,
	)
	if err != nil {
		return fmt.Errorf("update episode status: %w", err)
	}
	return nil
}

// GetSceneByEpisodeAndNum 通过 episode_id + scene_num 定位 scene
func (s *Store) GetSceneByEpisodeAndNum(ctx context.Context, episodeID string, sceneNum int) (*model.Scene, error) {
	var sc model.Scene
	var candidatesJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, episode_id, scene_num, description, dialogue,
		 duration_ms, status, image_candidates, selected_image, video_url,
		 audio_url, error_message, created_at, updated_at
		 FROM scenes WHERE episode_id = ? AND scene_num = ?`, episodeID, sceneNum,
	).Scan(
		&sc.ID, &sc.EpisodeID, &sc.SceneNum, &sc.Description, &sc.Dialogue,
		&sc.DurationMs, &sc.Status, &candidatesJSON, &sc.SelectedImage,
		&sc.VideoURL, &sc.AudioURL, &sc.ErrorMessage, &sc.CreatedAt, &sc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get scene by episode and num: %w", err)
	}
	if candidatesJSON != "" && candidatesJSON != "[]" {
		if err := json.Unmarshal([]byte(candidatesJSON), &sc.ImageCandidates); err != nil {
			return nil, fmt.Errorf("unmarshal image_candidates: %w", err)
		}
	}
	if sc.ImageCandidates == nil {
		sc.ImageCandidates = []string{}
	}
	return &sc, nil
}
