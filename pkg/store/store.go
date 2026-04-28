package store

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		world_setting TEXT NOT NULL DEFAULT '',
		story_text TEXT NOT NULL DEFAULT '',
		cover_url TEXT DEFAULT '',
		tags TEXT DEFAULT '[]',
		status TEXT NOT NULL DEFAULT 'draft',
		character_count INTEGER NOT NULL DEFAULT 0,
		episode_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS characters (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		ref_image_url TEXT DEFAULT '',
		lora_path TEXT DEFAULT '',
		feature_tags TEXT DEFAULT '[]',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS episodes (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		episode_num INTEGER NOT NULL,
		title TEXT DEFAULT '',
		script_text TEXT DEFAULT '',
		scene_count INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'pending',
		final_video_url TEXT DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
		UNIQUE(project_id, episode_num)
	);

	CREATE TABLE IF NOT EXISTS scenes (
		id TEXT PRIMARY KEY,
		episode_id TEXT NOT NULL,
		scene_num INTEGER NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		dialogue TEXT DEFAULT '',
		duration_ms INTEGER DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'pending',
		image_candidates TEXT DEFAULT '[]',
		selected_image TEXT DEFAULT '',
		video_url TEXT DEFAULT '',
		audio_url TEXT DEFAULT '',
		error_message TEXT DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE,
		UNIQUE(episode_id, scene_num)
	);

	CREATE INDEX IF NOT EXISTS idx_episodes_project ON episodes(project_id);
	CREATE INDEX IF NOT EXISTS idx_scenes_episode ON scenes(episode_id);
	CREATE INDEX IF NOT EXISTS idx_characters_project ON characters(project_id);
	`
	_, err := s.db.Exec(schema)
	return err
}
