package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/fabula-studio/backend/internal/db/sqlc"
	"github.com/fabula-studio/backend/internal/schema"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const generationCost int32 = 300

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	parts := splitAPIPath(r.URL.Path)
	if len(parts) != 3 || parts[0] != "api" || parts[1] != "auth" {
		http.NotFound(w, r)
		return
	}
	switch parts[2] {
	case "register":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleRegister(w, r)
	case "login":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleLogin(w, r)
	case "me":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.handleMe(w, r)
	case "logout":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if _, ok := s.requireUserID(w, r); !ok {
			return
		}
		writeAPISuccess(w, "退出登录成功", true)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Email) == "" || len(req.Password) < 6 || strings.TrimSpace(req.Nickname) == "" {
		writeAPIError(w, http.StatusBadRequest, codeInvalid, "参数校验失败")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, codeInternal, "服务器内部错误")
		return
	}
	u, err := s.store.Users.Create(r.Context(), uuid.NewString(), req.Email, string(hash), req.Nickname)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeAPIError(w, http.StatusBadRequest, codeEmailExists, "邮箱已被注册")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, codeDB, "数据库操作失败")
		return
	}
	token, err := s.makeToken(u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, codeInternal, "服务器内部错误")
		return
	}
	writeAPISuccess(w, "注册成功", authTokenDTO{Token: token, User: userToDTO(u)})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Email) == "" || req.Password == "" {
		writeAPIError(w, http.StatusBadRequest, codeInvalid, "参数校验失败")
		return
	}
	u, err := s.store.Users.ByEmail(r.Context(), req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		writeAPIError(w, http.StatusBadRequest, codeBadLogin, "邮箱或密码错误")
		return
	}
	token, err := s.makeToken(u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, codeInternal, "服务器内部错误")
		return
	}
	writeAPISuccess(w, "登录成功", authTokenDTO{Token: token, User: userToDTO(u)})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	u, err := s.store.Users.ByID(r.Context(), userID)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, codeUnauth, "未登录或登录已过期")
		return
	}
	writeAPISuccess(w, "success", userToDTO(u))
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	parts := splitAPIPath(r.URL.Path)
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "projects" {
		http.NotFound(w, r)
		return
	}
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateProject(w, r, userID)
		case http.MethodGet:
			s.handleListProjects(w, r, userID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodGet:
			s.handleGetProject(w, r, userID, parts[2])
		case http.MethodPatch:
			s.handleUpdateProject(w, r, userID, parts[2])
		case http.MethodDelete:
			s.handleDeleteProject(w, r, userID, parts[2])
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 4 && parts[3] == "generate" && r.Method == http.MethodPost {
		s.handleGenerate(w, r, userID, parts[2])
		return
	}
	if len(parts) == 5 && parts[3] == "generate" && parts[4] == "status" && r.Method == http.MethodGet {
		s.handleGenerateStatus(w, r, userID, parts[2])
		return
	}
	if len(parts) == 4 && parts[3] == "scenes" && r.Method == http.MethodGet {
		s.handleProjectScenes(w, r, userID, parts[2])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request, userID string) {
	var req createProjectRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.SourceText) == "" || req.Config.Style == "" || req.Config.DialogueLevel == "" || req.Config.AdaptationMode == "" {
		writeAPIError(w, http.StatusBadRequest, codeInvalid, "参数校验失败")
		return
	}
	if len([]rune(strings.TrimSpace(req.SourceText))) < 20 {
		writeAPIError(w, http.StatusBadRequest, codeTextShort, "小说文本过短")
		return
	}
	cfg, err := json.Marshal(req.Config)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, codeInvalid, "参数校验失败")
		return
	}
	p, err := s.store.Projects.Create(r.Context(), sqlc.CreateProjectParams{ID: uuid.NewString(), UserID: userID, Title: req.Title, NovelTitle: textValue(req.NovelTitle), SourceText: req.SourceText, ConfigJson: string(cfg)})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, codeDB, "数据库操作失败")
		return
	}
	writeAPISuccess(w, "项目创建成功", projectToDTO(p, false, nil))
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request, userID string) {
	page := parseInt32(r.URL.Query().Get("page"), 1)
	pageSize := parseInt32(r.URL.Query().Get("pageSize"), 10)
	result, err := s.store.Projects.List(r.Context(), userID, r.URL.Query().Get("keyword"), page, pageSize)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, codeDB, "数据库操作失败")
		return
	}
	items := make([]projectDTO, 0, len(result.Items))
	for _, p := range result.Items {
		items = append(items, projectRowToDTO(p))
	}
	writeAPISuccess(w, "success", pageResult[projectDTO]{List: items, Total: result.Total, Page: page, PageSize: pageSize})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	p, err := s.store.Projects.ByIDForUser(r.Context(), projectID, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, codeProjectMiss, "项目不存在")
		return
	}
	writeAPISuccess(w, "success", projectToDTO(p, true, nil))
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	var req updateProjectRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Title) == "" {
		writeAPIError(w, http.StatusBadRequest, codeInvalid, "参数校验失败")
		return
	}
	p, err := s.store.Projects.UpdateInfo(r.Context(), projectID, userID, req.Title, textValue(req.NovelTitle))
	if err != nil {
		writeAPIError(w, http.StatusNotFound, codeProjectMiss, "项目不存在")
		return
	}
	writeAPISuccess(w, "项目更新成功", projectToDTO(p, false, nil))
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	ok, err := s.store.Projects.Delete(r.Context(), projectID, userID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, codeDB, "数据库操作失败")
		return
	}
	if !ok {
		writeAPIError(w, http.StatusNotFound, codeProjectMiss, "项目不存在")
		return
	}
	writeAPISuccess(w, "项目删除成功", true)
}

func (s *Server) handleProjectScenes(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	if _, err := s.store.Projects.ByIDForUser(r.Context(), projectID, userID); err != nil {
		writeAPIError(w, http.StatusNotFound, codeProjectMiss, "项目不存在")
		return
	}
	scenes, err := s.store.Scenes.ListForProject(r.Context(), projectID, userID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, codeDB, "数据库操作失败")
		return
	}
	items := make([]sceneDTO, 0, len(scenes))
	for _, sc := range scenes {
		items = append(items, sceneToDTO(sc, false))
	}
	writeAPISuccess(w, "success", items)
}

func (s *Server) handleScenes(w http.ResponseWriter, r *http.Request) {
	parts := splitAPIPath(r.URL.Path)
	if len(parts) != 3 || parts[0] != "api" || parts[1] != "scenes" {
		http.NotFound(w, r)
		return
	}
	userID, ok := s.requireUserID(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		sc, err := s.store.Scenes.ByIDForUser(r.Context(), parts[2], userID)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, codeSceneMiss, "场次不存在")
			return
		}
		writeAPISuccess(w, "success", sceneToDTO(sc, true))
	case http.MethodPatch:
		s.handleUpdateScene(w, r, userID, parts[2])
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUpdateScene(w http.ResponseWriter, r *http.Request, userID, sceneID string) {
	current, err := s.store.Scenes.ByIDForUser(r.Context(), sceneID, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, codeSceneMiss, "场次不存在")
		return
	}
	var req updateSceneRequest
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Content) == "" {
		writeAPIError(w, http.StatusBadRequest, codeInvalid, "参数校验失败")
		return
	}
	title := current.Title
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
	}
	if title == "" {
		writeAPIError(w, http.StatusBadRequest, codeInvalid, "参数校验失败")
		return
	}
	loc, timeText, summary := current.Location, current.TimeText, current.Summary
	if req.Location != nil {
		loc = textValue(strings.TrimSpace(*req.Location))
	}
	if req.TimeText != nil {
		timeText = textValue(strings.TrimSpace(*req.TimeText))
	}
	if req.Summary != nil {
		summary = textValue(strings.TrimSpace(*req.Summary))
	}
	updated, err := s.store.Scenes.UpdateForUser(r.Context(), sqlc.UpdateSceneForUserParams{ID: sceneID, UserID: userID, Title: title, Location: loc, TimeText: timeText, Summary: summary, Content: req.Content})
	if err != nil {
		writeAPIError(w, http.StatusNotFound, codeSceneMiss, "场次不存在")
		return
	}
	writeAPISuccess(w, "场次保存成功", map[string]string{"id": updated.ID, "updatedAt": timestamp(updated.UpdatedAt)})
}

func (s *Server) handleGenerateStatus(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	p, err := s.store.Projects.ByIDForUser(r.Context(), projectID, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, codeProjectMiss, "项目不存在")
		return
	}
	progress := 0
	step := "未开始"
	if p.Status == "generating" {
		status := s.pipeline.Status()
		progress = status.Progress
		step = status.CurrentStep
	}
	if p.Status == "completed" {
		progress = 100
		step = "已完成"
	}
	if p.Status == "failed" {
		step = "生成失败"
	}
	writeAPISuccess(w, "success", generationStatusDTO{ProjectID: projectID, Status: p.Status, Progress: progress, CurrentStep: step, Artifacts: generationArtifactsFromPipeline(s.pipeline.Result())})
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request, userID, projectID string) {
	p, err := s.store.Projects.ByIDForUser(r.Context(), projectID, userID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, codeProjectMiss, "项目不存在")
		return
	}
	if strings.TrimSpace(p.SourceText) == "" {
		writeAPIError(w, http.StatusBadRequest, codeNoSource, "项目缺少小说文本")
		return
	}
	u, err := s.store.Users.ByID(r.Context(), userID)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, codeUnauth, "未登录或登录已过期")
		return
	}
	if u.AiPoints < generationCost {
		writeAPIError(w, http.StatusBadRequest, codeNoPoints, "AI 点数不足")
		return
	}
	_, _ = s.store.Queries.UpdateProjectStatus(r.Context(), sqlc.UpdateProjectStatusParams{ID: projectID, UserID: userID, Status: "generating"})
	profile := adaptationProfileFromConfigJSON(p.ConfigJson)
	ctx := schema.WithAdaptationProfile(r.Context(), &profile)
	sp, err := s.pipeline.Convert(ctx, p.Title, u.Nickname, splitChapters(p.SourceText))
	if err != nil {
		s.markGenerationFailed(r.Context(), projectID, userID, err.Error())
		writeAPIError(w, http.StatusInternalServerError, codeAIFailed, "AI 生成失败")
		return
	}
	if sp == nil || len(sp.Scenes) == 0 {
		s.markGenerationFailed(r.Context(), projectID, userID, "empty generation result")
		writeAPIError(w, http.StatusInternalServerError, codeAIEmpty, "AI 生成结果为空")
		return
	}
	var saved []sqlc.Scene
	var remaining int32
	err = s.store.WithTx(r.Context(), func(q *sqlc.Queries) error {
		updatedUser, err := q.DecrementUserAIPoints(r.Context(), sqlc.DecrementUserAIPointsParams{ID: userID, AiPoints: generationCost})
		if err != nil {
			return err
		}
		remaining = updatedUser.AiPoints
		if err := q.DeleteScenesByProjectID(r.Context(), projectID); err != nil {
			return err
		}
		for _, scene := range sp.Scenes {
			created, err := q.CreateScene(r.Context(), screenplaySceneToCreate(projectID, scene))
			if err != nil {
				return err
			}
			saved = append(saved, created)
		}
		_, err = q.UpdateProjectStatus(r.Context(), sqlc.UpdateProjectStatusParams{ID: projectID, UserID: userID, Status: "completed"})
		return err
	})
	if err != nil {
		s.markGenerationFailed(r.Context(), projectID, userID, err.Error())
		writeAPIError(w, http.StatusInternalServerError, codeDB, "数据库操作失败")
		return
	}
	items := make([]sceneDTO, 0, len(saved))
	for _, sc := range saved {
		items = append(items, sceneToDTO(sc, true))
	}
	writeAPISuccess(w, "剧本生成成功", generationResponse{ProjectID: projectID, Status: "completed", CostPoints: generationCost, RemainingPoints: remaining, Scenes: items, Artifacts: generationArtifactsFromPipeline(s.pipeline.Result())})
}

func (s *Server) markGenerationFailed(ctx context.Context, projectID, userID, msg string) {
	_, _ = s.store.Queries.UpdateProjectStatus(ctx, sqlc.UpdateProjectStatusParams{ID: projectID, UserID: userID, Status: "failed", ErrorMessage: textValue(msg)})
}

func screenplaySceneToCreate(projectID string, scene schema.Scene) sqlc.CreateSceneParams {
	raw, _ := json.Marshal(sceneRawJSON{Characters: scene.CharactersPresent, Script: screenplayBlocks(scene.Content)})
	return sqlc.CreateSceneParams{ID: uuid.NewString(), ProjectID: projectID, SceneNo: int32(scene.Sequence), Title: scene.Heading, Location: textValue(scene.Setting.Location), TimeText: textValue(scene.Setting.Time), Summary: textValue(scene.Synopsis), Content: sceneContent(scene), RawJson: textValue(string(raw))}
}

func screenplayBlocks(elements []schema.SceneElement) []scriptBlock {
	blocks := make([]scriptBlock, 0, len(elements))
	for _, e := range elements {
		t := string(e.Type)
		if t == "shot" || t == "parenthetical" {
			t = "action"
		}
		blocks = append(blocks, scriptBlock{Type: t, Character: e.Character, Content: e.Text})
	}
	return blocks
}

func sceneContent(scene schema.Scene) string {
	var b strings.Builder
	b.WriteString("【第 ")
	b.WriteString(strconv.Itoa(scene.Sequence))
	b.WriteString(" 场】")
	b.WriteString(scene.Heading)
	b.WriteString("\n\n地点：")
	b.WriteString(scene.Setting.Location)
	b.WriteString("\n时间：")
	b.WriteString(scene.Setting.Time)
	b.WriteString("\n")
	for _, e := range scene.Content {
		b.WriteString("\n")
		switch e.Type {
		case schema.ElementDialogue:
			b.WriteString(e.Character)
			b.WriteString("：")
		case schema.ElementTransition:
			b.WriteString("转场：")
		default:
			b.WriteString("动作：")
		}
		b.WriteString(e.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func splitChapters(source string) []string {
	parts := strings.FieldsFunc(source, func(r rune) bool { return r == '\f' })
	if len(parts) == 0 {
		return []string{source}
	}
	chapters := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			chapters = append(chapters, t)
		}
	}
	if len(chapters) == 0 {
		return []string{source}
	}
	return chapters
}

func parseInt32(s string, fallback int32) int32 {
	if s == "" {
		return fallback
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil || n < 1 {
		return fallback
	}
	return int32(n)
}

func isNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
