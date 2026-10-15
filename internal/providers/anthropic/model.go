// Package anthropic is a small Eino chat model for the Anthropic Messages API, on the core
// anthropic-sdk-go package only. eino-ext's claude model also imports the SDK's bedrock and vertex
// packages, which pull the AWS and Google Cloud SDKs into the app.
//
// It covers what the built-in agent needs: system prompt, text, tool definitions, tool calls and
// results, usage. Images, thinking, caching and tool choice are not supported. Stream wraps
// Generate, because the agent does not stream.
package anthropic

import (
	"context"
	"errors"
	"net/http"
	"time"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Config is how to reach the API.
type Config struct {
	APIKey string // sent as x-api-key; empty sends none
	// BaseURL replaces https://api.anthropic.com, e.g. a gateway; empty keeps the default.
	BaseURL string
	Model   string
	// MaxTokens is the default cap on the reply; model.WithMaxTokens overrides it per call.
	MaxTokens int
	// HTTPClient is the transport; nil uses the SDK's default.
	HTTPClient *http.Client
	// RequestTimeout bounds one request; zero leaves the SDK's default.
	RequestTimeout time.Duration
}

// ChatModel implements model.ToolCallingChatModel.
type ChatModel struct {
	client sdk.Client
	cfg    Config
	tools  []*schema.ToolInfo
}

var _ model.ToolCallingChatModel = (*ChatModel)(nil)

// New builds the model. It does not contact the API.
func New(cfg Config) (*ChatModel, error) {
	if cfg.Model == "" {
		return nil, errors.New("anthropic: no model")
	}
	if cfg.MaxTokens <= 0 {
		return nil, errors.New("anthropic: MaxTokens must be positive")
	}
	// WithoutEnvironmentDefaults: the key and address come from the app, never from
	// ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN or ANTHROPIC_BASE_URL in the environment.
	opts := []option.RequestOption{option.WithoutEnvironmentDefaults()}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.HTTPClient != nil {
		opts = append(opts, option.WithHTTPClient(cfg.HTTPClient))
	}
	if cfg.RequestTimeout > 0 {
		opts = append(opts, option.WithRequestTimeout(cfg.RequestTimeout))
	}
	return &ChatModel{client: sdk.NewClient(opts...), cfg: cfg}, nil
}

// WithTools implements model.ToolCallingChatModel. It returns a copy; the receiver is unchanged.
func (m *ChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if len(tools) == 0 {
		return nil, errors.New("anthropic: no tools to bind")
	}
	next := *m
	next.tools = append([]*schema.ToolInfo(nil), tools...)
	return &next, nil
}

// Generate implements model.BaseChatModel.
func (m *ChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	params, err := m.params(in, opts)
	if err != nil {
		return nil, err
	}
	resp, err := m.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return toMessage(resp), nil
}

// Stream implements model.BaseChatModel with a single chunk: the whole reply.
func (m *ChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderFromArray([]*schema.Message{msg}), nil
}

// params turns the conversation and the call options into a Messages request.
func (m *ChatModel) params(in []*schema.Message, opts []model.Option) (sdk.MessageNewParams, error) {
	o := model.GetCommonOptions(&model.Options{Model: &m.cfg.Model, Tools: m.tools}, opts...)
	maxTokens := m.cfg.MaxTokens
	if o.MaxTokens != nil {
		maxTokens = *o.MaxTokens
	}
	system, msgs, err := toMessages(in)
	if err != nil {
		return sdk.MessageNewParams{}, err
	}
	tools, err := toTools(o.Tools)
	if err != nil {
		return sdk.MessageNewParams{}, err
	}
	p := sdk.MessageNewParams{
		Model: sdk.Model(*o.Model), MaxTokens: int64(maxTokens),
		System: system, Messages: msgs, Tools: tools, StopSequences: o.Stop,
	}
	if o.Temperature != nil {
		p.Temperature = sdk.Float(float64(*o.Temperature))
	}
	if o.TopP != nil {
		p.TopP = sdk.Float(float64(*o.TopP))
	}
	return p, nil
}
