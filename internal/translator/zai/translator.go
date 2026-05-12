package zai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
	"github.com/tidwall/gjson"
)

type ZaiMessage struct {
	ID          string   `json:"id"`
	ParentID    *string  `json:"parentId"`
	ChildrenIds []string `json:"childrenIds"`
	Role        string   `json:"role"`
	Content     string   `json:"content"`
	Timestamp   int64    `json:"timestamp"`
	Models      []string `json:"models"`
}

type ZaiChatHistory struct {
	Messages  map[string]ZaiMessage `json:"messages"`
	CurrentId string                `json:"currentId"`
}

type ZaiChat struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Models         []string       `json:"models"`
	Params         map[string]any `json:"params"`
	History        ZaiChatHistory `json:"history"`
	Tags           []any          `json:"tags"`
	Flags          []any          `json:"flags"`
	Features       []any          `json:"features"`
	McpServers     []any          `json:"mcp_servers"`
	EnableThinking bool           `json:"enable_thinking"`
	AutoWebSearch  bool           `json:"auto_web_search"`
	MessageVersion int            `json:"message_version"`
	Extra          map[string]any `json:"extra"`
	Timestamp      int64          `json:"timestamp"`
	Type           string         `json:"type"`
}

type ZaiChatCreateRequest struct {
	Chat ZaiChat `json:"chat"`
}

type ZaiStreamResponse struct {
	Type string `json:"type"`
	Data struct {
		DeltaContent string `json:"delta_content"`
		Phase        string `json:"phase"`
	} `json:"data"`
}

type ChatCompletionChunkChoice struct {
	Index        int                      `json:"index"`
	Delta        ChatCompletionChunkDelta `json:"delta"`
	FinishReason *string                  `json:"finish_reason"`
}

type ChatCompletionChunkDelta struct {
	Role             string `json:"role,omitempty"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type ChatCompletionChunk struct {
	ID      string                      `json:"id"`
	Object  string                      `json:"object"`
	Created int64                       `json:"created"`
	Model   string                      `json:"model"`
	Choices []ChatCompletionChunkChoice `json:"choices"`
}

func TranslateRequest(rawJSON []byte, model string) ([]byte, error) {
	root := gjson.ParseBytes(rawJSON)

	modelList := []string{model}
	messagesMap := make(map[string]ZaiMessage)

	var lastID string
	var prevID *string
	now := time.Now().Unix()

	for i, msg := range root.Get("messages").Array() {
		id := uuid.New().String()

		role := msg.Get("role").String()
		if role == "system" {
			role = "user"
		}

		content := ""

		if msg.Get("content").Type == gjson.String {
			content = msg.Get("content").String()
		} else if msg.Get("content").Type == gjson.JSON {
			for _, part := range msg.Get("content").Array() {
				if part.Get("type").String() == "text" {
					content += part.Get("text").String()
				}
			}
		}

		zm := ZaiMessage{
			ID:          id,
			ParentID:    prevID,
			ChildrenIds: []string{},
			Role:        role,
			Content:     content,
			Timestamp:   now + int64(i),
			Models:      modelList,
		}

		if prevID != nil {
			prevMsg := messagesMap[*prevID]
			prevMsg.ChildrenIds = append(prevMsg.ChildrenIds, id)
			messagesMap[*prevID] = prevMsg
		}

		messagesMap[id] = zm
		lastID = id

		tmpID := id
		prevID = &tmpID
	}

	zaiReq := ZaiChatCreateRequest{
		Chat: ZaiChat{
			ID:     "",
			Title:  "New Chat",
			Models: modelList,
			Params: make(map[string]any),
			History: ZaiChatHistory{
				Messages:  messagesMap,
				CurrentId: lastID,
			},
			Tags:           []any{},
			Flags:          []any{},
			Features:       []any{},
			McpServers:     []any{},
			EnableThinking: true,
			AutoWebSearch:  false,
			MessageVersion: 1,
			Extra:          make(map[string]any),
			Timestamp:      time.Now().UnixMilli(),
			Type:           "default",
		},
	}

	return json.Marshal(zaiReq)
}

func TranslateStreamResponse(data []byte, model string, id string) (*ChatCompletionChunk, error) {
	var zResp ZaiStreamResponse
	if err := json.Unmarshal(data, &zResp); err != nil {
		return nil, fmt.Errorf("failed to decode zai response: %w", err)
	}

	if zResp.Type != "chat:completion" {
		return nil, nil // Ignore non-completion events
	}

	chunk := &ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Model:   model,
		Created: time.Now().Unix(),
		Choices: []ChatCompletionChunkChoice{
			{
				Index: 0,
				Delta: ChatCompletionChunkDelta{
					Role: "assistant",
				},
			},
		},
	}

	if zResp.Data.Phase == "thinking" {
		chunk.Choices[0].Delta.ReasoningContent = zResp.Data.DeltaContent
	} else {
		chunk.Choices[0].Delta.Content = zResp.Data.DeltaContent
	}

	return chunk, nil
}

func NewZaiRequestTransform() translator.RequestTransform {
	return func(model string, rawJSON []byte, stream bool) []byte {
		res, _ := TranslateRequest(rawJSON, model)
		return res
	}
}

func NewZaiResponseStreamTransform() translator.ResponseStreamTransform {
	return func(ctx context.Context, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) [][]byte {
		return [][]byte{rawJSON}
	}
}

func NewZaiResponseNonStreamTransform() translator.ResponseNonStreamTransform {
	return func(ctx context.Context, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) []byte {
		return rawJSON
	}
}

func NewZaiTokenCountTransform() translator.ResponseTokenCountTransform {
	return func(ctx context.Context, count int64) []byte {
		return nil
	}
}
