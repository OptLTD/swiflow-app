package model

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/genai"
)

type GeminiModel struct {
	cfg LLMConfig

	mu   sync.Mutex
	reqs sync.Map // map[string]*requestContext

	fn func() *genai.Client
}

// NewGeminiModel 创建一个新的 GeminiModel 实例
func NewGeminiModel(cfg LLMConfig) *GeminiModel {
	return &GeminiModel{cfg: cfg}
}

func (m *GeminiModel) client() *genai.Client {
	if m.fn != nil {
		return m.fn()
	}
	ctx := context.Background()
	httpClient := NewProxyHttpClient(nil)
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: m.cfg.ApiKey, HTTPClient: httpClient,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Gemini client: %v", err))
	}
	return client
}

func (m *GeminiModel) Cancel(group string) error {
	m.reqs.Range(func(key, val any) bool {
		reqCtx := val.(*requestContext)
		if reqCtx.group == group {
			reqCtx.cancel()
			m.reqs.Delete(key)
		}
		return true
	})
	return nil
}

func (m *GeminiModel) Stream(group string, msgs []Message, handle Handle) error {
	reqID := generateReqID()
	ctx, cancel := context.WithCancel(context.Background())
	m.reqs.Store(reqID, &requestContext{
		ctx: ctx, cancel: cancel, group: group,
	})
	defer m.Cancel(reqID)

	chat, err := m.client().Chats.Create(ctx, m.cfg.UseModel, nil, nil)
	if err != nil {
		return fmt.Errorf("Gemini chat create error: %v", err)
	}

	// 构造历史消息
	for idx, msg := range msgs {
		if idx == 0 && msg.Role == "system" {
			// Gemini API 暂不直接支持 system prompt，可考虑拼接到首条 user 消息
			continue
		}
		if idx == len(msgs)-1 {
			// 最后一条用户输入，流式发送
			for result, err := range chat.SendMessageStream(ctx, genai.Part{Text: msg.Content}) {
				if err != nil {
					return fmt.Errorf("Gemini stream error: %v", err)
				}
				choices := []Choice{{Message: Message{
					Role: "assistant", Content: result.Text(),
				}}}
				handle(choices)
			}
			continue
		}
		// _ = chat.History(false) // 这里不需要手动追加历史，SDK 会自动维护
	}
	return nil
}

func (m *GeminiModel) Respond(group string, msgs []Message) ([]Choice, error) {
	reqID := generateReqID()
	ctx, cancel := context.WithCancel(context.Background())

	m.reqs.Store(reqID, &requestContext{
		ctx: ctx, cancel: cancel, group: group,
	})
	defer m.Cancel(reqID)

	chat, err := m.client().Chats.Create(ctx, m.cfg.UseModel, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("Gemini chat create error: %v", err)
	}

	var resp *genai.GenerateContentResponse
	for idx, msg := range msgs {
		if idx == 0 && msg.Role == "system" {
			continue
		}
		if idx == len(msgs)-1 {
			resp, err = chat.SendMessage(ctx, genai.Part{Text: msg.Content})
			continue
		}
		// _ = chat.History(false)
	}
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %v", err)
	}

	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0 {
		text := resp.Text()
		choices := []Choice{{Message: Message{
			Role: "assistant", Content: text,
		}}}
		return choices, nil
	}
	return []Choice{}, fmt.Errorf("Gemini API returned empty response")
}
