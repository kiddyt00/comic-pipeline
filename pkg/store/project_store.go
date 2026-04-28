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

func (s *Store) CreateProject(ctx context.Context, title, worldSetting, storyText string) (*model.Project, error) {
	now := time.Now().UTC()
	p := &model.Project{
		ID:           uuid.New().String(),
		Title:        title,
		WorldSetting: worldSetting,
		StoryText:    storyText,
		Tags:         []string{},
		Status:       model.StatusDraft,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	tagsJSON, err := json.Marshal(p.Tags)
	if err != nil {
		return nil, fmt.Errorf("marshal tags: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO projects (id, title, world_setting, story_text, cover_url, tags, status,
		 character_count, episode_count, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Title, p.WorldSetting, p.StoryText, p.CoverURL, string(tagsJSON),
		string(p.Status), p.CharacterCount, p.EpisodeCount, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}

	return p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, world_setting, story_text, cover_url, tags, status,
		 character_count, episode_count, created_at, updated_at
		 FROM projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var tagsJSON string
		if err := rows.Scan(
			&p.ID, &p.Title, &p.WorldSetting, &p.StoryText, &p.CoverURL,
			&tagsJSON, &p.Status, &p.CharacterCount, &p.EpisodeCount,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		if tagsJSON != "" {
			if err := json.Unmarshal([]byte(tagsJSON), &p.Tags); err != nil {
				return nil, fmt.Errorf("unmarshal tags: %w", err)
			}
		}
		if p.Tags == nil {
			p.Tags = []string{}
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if projects == nil {
		projects = []model.Project{}
	}
	return projects, nil
}

func (s *Store) GetProject(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	var tagsJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, world_setting, story_text, cover_url, tags, status,
		 character_count, episode_count, created_at, updated_at
		 FROM projects WHERE id = ?`, id,
	).Scan(
		&p.ID, &p.Title, &p.WorldSetting, &p.StoryText, &p.CoverURL,
		&tagsJSON, &p.Status, &p.CharacterCount, &p.EpisodeCount,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &p.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return &p, nil
}

func (s *Store) UpdateProjectStatus(ctx context.Context, id string, status model.ProjectStatus, episodeCount int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE projects SET status = ?, episode_count = ?, updated_at = ? WHERE id = ?`,
		string(status), episodeCount, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("update project status: %w", err)
	}
	return nil
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
