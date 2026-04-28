package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kiddyt00/comic-pipeline/pkg/n8n"
)

func TestTriggerEpisode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/webhook/episode-pipeline" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content-type: %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := n8n.New(server.URL)
	err := client.TriggerEpisode(context.Background(), n8n.TriggerPayload{
		ProjectID:    "prj_1",
		EpisodeID:    "ep_1",
		EpisodeNum:   1,
		WorldSetting: "修仙世界",
		StoryText:    "第一章...",
		CallbackURL:  "http://localhost:8080/internal/callback",
	})
	if err != nil {
		t.Fatalf("TriggerEpisode: %v", err)
	}
}
