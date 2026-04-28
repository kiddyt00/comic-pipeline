package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kiddyt00/comic-pipeline/pkg/n8n"
	"github.com/kiddyt00/comic-pipeline/pkg/store"
)

type Server struct {
	store     *store.Store
	n8nClient *n8n.Client
	outputDir string
}

func Serve(ctx context.Context, dbPath, n8nBaseURL, outputDir string, port int) error {
	s, err := store.New(dbPath)
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}
	defer s.Close()

	srv := &Server{
		store:     s,
		n8nClient: n8n.New(n8nBaseURL),
		outputDir: outputDir,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.handleIndex)
	mux.HandleFunc("POST /projects", srv.handleCreateProject)
	mux.HandleFunc("GET /projects", srv.handleListProjects)
	mux.HandleFunc("GET /projects/{id}", srv.handleProjectDetail)
	mux.HandleFunc("DELETE /projects/{id}", srv.handleDeleteProject)
	mux.HandleFunc("POST /projects/{id}/episodes", srv.handleCreateEpisode)
	mux.HandleFunc("POST /projects/{id}/episodes/{epId}/run", srv.handleTriggerPipeline)
	mux.HandleFunc("GET /projects/{id}/episodes/{epId}/scenes", srv.handleScenesFragment)
	mux.HandleFunc("POST /internal/callback", srv.handleCallback)
	mux.HandleFunc("GET /output/", srv.handleServeOutput)

	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	fmt.Printf("🎬 comic-pipeline 面板: http://localhost%s\n", addr)
	return httpServer.ListenAndServe()
}
