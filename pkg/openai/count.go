package openai

import openai "github.com/gptscript-ai/chat-completion-client"

func dropMessagesOverCount(maxTokens int, msgs []openai.ChatCompletionMessage) (result []openai.ChatCompletionMessage) {
	var (
		lastSystem   int
		withinBudget int
		budget       = maxTokens
	)

	if maxTokens == 0 {
		budget = 300_000
	} else {
		budget *= 3
	}

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
