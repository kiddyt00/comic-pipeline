package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kiddyt00/comic-pipeline/pkg/model"
	"github.com/kiddyt00/comic-pipeline/pkg/n8n"
)

func (srv *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	render(w, "index.html", nil)
}

func (srv *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := srv.store.ListProjects(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, "_project_list.html", projects)
}

// 其他 handler 先放骨架（返回 501 Not Implemented），后续任务实现
func (srv *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	worldSetting := r.FormValue("world_setting")
	storyText := r.FormValue("story_text")

	if title == "" || storyText == "" {
		http.Error(w, "标题和故事文本为必填", http.StatusBadRequest)
		return
	}

	_, err := srv.store.CreateProject(r.Context(), title, worldSetting, storyText)
	if err != nil {
		http.Error(w, fmt.Sprintf("创建项目失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回更新后的项目列表片段
	projects, _ := srv.store.ListProjects(r.Context())
	render(w, "_project_list.html", projects)
}

func (srv *Server) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, err := srv.store.GetProject(r.Context(), id)
	if err != nil || project == nil {
		http.Error(w, "项目不存在", http.StatusNotFound)
		return
	}
	episodes, _ := srv.store.ListEpisodes(r.Context(), id)

	var currentEp *model.Episode
	epID := r.URL.Query().Get("ep")
	for i, ep := range episodes {
		if ep.ID == epID {
			currentEp = &episodes[i]
			break
		}
	}
	if currentEp == nil && len(episodes) > 0 {
		currentEp = &episodes[0]
	}

	var scenes []model.Scene
	if currentEp != nil {
		scenes, _ = srv.store.ListScenes(r.Context(), currentEp.ID)
	}

	data := struct {
		Project   *model.Project
		Episodes  []model.Episode
		CurrentEp *model.Episode
		Scenes    []model.Scene
	}{project, episodes, currentEp, scenes}
	render(w, "project.html", data)
}

func (srv *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := srv.store.DeleteProject(r.Context(), id); err != nil {
		http.Error(w, "删除失败", http.StatusInternalServerError)
		return
	}
	projects, _ := srv.store.ListProjects(r.Context())
	render(w, "_project_list.html", projects)
}

func (srv *Server) handleCreateEpisode(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	project, err := srv.store.GetProject(r.Context(), projectID)
	if err != nil || project == nil {
		http.Error(w, "项目不存在", http.StatusNotFound)
		return
	}

	nextNum := project.EpisodeCount + 1
	_, err = srv.store.CreateEpisode(r.Context(), projectID, nextNum)
	if err != nil {
		http.Error(w, fmt.Sprintf("创建剧集失败: %v", err), http.StatusInternalServerError)
		return
	}

	srv.store.UpdateProjectStatus(r.Context(), projectID, model.StatusProducing, nextNum)

	http.Redirect(w, r, "/projects/"+projectID, http.StatusSeeOther)
}
func (srv *Server) handleTriggerPipeline(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	epID := r.PathValue("epId")

	project, err := srv.store.GetProject(r.Context(), projectID)
	if err != nil || project == nil {
		http.Error(w, "项目不存在", http.StatusNotFound)
		return
	}

	ep, err := srv.store.GetEpisode(r.Context(), epID)
	if err != nil || ep == nil {
		http.Error(w, "剧集不存在", http.StatusNotFound)
		return
	}

	callbackURL := fmt.Sprintf("http://%s/internal/callback", r.Host)

	payload := n8n.TriggerPayload{
		ProjectID:     projectID,
		EpisodeID:     epID,
		EpisodeNum:    ep.EpisodeNum,
		WorldSetting:  project.WorldSetting,
		StoryText:     project.StoryText,
		CallbackURL:   callbackURL,
		ImageProvider: "jimeng",
		VideoProvider: "kling",
		TTSProvider:   "edge-tts",
		OutputFormat:  "bilibili_1080p",
	}

	if err := srv.n8nClient.TriggerEpisode(r.Context(), payload); err != nil {
		http.Error(w, fmt.Sprintf("触发流水线失败: %v", err), http.StatusInternalServerError)
		return
	}

	srv.store.UpdateEpisodeStatus(r.Context(), epID, model.SceneScriptDone, "")

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

type callbackBody struct {
	ProjectID string `json:"project_id"`
	EpisodeID string `json:"episode_id"`
	Event     string `json:"event"` // script_done | scene_image_done | scene_video_done | tts_done | episode_complete | error
	SceneNum  int    `json:"scene_num,omitempty"`
	Data      struct {
		ScriptText    string        `json:"script_text,omitempty"`
		SceneCount    int           `json:"scene_count,omitempty"`
		Scenes        []model.Scene `json:"scenes,omitempty"`
		Candidates    []string      `json:"candidates,omitempty"`
		VideoURL      string        `json:"video_url,omitempty"`
		AudioURL      string        `json:"audio_url,omitempty"`
		FinalVideoURL string        `json:"final_video_url,omitempty"`
		Error         string        `json:"error,omitempty"`
	} `json:"data"`
}

func (srv *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	var cb callbackBody
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	switch cb.Event {
	case "script_done":
		srv.store.UpdateEpisodeScript(r.Context(), cb.EpisodeID, cb.Data.ScriptText, cb.Data.SceneCount)
		srv.store.UpsertScenes(r.Context(), cb.EpisodeID, cb.Data.Scenes)
	case "scene_image_done":
		scene, err := srv.store.GetSceneByEpisodeAndNum(r.Context(), cb.EpisodeID, cb.SceneNum)
		if err == nil && scene != nil {
			srv.store.UpdateSceneStatus(r.Context(), scene.ID, model.SceneImageDone,
				cb.Data.Candidates, "", "", "")
		}
	case "scene_video_done":
		scene, err := srv.store.GetSceneByEpisodeAndNum(r.Context(), cb.EpisodeID, cb.SceneNum)
		if err == nil && scene != nil {
			srv.store.UpdateSceneStatus(r.Context(), scene.ID, model.SceneVideoDone,
				nil, "", cb.Data.VideoURL, "")
		}
	case "tts_done":
		// TTS complete — could update scene audio URLs
	case "episode_complete":
		srv.store.UpdateEpisodeStatus(r.Context(), cb.EpisodeID, model.SceneComplete, cb.Data.FinalVideoURL)
	case "error":
		if cb.SceneNum > 0 {
			scene, err := srv.store.GetSceneByEpisodeAndNum(r.Context(), cb.EpisodeID, cb.SceneNum)
			if err == nil && scene != nil {
				srv.store.UpdateSceneStatus(r.Context(), scene.ID, model.SceneFailed,
					nil, "", "", cb.Data.Error)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}
func (srv *Server) handleServeOutput(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix("/output/", http.FileServer(http.Dir(srv.outputDir))).ServeHTTP(w, r)
}

func (srv *Server) handleScenesFragment(w http.ResponseWriter, r *http.Request) {
	epID := r.PathValue("epId")
	scenes, _ := srv.store.ListScenes(r.Context(), epID)
	render(w, "_scene_cards.html", scenes)
}
