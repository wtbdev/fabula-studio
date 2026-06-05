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
	evtCount := 0
	for evt := range eventChan {
		evtCount++
		if evt.Error != nil {
			fmt.Printf("[Agent] Error event: %s\n", evt.Error.Message)
			return "", fmt.Errorf("agent error: %s", evt.Error.Message)
		}
		if len(evt.Response.Choices) > 0 {
			content := evt.Response.Choices[0].Message.Content
			sb.WriteString(content)
		}
	}
	result := sb.String()
	if result == "" {
		fmt.Printf("[Agent] Warning: empty response after %d events\n", evtCount)
	}
	return result, nil
}
