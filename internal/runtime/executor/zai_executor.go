package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/auth/zai"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/thinking"
	zaitranslator "github.com/router-for-me/CLIProxyAPI/v7/internal/translator/zai"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/util"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
)

type ZaiExecutor struct {
	cfg *config.Config
}

func NewZaiExecutor(cfg *config.Config) *ZaiExecutor { return &ZaiExecutor{cfg: cfg} }

func (e *ZaiExecutor) Identifier() string { return "zai" }

func (e *ZaiExecutor) createChat(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, httpClient *http.Client) (string, error) {

	model := thinking.ParseSuffix(req.Model).ModelName

	zaiReqBody, err := zaitranslator.TranslateRequest(req.Payload, model)
	if err != nil {
		return "", fmt.Errorf("zai executor: translate request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://chat.z.ai/api/v1/chats/new", bytes.NewReader(zaiReqBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	token := zai.ZaiCreds(auth)
	if token != "" {
		httpReq.Header.Set("Authorization", token)
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("zai create chat failed: %d %s", resp.StatusCode, string(b))
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	// zai response id mapping is simple: {"id":"..."} ... but it might be inside 'chat' or top level depending on response. Let's try top level first, then chat.id

	// fallback if not using gjson directly to avoid dep issues here:
	var parsedResp map[string]any
	if err := json.Unmarshal(bodyBytes, &parsedResp); err != nil {
		return "", fmt.Errorf("failed to parse chat/new response: %w", err)
	}

	idStr := ""
	if idVal, ok := parsedResp["id"].(string); ok {
		idStr = idVal
	} else if chatVal, ok := parsedResp["chat"].(map[string]any); ok {
		if idVal, ok := chatVal["id"].(string); ok {
			idStr = idVal
		}
	}

	if idStr == "" {
		return "", fmt.Errorf("zai create chat failed: empty id. Response: %s", string(bodyBytes))
	}

	return idStr, nil
}

func (e *ZaiExecutor) Execute(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (resp cliproxyexecutor.Response, err error) {
	return resp, fmt.Errorf("zai executor: non-streaming not implemented, Z.ai uses SSE only")
}

func (e *ZaiExecutor) ExecuteStream(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (stream *cliproxyexecutor.StreamResult, err error) {
	httpClient := helps.NewProxyAwareHTTPClient(ctx, e.cfg, auth, 0)

	// 1. Create chat
	chatID, err := e.createChat(ctx, auth, req, httpClient)
	if err != nil {
		return nil, err
	}

	// 2. Stream completions
	model := thinking.ParseSuffix(req.Model).ModelName

	tokenStr := zai.ZaiCreds(auth)
	tokenRaw := strings.TrimPrefix(tokenStr, "Bearer ")

	ts := time.Now().UnixMilli()
	url := fmt.Sprintf("https://chat.z.ai/api/v2/chat/completions?timestamp=%d&requestId=%s&token=%s&version=0.0.1&platform=web", ts, chatID, tokenRaw)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	util.ApplyCustomHeadersFromAttrs(httpReq, auth.Attributes)

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("zai stream failed: %d %s", httpResp.StatusCode, string(b))
	}

	reader := bufio.NewReader(httpResp.Body)

	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		defer httpResp.Body.Close()

		tb := &zaitranslator.ToolCallBuffer{}

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					out <- cliproxyexecutor.StreamChunk{Err: err}
				}
				break
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}

			dataStr := bytes.TrimPrefix(line, []byte("data: "))
			if string(dataStr) == "[DONE]" {
				out <- cliproxyexecutor.StreamChunk{Payload: []byte("data: [DONE]\n\n")}
				continue
			}

			chunk, errTranslate := zaitranslator.TranslateStreamResponse(dataStr, model, chatID, tb)
			if errTranslate != nil || chunk == nil {
				continue
			}

			outBytes, errJson := json.Marshal(chunk)
			if errJson == nil {
				out <- cliproxyexecutor.StreamChunk{Payload: append(append([]byte("data: "), outBytes...), []byte("\n\n")...)}
			}
		}
	}()

	return &cliproxyexecutor.StreamResult{
		Headers: httpResp.Header,
		Chunks:  out,
	}, nil
}

func (e *ZaiExecutor) CountTokens(ctx context.Context, auth *cliproxyauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, fmt.Errorf("zai executor: CountTokens not implemented")
}

func (e *ZaiExecutor) PrepareRequest(req *http.Request, auth *cliproxyauth.Auth) error {
	return nil
}

func (e *ZaiExecutor) HttpRequest(ctx context.Context, auth *cliproxyauth.Auth, req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("zai executor: HttpRequest not implemented")
}

func (e *ZaiExecutor) Refresh(ctx context.Context, auth *cliproxyauth.Auth) (*cliproxyauth.Auth, error) {
	return auth, nil
}
