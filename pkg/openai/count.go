package openai

import (
	"encoding/json"

	openai "github.com/gptscript-ai/chat-completion-client"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

func init() {
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
}

const DefaultMaxTokens = 400_000 // This is the limit for GPT-5

func decreaseTenPercent(maxTokens int) int {
	maxTokens = getBudget(maxTokens)
	return int(float64(maxTokens) * 0.9)
}

func getBudget(maxTokens int) int {
	if maxTokens <= 0 {
		return DefaultMaxTokens
	}
	return maxTokens
}

func dropMessagesOverCount(maxTokens, toolTokenCount int, msgs []openai.ChatCompletionMessage) (result []openai.ChatCompletionMessage, err error) {
	var (
		lastSystem   int
		withinBudget int
		budget       = getBudget(maxTokens) - toolTokenCount
	)

	for i, msg := range msgs {
		if msg.Role == openai.ChatMessageRoleSystem {
			count, err := countMessage(msg)
			if err != nil {
				return nil, err
			}
			budget -= count
			lastSystem = i
			result = append(result, msg)
		} else {
			break
		}
	}

	for i := len(msgs) - 1; i > lastSystem; i-- {
		withinBudget = i
		count, err := countMessage(msgs[i])
		if err != nil {
			return nil, err
		}
		budget -= count
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
		return msgs, nil
	}

	return append(result, msgs[withinBudget:]...), nil
}

func countMessage(msg openai.ChatCompletionMessage) (int, error) {
	encoding, err := tiktoken.GetEncoding("o200k_base")
	if err != nil {
		return 0, err
	}

	count := len(encoding.Encode(msg.Role, nil, nil))
	count += len(encoding.Encode(msg.Content, nil, nil))
	for _, content := range msg.MultiContent {
		count += len(encoding.Encode(content.Text, nil, nil))
	}
	for _, tool := range msg.ToolCalls {
		count += len(encoding.Encode(tool.Function.Name, nil, nil))
		count += len(encoding.Encode(tool.Function.Arguments, nil, nil))
	}
	count += len(encoding.Encode(msg.ToolCallID, nil, nil))

	return count, nil
}

func countTools(tools []types.ChatCompletionTool) (int, error) {
	encoding, err := tiktoken.GetEncoding("o200k_base")
	if err != nil {
		return 0, err
	}

	toolJSON, err := json.Marshal(tools)
	if err != nil {
		return 0, err
	}

	return len(encoding.Encode(string(toolJSON), nil, nil)), nil
}
