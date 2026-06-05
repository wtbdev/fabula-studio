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

// RunAgent executes a single-turn agent run and returns the text response.
func RunAgent(ctx context.Context, agt *llmagent.LLMAgent, prompt string) (string, error) {
	r := runner.NewRunner("fabula-studio", agt)
	msg := model.NewUserMessage(prompt)
	eventChan, err := r.Run(ctx, "default-user", fmt.Sprintf("session-%d", time.Now().UnixNano()), msg)
	if err != nil {
		return "", fmt.Errorf("agent run failed: %w", err)
	}

	var sb strings.Builder
	for evt := range eventChan {
		if evt.Error != nil {
			return "", fmt.Errorf("agent error: %s", evt.Error.Message)
		}
		if len(evt.Response.Choices) > 0 {
			sb.WriteString(evt.Response.Choices[0].Message.Content)
		}
	}
	return sb.String(), nil
}
