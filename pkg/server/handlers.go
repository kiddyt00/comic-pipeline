package server

import (
	"fmt"
	"net/http"
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
	data := struct {
		Project  interface{}
		Episodes interface{}
	}{project, episodes}
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

// 占位 handler（后续任务实现）
func (srv *Server) handleCreateEpisode(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
func (srv *Server) handleTriggerPipeline(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
func (srv *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
func (srv *Server) handleServeOutput(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix("/output/", http.FileServer(http.Dir(srv.outputDir))).ServeHTTP(w, r)
}
