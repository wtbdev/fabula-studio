package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/fabula-studio/backend/internal/db/sqlc"
	"github.com/fabula-studio/backend/internal/pipeline"
	"github.com/fabula-studio/backend/internal/schema"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	codeSuccess       = 0
	codeUnauth        = 40001
	codeInvalid       = 40002
	codeEmailExists   = 40003
	codeBadLogin      = 40004
	codeProjectMiss   = 40401
	codeSceneMiss     = 40402
	codeTextShort     = 41001
	codeNoSource      = 41002
	codeBadStatus     = 41003
	codeNoPoints      = 50001
	codeInternal      = 50000
	codeDB            = 50002
	codeAIFailed      = 51001
	codeAIParseFailed = 51002
	codeAIEmpty       = 51003
)

func writeAPISuccess[T any](w http.ResponseWriter, message string, data T) {
	writeJSON(w, http.StatusOK, apiResponse[T]{Code: codeSuccess, Message: message, Data: data})
}

func writeAPIError(w http.ResponseWriter, status, code int, message string) {
	writeJSON(w, status, apiResponse[any]{Code: code, Message: message, Data: nil})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func textValue(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func textString(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func timestamp(v pgtype.Timestamptz) string {
	if !v.Valid {
		return ""
	}
	return v.Time.UTC().Format(time.RFC3339Nano)
}

func userToDTO(u sqlc.User) userDTO {
	return userDTO{ID: u.ID, Email: u.Email, Nickname: u.Nickname, AIPoints: u.AiPoints, CreatedAt: timestamp(u.CreatedAt), UpdatedAt: timestamp(u.UpdatedAt)}
}

func projectToDTO(p sqlc.Project, includeSource bool, sceneCount *int32) projectDTO {
	cfg := parseAdaptConfig(p.ConfigJson)
	profile := adaptationProfileFromConfig(cfg)
	dto := projectDTO{ID: p.ID, UserID: p.UserID, Title: p.Title, NovelTitle: textPtr(p.NovelTitle), Config: cfg, AdaptationProfile: &profile, Status: p.Status, ErrorMessage: textPtr(p.ErrorMessage), SceneCount: sceneCount, CreatedAt: timestamp(p.CreatedAt), UpdatedAt: timestamp(p.UpdatedAt)}
	if includeSource {
		dto.SourceText = &p.SourceText
	}
	return dto
}

func projectRowToDTO(p sqlc.ListProjectsRow) projectDTO {
	count := p.SceneCount
	cfg := parseAdaptConfig(p.ConfigJson)
	profile := adaptationProfileFromConfig(cfg)
	return projectDTO{ID: p.ID, Title: p.Title, NovelTitle: textPtr(p.NovelTitle), Config: cfg, AdaptationProfile: &profile, Status: p.Status, ErrorMessage: textPtr(p.ErrorMessage), SceneCount: &count, CreatedAt: timestamp(p.CreatedAt), UpdatedAt: timestamp(p.UpdatedAt)}
}

func parseAdaptConfig(raw string) adaptConfig {
	var cfg adaptConfig
	_ = json.Unmarshal([]byte(raw), &cfg)
	return cfg
}

func adaptationProfileFromConfig(cfg adaptConfig) schema.AdaptationProfile {
	return schema.AdaptationProfile{Style: strings.TrimSpace(cfg.Style), DialogueLevel: strings.TrimSpace(cfg.DialogueLevel), AdaptationMode: strings.TrimSpace(cfg.AdaptationMode), SceneGranularity: strings.TrimSpace(cfg.SceneGranularity), NarrationLevel: strings.TrimSpace(cfg.NarrationLevel), CustomGuidance: strings.TrimSpace(cfg.CustomPrompt)}
}

func adaptationProfileFromConfigJSON(raw string) schema.AdaptationProfile {
	return adaptationProfileFromConfig(parseAdaptConfig(raw))
}

func generationArtifactsFromPipeline(result *pipeline.PipelineResult) *schema.GenerationArtifacts {
	if result == nil {
		return nil
	}
	return result.Artifacts
}

func sceneToDTO(s sqlc.Scene, includeRaw bool) sceneDTO {
	dto := sceneDTO{ID: s.ID, ProjectID: s.ProjectID, SceneNo: s.SceneNo, Title: s.Title, Location: textString(s.Location), TimeText: textString(s.TimeText), Summary: textString(s.Summary), Content: s.Content, CreatedAt: timestamp(s.CreatedAt), UpdatedAt: timestamp(s.UpdatedAt)}
	if includeRaw && s.RawJson.Valid && s.RawJson.String != "" {
		raw := json.RawMessage(s.RawJson.String)
		dto.RawJSON = &raw
	}
	return dto
}

func splitAPIPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

type claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

func (s *Server) makeToken(userID string) (string, error) {
	c := claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWTTTL)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(s.config.JWTSecret))
}

func (s *Server) parseToken(token string) (string, error) {
	if token == "" {
		return "", errors.New("missing token")
	}
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) { return []byte(s.config.JWTSecret), nil })
	if err != nil || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || c.UserID == "" {
		return "", errors.New("invalid claims")
	}
	return c.UserID, nil
}

func (s *Server) requireUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := s.parseToken(bearerToken(r))
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, codeUnauth, "未登录或登录已过期")
		return "", false
	}
	return id, true
}
