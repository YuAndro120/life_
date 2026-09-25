package gigachat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"shtil/backend/internal/llm"
	"shtil/backend/internal/llm/tasks"
)

const lawFunctionName = "extract_law"

// ExtractLaw извлекает структуру закона. Ответ проверяется по тексту (цитаты должны в нём находиться);
// при несоответствии запрос повторяется, но ответ не «чинится».
func (c *Client) ExtractLaw(ctx context.Context, in llm.LawInput) (llm.LawDraft, llm.Usage, error) {
	if len(in.Fragments) == 0 {
		return llm.LawDraft{}, llm.Usage{}, errors.New("нет текста закона")
	}
	user, sent := tasks.LawUser(in)
	reqBody, err := json.Marshal(chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: tasks.LawSystem},
			{Role: "user", Content: user},
		},
		Functions:    []chatFunction{toFunction(tasks.LawSchema())},
		FunctionCall: map[string]any{"name": tasks.LawFunction},
		Temperature:  0.1,
		MaxTokens:    tasks.LawMaxTokens,
	})
	if err != nil {
		return llm.LawDraft{}, llm.Usage{}, err
	}
	var total llm.Usage
	var lastErr error
	for attempt := 1; attempt <= c.cfg.MaxAttempts; attempt++ {
		resp, err := c.chat(ctx, reqBody)
		if err != nil {
			return llm.LawDraft{}, total, err
		}
		total.PromptTokens += resp.Usage.PromptTokens
		total.CompletionTokens += resp.Usage.CompletionTokens

		if refusal(resp) {
			return llm.LawDraft{}, total, llm.ErrRefused
		}
		args, err := functionArgs(resp, tasks.LawFunction)
		if err == nil {
			var d llm.LawDraft
			if d, err = tasks.ParseLaw(args, sent); err == nil {
				if err = d.Validate(); err == nil {
					return d, total, nil
				}
			}
		}
		lastErr = err
		c.cfg.Log.Warn("модель вернула некорректный разбор закона", "attempt", attempt, "err", err, "ответ", rawAnswer(resp))
	}
	return llm.LawDraft{}, total, fmt.Errorf("%w: %v", llm.ErrInvalid, lastErr)
}

// rawAnswer — начало ответа модели для журнала (только для отладки, текст закона в него не входит).
func rawAnswer(resp chatResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	m := resp.Choices[0].Message
	s := m.Content
	if m.FunctionCall != nil {
		s = string(m.FunctionCall.Arguments)
	}
	if r := []rune(s); len(r) > 400 {
		s = string(r[:400]) + "…"
	}
	return s
}

// refusal — стандартный отказ модерации GigaChat: вместо вызова функции приходит текст про «чувствительные темы».
func refusal(resp chatResponse) bool {
	if len(resp.Choices) == 0 || resp.Choices[0].Message.FunctionCall != nil {
		return false
	}
	c := strings.ToLower(resp.Choices[0].Message.Content)
	return strings.Contains(c, "чувствительные темы") || strings.Contains(c, "не обладает собственным мнением")
}
