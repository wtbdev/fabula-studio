package server

import (
	"encoding/json"

	"github.com/fabula-studio/backend/internal/schema"
)

type apiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type pageResult[T any] struct {
	List     []T   `json:"list"`
	Total    int32 `json:"total"`
	Page     int32 `json:"page"`
	PageSize int32 `json:"pageSize"`
}

type userDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	AIPoints  int32  `json:"aiPoints"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type authTokenDTO struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

type adaptConfig struct {
	Style            string `json:"style"`
	DialogueLevel    string `json:"dialogueLevel"`
	AdaptationMode   string `json:"adaptationMode"`
	SceneGranularity string `json:"sceneGranularity,omitempty"`
	NarrationLevel   string `json:"narrationLevel,omitempty"`
	CustomPrompt     string `json:"customPrompt,omitempty"`
}

type projectDTO struct {
	ID                string                      `json:"id"`
	UserID            string                      `json:"userId,omitempty"`
	Title             string                      `json:"title"`
	NovelTitle        *string                     `json:"novelTitle,omitempty"`
	SourceText        *string                     `json:"sourceText,omitempty"`
	Config            adaptConfig                 `json:"config,omitempty"`
	AdaptationProfile *schema.AdaptationProfile   `json:"adaptationProfile,omitempty"`
	Artifacts         *schema.GenerationArtifacts `json:"artifacts,omitempty"`
	Status            string                      `json:"status"`
	ErrorMessage      *string                     `json:"errorMessage,omitempty"`
	SceneCount        *int32                      `json:"sceneCount,omitempty"`
	CreatedAt         string                      `json:"createdAt"`
	UpdatedAt         string                      `json:"updatedAt"`
}

type scriptBlock struct {
	Type      string `json:"type"`
	Character string `json:"character,omitempty"`
	Content   string `json:"content"`
}

type sceneRawJSON struct {
	Characters []string      `json:"characters,omitempty"`
	Script     []scriptBlock `json:"script,omitempty"`
	Source     any           `json:"source,omitempty"`
}

type sceneDTO struct {
	ID        string           `json:"id"`
	ProjectID string           `json:"projectId"`
	SceneNo   int32            `json:"sceneNo"`
	Title     string           `json:"title"`
	Location  string           `json:"location,omitempty"`
	TimeText  string           `json:"timeText,omitempty"`
	Summary   string           `json:"summary,omitempty"`
	Content   string           `json:"content"`
	RawJSON   *json.RawMessage `json:"rawJson,omitempty"`
	CreatedAt string           `json:"createdAt"`
	UpdatedAt string           `json:"updatedAt"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createProjectRequest struct {
	Title      string      `json:"title"`
	NovelTitle string      `json:"novelTitle"`
	SourceText string      `json:"sourceText"`
	Config     adaptConfig `json:"config"`
}

type updateProjectRequest struct {
	Title      string `json:"title"`
	NovelTitle string `json:"novelTitle"`
}

type updateSceneRequest struct {
	Title    *string `json:"title"`
	Location *string `json:"location"`
	TimeText *string `json:"timeText"`
	Summary  *string `json:"summary"`
	Content  string  `json:"content"`
}

type generationResponse struct {
	ProjectID       string                      `json:"projectId"`
	Status          string                      `json:"status"`
	CostPoints      int32                       `json:"costPoints"`
	RemainingPoints int32                       `json:"remainingPoints"`
	Scenes          []sceneDTO                  `json:"scenes"`
	Artifacts       *schema.GenerationArtifacts `json:"artifacts,omitempty"`
}

type generationStatusDTO struct {
	ProjectID   string                      `json:"projectId"`
	Status      string                      `json:"status"`
	Progress    int                         `json:"progress"`
	CurrentStep string                      `json:"currentStep"`
	Artifacts   *schema.GenerationArtifacts `json:"artifacts,omitempty"`
}
