package server

import (
	"github.com/fabula-studio/backend/internal/config"
	"github.com/fabula-studio/backend/internal/pipeline"
)

func pipelineConfigFromAppConfig(cfg config.Config) pipeline.Config {
	pipelineConfig := pipeline.DefaultConfig()
	pipelineConfig.MaxConcurrency = cfg.PipelineMaxConcurrency
	return pipelineConfig
}
