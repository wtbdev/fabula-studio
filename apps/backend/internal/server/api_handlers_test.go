package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fabula-studio/backend/internal/config"
	"github.com/google/uuid"
)

type testAPI struct {
	t   *testing.T
	srv *Server
}

func newTestAPI(t *testing.T) *testAPI {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for backend API integration tests")
	}
	srv := New(config.Config{Addr: ":0", ModelName: "test", DatabaseURL: dsn, JWTSecret: "test-secret", JWTTTL: time.Hour})
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	_, err := srv.store.Pool.Exec(context.Background(), "TRUNCATE ai_logs, scenes, projects, users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate test database: %v", err)
	}
	return &testAPI{t: t, srv: srv}
}

func (a *testAPI) request(method, path, token, body string) (int, map[string]any) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	a.srv.http.Handler.ServeHTTP(rr, req)
	var payload map[string]any
	if err := json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&payload); err != nil {
		a.t.Fatalf("decode response %s %s status=%d body=%q: %v", method, path, rr.Code, rr.Body.String(), err)
	}
	return rr.Code, payload
}

func requireCode(t *testing.T, payload map[string]any, want float64) {
	t.Helper()
	if payload["code"] != want {
		t.Fatalf("code=%v want=%v payload=%v", payload["code"], want, payload)
	}
}

func registerUser(t *testing.T, api *testAPI, email string) string {
	_, payload := api.request(http.MethodPost, "/api/auth/register", "", `{"email":"`+email+`","password":"123456","nickname":"Tester"}`)
	requireCode(t, payload, 0)
	data := payload["data"].(map[string]any)
	return data["token"].(string)
}

func createProject(t *testing.T, api *testAPI, token, title string) string {
	body := `{"title":"` + title + `","novelTitle":"Novel","sourceText":"第一章 很长的小说文本。第二章 很长的小说文本。第三章 很长的小说文本。","config":{"style":"影视剧","dialogueLevel":"适中","adaptationMode":"忠实原文"}}`
	_, payload := api.request(http.MethodPost, "/api/projects", token, body)
	requireCode(t, payload, 0)
	data := payload["data"].(map[string]any)
	return data["id"].(string)
}

func TestAuthRegisterLoginMeLogoutAndDuplicate(t *testing.T) {
	api := newTestAPI(t)
	token := registerUser(t, api, "user@example.com")
	_, duplicate := api.request(http.MethodPost, "/api/auth/register", "", `{"email":"user@example.com","password":"123456","nickname":"Tester"}`)
	requireCode(t, duplicate, 40003)
	_, login := api.request(http.MethodPost, "/api/auth/login", "", `{"email":"user@example.com","password":"123456"}`)
	requireCode(t, login, 0)
	_, me := api.request(http.MethodGet, "/api/auth/me", token, "")
	requireCode(t, me, 0)
	if me["data"].(map[string]any)["email"] != "user@example.com" {
		t.Fatalf("unexpected me payload: %v", me)
	}
	_, logout := api.request(http.MethodPost, "/api/auth/logout", token, "")
	requireCode(t, logout, 0)
}

func TestProtectedEndpointWithoutToken(t *testing.T) {
	api := newTestAPI(t)
	status, payload := api.request(http.MethodGet, "/api/projects", "", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", status, http.StatusUnauthorized)
	}
	requireCode(t, payload, 40001)
}

func TestProjectCRUDAndOwnership(t *testing.T) {
	api := newTestAPI(t)
	owner := registerUser(t, api, "owner@example.com")
	other := registerUser(t, api, "other@example.com")
	projectID := createProject(t, api, owner, "Project A")
	_, list := api.request(http.MethodGet, "/api/projects?page=1&pageSize=10", owner, "")
	requireCode(t, list, 0)
	items := list["data"].(map[string]any)["list"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["id"] != projectID {
		t.Fatalf("unexpected list: %v", list)
	}
	_, detail := api.request(http.MethodGet, "/api/projects/"+projectID, owner, "")
	requireCode(t, detail, 0)
	_, denied := api.request(http.MethodGet, "/api/projects/"+projectID, other, "")
	requireCode(t, denied, 40401)
	_, updated := api.request(http.MethodPatch, "/api/projects/"+projectID, owner, `{"title":"Project B","novelTitle":"Novel B"}`)
	requireCode(t, updated, 0)
	_, deleted := api.request(http.MethodDelete, "/api/projects/"+projectID, owner, "")
	requireCode(t, deleted, 0)
	_, missing := api.request(http.MethodGet, "/api/projects/"+projectID, owner, "")
	requireCode(t, missing, 40401)
}

func TestScenesSortedUpdateValidationAndOwnership(t *testing.T) {
	api := newTestAPI(t)
	owner := registerUser(t, api, "scene-owner@example.com")
	other := registerUser(t, api, "scene-other@example.com")
	projectID := createProject(t, api, owner, "Scene Project")
	_, err := api.srv.store.Pool.Exec(context.Background(), `INSERT INTO scenes (id, project_id, scene_no, title, content, created_at, updated_at) VALUES ($1,$2,2,'Second','second',now(),now()), ($3,$2,1,'First','first',now(),now())`, "scene-2", projectID, "scene-1")
	if err != nil {
		t.Fatalf("insert scenes: %v", err)
	}
	_, list := api.request(http.MethodGet, "/api/projects/"+projectID+"/scenes", owner, "")
	requireCode(t, list, 0)
	items := list["data"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["sceneNo"] != float64(1) || items[1].(map[string]any)["sceneNo"] != float64(2) {
		t.Fatalf("scenes not sorted: %v", list)
	}
	_, denied := api.request(http.MethodGet, "/api/scenes/scene-1", other, "")
	requireCode(t, denied, 40402)
	_, invalid := api.request(http.MethodPatch, "/api/scenes/scene-1", owner, `{"content":""}`)
	requireCode(t, invalid, 40002)
	_, saved := api.request(http.MethodPatch, "/api/scenes/scene-1", owner, `{"title":"First Updated","content":"updated content"}`)
	requireCode(t, saved, 0)
	_, detail := api.request(http.MethodGet, "/api/scenes/scene-1", owner, "")
	requireCode(t, detail, 0)
	if detail["data"].(map[string]any)["content"] != "updated content" {
		t.Fatalf("scene not updated: %v", detail)
	}
}

func TestGenerationValidationErrors(t *testing.T) {
	api := newTestAPI(t)
	token := registerUser(t, api, "generate@example.com")
	userID, err := api.srv.parseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	emptyProject := uuid.NewString()
	_, err = api.srv.store.Pool.Exec(context.Background(), `INSERT INTO projects (id, user_id, title, source_text, config_json, status, created_at, updated_at) VALUES ($1,$2,'Empty','', '{}', 'draft', now(), now())`, emptyProject, userID)
	if err != nil {
		t.Fatalf("insert empty project: %v", err)
	}
	_, noSource := api.request(http.MethodPost, "/api/projects/"+emptyProject+"/generate", token, `{}`)
	requireCode(t, noSource, 41002)
	lowPointsProject := createProject(t, api, token, "Low Points")
	_, err = api.srv.store.Pool.Exec(context.Background(), `UPDATE users SET ai_points = 100 WHERE id = $1`, userID)
	if err != nil {
		t.Fatalf("lower points: %v", err)
	}
	_, noPoints := api.request(http.MethodPost, "/api/projects/"+lowPointsProject+"/generate", token, `{}`)
	requireCode(t, noPoints, 50001)
}
