// Package agent provides trpc-agent-go based AI agents for novel analysis
// and screenplay generation.
package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/runner"
)

// Run executes a single-turn agent run using the framework Runner.
// It creates a Runner per call, which automatically handles OTel spans and logging.
func Run(ctx context.Context, agt *llmagent.LLMAgent, prompt string) (string, error) {
	r := runner.NewRunner("fabula-studio", agt)
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	msg := model.NewUserMessage(prompt)
	eventChan, err := r.Run(ctx, "default-user", sessionID, msg)
	if err != nil {
		return "", fmt.Errorf("agent run failed: %w", err)
	}

	var sb strings.Builder
	for evt := range eventChan {
		if evt.Error != nil {
			return "", fmt.Errorf("agent error: %s", evt.Error.Message)
		}
		for _, choice := range evt.Response.Choices {
			// Collect from Message.Content (non-streaming) or Delta.Content (streaming chunks)
			content := choice.Message.Content
			if content == "" {
				content = choice.Delta.Content
			}
			if content != "" {
				sb.WriteString(content)
			}
		}
	}
	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("agent returned empty response")
	}
	return result, nil

}
