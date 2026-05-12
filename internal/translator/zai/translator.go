package zai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	Index    int          `json:"index"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function ToolFunction `json:"function,omitempty"`
}

type ToolFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
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

	// Option B: Inject Tool Calling Instructions if tools exist in request
	if root.Get("tools").Exists() {
		toolsJSON := root.Get("tools").Raw
		instruction := "\n\nSei un agente in grado di usare strumenti. Se vuoi usare lo strumento X, rispondi ESATTAMENTE in formato XML <tool_call><name>X</name><args>...</args></tool_call>\n" + toolsJSON

		// Append this to the last message (assuming there is one)
		if lastID != "" {
			msg := messagesMap[lastID]
			msg.Content += instruction
			messagesMap[lastID] = msg
		}
	}

	enableThinking := true
	if root.Get("thinking.budget_tokens").Exists() {
		budget := root.Get("thinking.budget_tokens").Int()
		if budget == 0 {
			enableThinking = false
		}
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
			EnableThinking: enableThinking,
			AutoWebSearch:  false,
			MessageVersion: 1,
			Extra:          make(map[string]any),
			Timestamp:      time.Now().UnixMilli(),
			Type:           "default",
		},
	}

	return json.Marshal(zaiReq)
}

// ToolCallBuffer is used to accumulate XML fragments across multiple chunks
type ToolCallBuffer struct {
	Active bool
	Buffer string
}

func TranslateStreamResponse(data []byte, model string, id string, tb *ToolCallBuffer) (*ChatCompletionChunk, error) {
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
		content := zResp.Data.DeltaContent

		if tb.Active {
			tb.Buffer += content

			// Check if we reached the end of the tool call
			if strings.Contains(tb.Buffer, "</tool_call>") {
				// We have a full tool call
				nameStart := strings.Index(tb.Buffer, "<name>") + 6
				nameEnd := strings.Index(tb.Buffer, "</name>")

				argsStart := strings.Index(tb.Buffer, "<args>") + 6
				argsEnd := strings.Index(tb.Buffer, "</args>")

				if nameStart > 5 && nameEnd > nameStart && argsStart > 5 && argsEnd > argsStart {
					name := tb.Buffer[nameStart:nameEnd]
					args := tb.Buffer[argsStart:argsEnd]

					chunk.Choices[0].Delta.Content = ""
					chunk.Choices[0].Delta.ToolCalls = []ToolCall{
						{
							Index: 0,
							ID:    "call_" + uuid.New().String()[:8],
							Type:  "function",
							Function: ToolFunction{
								Name:      name,
								Arguments: args,
							},
						},
					}
				} else {
					// Fallback if parsing fails
					chunk.Choices[0].Delta.Content = tb.Buffer
				}

				// Reset buffer
				tb.Active = false
				tb.Buffer = ""
			} else {
				// Still accumulating, do not output anything yet
				chunk.Choices[0].Delta.Content = ""
			}
		} else {
			// Fast check if it might be starting a tool call
			if strings.HasPrefix(content, "<tool_call>") || (len(content) > 0 && content[0] == '<' && strings.HasPrefix("<tool_call>", content)) {
				tb.Active = true
				tb.Buffer = content
				chunk.Choices[0].Delta.Content = "" // suppress output while buffering
			} else {
				chunk.Choices[0].Delta.Content = content
			}
		}
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
		// Create a local buffer if param is nil, though in reality param should be persistent across stream chunks
		var tb *ToolCallBuffer
		if param != nil && *param != nil {
			tb = (*param).(*ToolCallBuffer)
		} else {
			tb = &ToolCallBuffer{}
			if param != nil {
				*param = tb
			}
		}

		chunk, err := TranslateStreamResponse(rawJSON, model, "stream-id", tb)
		if err != nil || chunk == nil {
			return [][]byte{rawJSON} // Fallback
		}
		res, _ := json.Marshal(chunk)
		return [][]byte{res}
	}
}

func NewZaiResponseNonStreamTransform() translator.ResponseNonStreamTransform {
	return func(ctx context.Context, model string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) []byte {
		tb := &ToolCallBuffer{}
		chunk, err := TranslateStreamResponse(rawJSON, model, "id-123", tb)
		if err != nil || chunk == nil {
			return rawJSON
		}
		res, _ := json.Marshal(chunk)
		return res
	}
}

func NewZaiTokenCountTransform() translator.ResponseTokenCountTransform {
	return func(ctx context.Context, count int64) []byte {
		return nil
	}
}
