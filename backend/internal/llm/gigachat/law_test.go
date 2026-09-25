package gigachat

import (
	"encoding/json"
	"testing"
)

func respWith(content string) chatResponse {
	var r chatResponse
	r.Choices = append(r.Choices, struct {
		Message struct {
			Content      string `json:"content"`
			FunctionCall *struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function_call"`
		} `json:"message"`
	}{})
	r.Choices[0].Message.Content = content
	return r
}

func TestRefusalDetected(t *testing.T) {
	if !refusal(respWith("Разговоры на чувствительные темы могут быть ограничены.")) {
		t.Fatal("отказ модерации не распознан")
	}
	if refusal(respWith("Обычный ответ без отказа")) {
		t.Fatal("обычный ответ принят за отказ")
	}
}
