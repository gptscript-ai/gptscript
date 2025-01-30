package openai

import (
	"encoding/json"

	openai "github.com/gptscript-ai/chat-completion-client"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

const DefaultMaxTokens = 128_000

func decreaseTenPercent(maxTokens int) int {
	maxTokens = getBudget(maxTokens)
	return int(float64(maxTokens) * 0.9)
}

func getBudget(maxTokens int) int {
	if maxTokens == 0 {
		return DefaultMaxTokens
	}
	return maxTokens
}

func dropMessagesOverCount(maxTokens int, msgs []openai.ChatCompletionMessage) (result []openai.ChatCompletionMessage) {
	var (
		lastSystem   int
		withinBudget int
		budget       = getBudget(maxTokens)
	)

	for i, msg := range msgs {
		if msg.Role == openai.ChatMessageRoleSystem {
			budget -= countMessage(msg)
			lastSystem = i
			result = append(result, msg)
		} else {
			break
		}
	}

	for i := len(msgs) - 1; i > lastSystem; i-- {
		withinBudget = i
		budget -= countMessage(msgs[i])
		if budget <= 0 {
			break
		}
	}

	// OpenAI gets upset if there is a tool message without a tool call preceding it.
	// Check the oldest message within budget, and if it is a tool message, just drop it.
	// We do this in a loop because it is possible for multiple tool messages to be in a row,
	// due to parallel tool calls.
	for withinBudget < len(msgs) && msgs[withinBudget].Role == openai.ChatMessageRoleTool {
		withinBudget++
	}

	if withinBudget == len(msgs)-1 {
		// We are going to drop all non system messages, which seems useless, so just return them
		// all and let it fail
		return msgs
	}

	return append(result, msgs[withinBudget:]...)
}

func countMessage(msg openai.ChatCompletionMessage) (count int) {
	count += len(msg.Role)
	count += len(msg.Content)
	for _, content := range msg.MultiContent {
		count += len(content.Text)
	}
	for _, tool := range msg.ToolCalls {
		count += len(tool.Function.Name)
		count += len(tool.Function.Arguments)
	}
	count += len(msg.ToolCallID)
	return count / 3
}

func countChatCompletionTools(tools []types.ChatCompletionTool) (count int, err error) {
	for _, t := range tools {
		count += len(t.Function.Name)
		count += len(t.Function.Description)
		paramsJSON, err := json.Marshal(t.Function.Parameters)
		if err != nil {
			return 0, err
		}
		count += len(paramsJSON)
	}
	return count / 3, nil
}

func countOpenAITools(tools []openai.Tool) (count int, err error) {
	for _, t := range tools {
		count += len(t.Function.Name)
		count += len(t.Function.Description)
		paramsJSON, err := json.Marshal(t.Function.Parameters)
		if err != nil {
			return 0, err
		}
		count += len(paramsJSON)
	}
	return count / 3, nil
}
