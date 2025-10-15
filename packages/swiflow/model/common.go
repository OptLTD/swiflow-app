package model

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

type CommonModel struct {
	cfg  LLMConfig
	mu   sync.Mutex
	reqs sync.Map

	fn func() ClientInterface
}

func NewCommonModel(cfg LLMConfig) *CommonModel {
	return &CommonModel{cfg: cfg}
}

func (m *CommonModel) client() ClientInterface {
	header := map[string]string{
		"X-Project-Id": m.cfg.TaskId,
	}
	config := openai.DefaultConfig(m.cfg.ApiKey)
	config.BaseURL = m.cfg.ApiUrl
	config.AssistantVersion = m.cfg.Version
	config.APIType = openai.APIType(m.cfg.ApiType)
	config.HTTPClient = NewProxyHttpClient(header)
	return openai.NewClientWithConfig(config)
}

func (m *CommonModel) logInfo() {
	log.Println("[LLM] provider:", m.cfg.Provider, "model:", m.cfg.UseModel)
	log.Println("[LLM] task-id:", m.cfg.TaskId, "api:", m.cfg.ApiUrl)
}

func (m *CommonModel) Cancel(group string) error {
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

func (m *CommonModel) Stream(group string, msgs []Message, handle Handle) error {
	reqID := generateReqID()
	ctx, cancel := context.WithCancel(context.Background())

	m.reqs.Store(reqID, &requestContext{
		ctx: ctx, cancel: cancel, group: group,
	})
	defer m.Cancel(reqID)

	stream, err := m.client().CreateChatCompletionStream(
		ctx, Request{
			Model: m.cfg.UseModel, Messages: msgs,
			StreamOptions: &openai.StreamOptions{
				IncludeUsage: true,
			}, Stream: true,
		},
	)
	if err != nil {
		m.logInfo()
		return err
	}
	defer stream.Close()

	result := []Choice{{}}
	for {
		if resp, err := stream.Recv(); err == nil {
			if len(resp.Choices) == 0 {
				continue
			}
			choice := resp.Choices[0]
			result[0] = Choice{
				Message: openai.ChatCompletionMessage{
					Role:    choice.Delta.Role,
					Content: choice.Delta.Content,
					Refusal: choice.Delta.Refusal,
				},
				Index: choice.Index, FinishReason: choice.FinishReason,
				ContentFilterResults: choice.ContentFilterResults,
			}
			handle(result)
		} else if err == io.EOF {
			break
		} else {
			m.logInfo()
			return fmt.Errorf("LLM API ERROR: %v", err)
		}
	}
	return nil
}

func (m *CommonModel) Respond(group string, msgs []Message) ([]Choice, error) {
	reqID := generateReqID()
	ctx, cancel := context.WithCancel(context.Background())

	m.reqs.Store(reqID, &requestContext{
		ctx: ctx, cancel: cancel, group: group,
	})
	defer m.Cancel(reqID)

	resp, err := m.client().CreateChatCompletion(
		ctx, Request{Model: m.cfg.UseModel, Messages: msgs},
	)

	if err != nil {
		m.logInfo()
		return []Choice{}, fmt.Errorf("LLM API ERROR: %v", err)
	}

	return resp.Choices, nil
}
