package zai

import (
	"encoding/json"
	"testing"
)

func TestTranslateRequest_ThinkingBudget(t *testing.T) {
	rawJSON := []byte(`{
		"messages": [{"role": "user", "content": "hello"}],
		"thinking": {"budget_tokens": 0}
	}`)

	res, err := TranslateRequest(rawJSON, "glm-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var req ZaiChatCreateRequest
	if err := json.Unmarshal(res, &req); err != nil {
		t.Fatalf("unexpected error unmarshaling: %v", err)
	}

	if req.Chat.EnableThinking != false {
		t.Errorf("expected enable_thinking to be false, got true")
	}
}

func TestTranslateResponse_ThinkingPhase(t *testing.T) {
	data := []byte(`{"type":"chat:completion","data":{"delta_content":"I am thinking...","phase":"thinking"}}`)
	tb := &ToolCallBuffer{}

	chunk, err := TranslateStreamResponse(data, "glm-5", "id-123", tb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunk.Choices[0].Delta.ReasoningContent != "I am thinking..." {
		t.Errorf("expected reasoning content 'I am thinking...', got %q", chunk.Choices[0].Delta.ReasoningContent)
	}
	if chunk.Choices[0].Delta.Content != "" {
		t.Errorf("expected empty content, got %q", chunk.Choices[0].Delta.Content)
	}
}
