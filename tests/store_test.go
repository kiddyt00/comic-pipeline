package tests

import (
	"context"
	"testing"

	"github.com/kiddyt00/comic-pipeline/pkg/model"
	"github.com/kiddyt00/comic-pipeline/pkg/store"
)

func setupStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndListProjects(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "仙尊重生记", "修仙世界", "第一章...")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
	if p.Title != "仙尊重生记" {
		t.Errorf("title = %q, want %q", p.Title, "仙尊重生记")
	}
	if p.Status != model.StatusDraft {
		t.Errorf("status = %q, want draft", p.Status)
	}

	projects, err := s.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
}

func TestGetProjectNotFound(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	p, err := s.GetProject(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if p != nil {
		t.Error("expected nil for nonexistent project")
	}
}

func TestDeleteProject(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	p, err := s.CreateProject(ctx, "测试", "世界", "故事")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	err = s.DeleteProject(ctx, p.ID)
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	got, err := s.GetProject(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}
