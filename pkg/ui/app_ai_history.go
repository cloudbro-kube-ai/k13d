package ui

import (
	"fmt"
	"strings"
)

const maxAIConversationContextMessages = 20

type aiConversationMessage struct {
	Role    string
	Content string
}

func (a *App) addAIConversationMessage(role, content string) {
	trimmedRole := strings.TrimSpace(strings.ToLower(role))
	trimmedContent := strings.TrimSpace(content)
	if trimmedRole == "" || trimmedContent == "" {
		return
	}

	a.aiMx.Lock()
	defer a.aiMx.Unlock()

	a.aiConversationHistory = append(a.aiConversationHistory, aiConversationMessage{
		Role:    trimmedRole,
		Content: trimmedContent,
	})
	if len(a.aiConversationHistory) > maxAIConversationContextMessages {
		a.aiConversationHistory = a.aiConversationHistory[len(a.aiConversationHistory)-maxAIConversationContextMessages:]
	}
}

func (a *App) buildAIConversationPrompt(basePrompt string) string {
	history := a.snapshotAIConversationHistory(maxAIConversationContextMessages)
	if len(history) == 0 {
		return basePrompt
	}

	var prompt strings.Builder
	prompt.WriteString("IMPORTANT: This is a continuation of an ongoing conversation inside the k13d terminal UI. You MUST maintain context from the previous messages below.\n\n")
	prompt.WriteString("=== CONVERSATION HISTORY ===\n")
	for i, msg := range history {
		switch msg.Role {
		case "user":
			prompt.WriteString(fmt.Sprintf("[%d] USER: %s\n", i+1, msg.Content))
		case "assistant":
			prompt.WriteString(fmt.Sprintf("[%d] ASSISTANT: %s\n", i+1, msg.Content))
		}
	}
	prompt.WriteString("=== END OF HISTORY ===\n\n")
	prompt.WriteString("Now answer the user's NEW question below while keeping the prior conversation context in mind.\n\n")
	prompt.WriteString(basePrompt)
	return prompt.String()
}

func (a *App) snapshotAIConversationHistory(limit int) []aiConversationMessage {
	a.aiMx.RLock()
	defer a.aiMx.RUnlock()

	if len(a.aiConversationHistory) == 0 {
		return nil
	}

	if limit <= 0 || len(a.aiConversationHistory) <= limit {
		history := make([]aiConversationMessage, len(a.aiConversationHistory))
		copy(history, a.aiConversationHistory)
		return history
	}

	start := len(a.aiConversationHistory) - limit
	history := make([]aiConversationMessage, len(a.aiConversationHistory[start:]))
	copy(history, a.aiConversationHistory[start:])
	return history
}
