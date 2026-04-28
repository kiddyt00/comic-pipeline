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

func TestCreateEpisodeAndScenes(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()
	p, err := s.CreateProject(ctx, "测试", "世界观", "故事")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	// 测试 CreateEpisode
	ep, err := s.CreateEpisode(ctx, p.ID, 1)
	if err != nil {
		t.Fatalf("CreateEpisode: %v", err)
	}
	if ep.ID == "" {
		t.Error("expected non-empty episode ID")
	}
	if ep.EpisodeNum != 1 {
		t.Errorf("episode_num = %d, want 1", ep.EpisodeNum)
	}
	if ep.ProjectID != p.ID {
		t.Errorf("project_id = %q, want %q", ep.ProjectID, p.ID)
	}
	if ep.Status != model.ScenePending {
		t.Errorf("status = %q, want pending", ep.Status)
	}

	// 测试 ListEpisodes
	eps, err := s.ListEpisodes(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListEpisodes: %v", err)
	}
	if len(eps) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps))
	}
	if eps[0].EpisodeNum != 1 {
		t.Errorf("episode_num = %d, want 1", eps[0].EpisodeNum)
	}

	// 测试 GetEpisode
	got, err := s.GetEpisode(ctx, ep.ID)
	if err != nil {
		t.Fatalf("GetEpisode: %v", err)
	}
	if got == nil {
		t.Fatal("expected episode, got nil")
	}
	if got.ID != ep.ID {
		t.Errorf("id = %q, want %q", got.ID, ep.ID)
	}

	// 测试 GetEpisode 不存在
	notFound, err := s.GetEpisode(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetEpisode nonexistent: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for nonexistent episode")
	}

	// 测试 UpsertScenes
	scenes := []model.Scene{
		{SceneNum: 1, Description: "主角登场", Status: model.ScenePending},
		{SceneNum: 2, Description: "冲突爆发", Status: model.ScenePending},
	}
	err = s.UpsertScenes(ctx, ep.ID, scenes)
	if err != nil {
		t.Fatalf("UpsertScenes: %v", err)
	}

	// 测试 ListScenes
	gotScenes, err := s.ListScenes(ctx, ep.ID)
	if err != nil {
		t.Fatalf("ListScenes: %v", err)
	}
	if len(gotScenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(gotScenes))
	}
	if gotScenes[0].SceneNum != 1 {
		t.Errorf("scene 0 num = %d, want 1", gotScenes[0].SceneNum)
	}
	if gotScenes[1].SceneNum != 2 {
		t.Errorf("scene 1 num = %d, want 2", gotScenes[1].SceneNum)
	}
	if gotScenes[0].Description != "主角登场" {
		t.Errorf("scene 0 description = %q, want 主角登场", gotScenes[0].Description)
	}

	// 测试 UpsertScenes - update existing
	updatedScenes := []model.Scene{
		{SceneNum: 1, Description: "主角登场-改", Status: model.SceneImageDone,
			ImageCandidates: []string{"img1.png", "img2.png"}},
		{SceneNum: 3, Description: "新镜头", Status: model.ScenePending},
	}
	err = s.UpsertScenes(ctx, ep.ID, updatedScenes)
	if err != nil {
		t.Fatalf("UpsertScenes update: %v", err)
	}

	// 验证 upsert 结果：3 个 scene（1 updated, 2 untouched, 3 new）
	gotScenes, err = s.ListScenes(ctx, ep.ID)
	if err != nil {
		t.Fatalf("ListScenes after upsert: %v", err)
	}
	if len(gotScenes) != 3 {
		t.Fatalf("expected 3 scenes after upsert, got %d", len(gotScenes))
	}
	// 查找 scene 1，验证它被更新了
	var scene1 *model.Scene
	for i := range gotScenes {
		if gotScenes[i].SceneNum == 1 {
			scene1 = &gotScenes[i]
			break
		}
	}
	if scene1 == nil {
		t.Fatal("scene 1 not found after upsert")
	}
	if scene1.Description != "主角登场-改" {
		t.Errorf("scene 1 description = %q, want 主角登场-改", scene1.Description)
	}
	if scene1.Status != model.SceneImageDone {
		t.Errorf("scene 1 status = %q, want image_done", scene1.Status)
	}
	if len(scene1.ImageCandidates) != 2 {
		t.Errorf("scene 1 image_candidates len = %d, want 2", len(scene1.ImageCandidates))
	}

	// 测试 GetSceneByEpisodeAndNum
	sc, err := s.GetSceneByEpisodeAndNum(ctx, ep.ID, 1)
	if err != nil {
		t.Fatalf("GetSceneByEpisodeAndNum: %v", err)
	}
	if sc == nil {
		t.Fatal("expected scene, got nil")
	}
	if sc.SceneNum != 1 {
		t.Errorf("scene_num = %d, want 1", sc.SceneNum)
	}

	// 测试 GetSceneByEpisodeAndNum 不存在
	scNotFound, err := s.GetSceneByEpisodeAndNum(ctx, ep.ID, 999)
	if err != nil {
		t.Fatalf("GetSceneByEpisodeAndNum nonexistent: %v", err)
	}
	if scNotFound != nil {
		t.Error("expected nil for nonexistent scene")
	}

	// 测试 UpdateSceneStatus
	err = s.UpdateSceneStatus(ctx, scene1.ID, model.SceneComplete,
		[]string{"selected.png"}, "selected.png", "video.mp4", "")
	if err != nil {
		t.Fatalf("UpdateSceneStatus: %v", err)
	}
	sc, err = s.GetSceneByEpisodeAndNum(ctx, ep.ID, 1)
	if err != nil {
		t.Fatalf("GetSceneByEpisodeAndNum after update: %v", err)
	}
	if sc.Status != model.SceneComplete {
		t.Errorf("status = %q, want complete", sc.Status)
	}
	if sc.SelectedImage != "selected.png" {
		t.Errorf("selected_image = %q, want selected.png", sc.SelectedImage)
	}
	if sc.VideoURL != "video.mp4" {
		t.Errorf("video_url = %q, want video.mp4", sc.VideoURL)
	}

	// 测试 UpdateEpisodeScript
	err = s.UpdateEpisodeScript(ctx, ep.ID, "剧本内容...", 3)
	if err != nil {
		t.Fatalf("UpdateEpisodeScript: %v", err)
	}
	ep, err = s.GetEpisode(ctx, ep.ID)
	if err != nil {
		t.Fatalf("GetEpisode after script update: %v", err)
	}
	if ep.ScriptText != "剧本内容..." {
		t.Errorf("script_text = %q, want 剧本内容...", ep.ScriptText)
	}
	if ep.SceneCount != 3 {
		t.Errorf("scene_count = %d, want 3", ep.SceneCount)
	}

	// 测试 UpdateEpisodeStatus
	err = s.UpdateEpisodeStatus(ctx, ep.ID, model.SceneComplete, "final.mp4")
	if err != nil {
		t.Fatalf("UpdateEpisodeStatus: %v", err)
	}
	ep, err = s.GetEpisode(ctx, ep.ID)
	if err != nil {
		t.Fatalf("GetEpisode after status update: %v", err)
	}
	if ep.Status != model.SceneComplete {
		t.Errorf("status = %q, want complete", ep.Status)
	}
	if ep.FinalVideoURL != "final.mp4" {
		t.Errorf("final_video_url = %q, want final.mp4", ep.FinalVideoURL)
	}

	// 验证 CASCADE：删除 project 会导致 episode 和 scenes 也被删除
	epID := ep.ID
	err = s.DeleteProject(ctx, p.ID)
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	deletedEp, err := s.GetEpisode(ctx, epID)
	if err != nil {
		t.Fatalf("GetEpisode after cascade: %v", err)
	}
	if deletedEp != nil {
		t.Error("expected nil episode after project delete (cascade)")
	}
	scenesAfterCascade, err := s.ListScenes(ctx, epID)
	if err != nil {
		t.Fatalf("ListScenes after cascade: %v", err)
	}
	if len(scenesAfterCascade) != 0 {
		t.Errorf("expected 0 scenes after cascade, got %d", len(scenesAfterCascade))
	}
}
