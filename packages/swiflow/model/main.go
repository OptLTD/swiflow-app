package model

import (
	"context"
	"fmt"
	"strings"
	"swiflow/config"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// DeepSeekModel 实现了 Model 接口
type ClientInterface interface {
	CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

type Handle = func([]openai.ChatCompletionChoice)

type Choice = openai.ChatCompletionChoice
type Message = openai.ChatCompletionMessage
type Request = openai.ChatCompletionRequest
type Response = openai.ChatCompletionResponse

type requestContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	group  string
}

func generateReqID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

type LLMClient interface {
	Cancel(string) error
	Stream(string, []Message, Handle) error
	Respond(string, []Message) ([]Choice, error)
}

type LLMConfig struct {
	TaskId   string `json:"taskId"`
	ApiKey   string `json:"apiKey"`
	ApiUrl   string `json:"apiUrl"`
	Provider string `json:"provider"`
	UseModel string `json:"useModel"`
	ApiType  string `json:"apiType,omitempty"`
	Version  string `json:"version,omitempty"`
}

func GetClient(cfg *LLMConfig) LLMClient {
	switch strings.ToUpper(cfg.Provider) {
	case "GEMINI":
		return NewGeminiModel(*cfg)
	default:
		return NewCommonModel(*cfg)
	}
}

func Respond(group string, msgs []Message) ([]Choice, error) {
	name := config.GetStr("CURRENT_MODEL", "DEEPSEEK")
	return GetClient(&LLMConfig{Provider: name}).Respond(group, msgs)
}

func Stream(group string, msgs []Message, handle Handle) error {
	name := config.GetStr("CURRENT_MODEL", "DEEPSEEK")
	return GetClient(&LLMConfig{Provider: name}).Stream(group, msgs, handle)
}
func Cancel(group string) error {
	name := config.GetStr("CURRENT_MODEL", "DEEPSEEK")
	return GetClient(&LLMConfig{Provider: name}).Cancel(group)
}
