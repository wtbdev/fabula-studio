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
		if len(evt.Response.Choices) > 0 {
			content := evt.Response.Choices[0].Message.Content
			sb.WriteString(content)
		}
	}
	return sb.String(), nil
}
