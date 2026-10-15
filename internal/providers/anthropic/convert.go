package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/cloudwego/eino/schema"
)

// toMessages splits the conversation into the system prompt and the alternating user/assistant
// messages. Tool results become tool_result blocks of a user message; consecutive ones share it,
// which is how the API wants the answers to parallel tool calls.
func toMessages(in []*schema.Message) (system []sdk.TextBlockParam, out []sdk.MessageParam, err error) {
	for i := 0; i < len(in); i++ {
		m := in[i]
		switch m.Role {
		case schema.System:
			if m.Content != "" {
				system = append(system, sdk.TextBlockParam{Text: m.Content})
			}
		case schema.User:
			if m.Content != "" {
				out = append(out, sdk.NewUserMessage(sdk.NewTextBlock(m.Content)))
			}
		case schema.Assistant:
			blocks, err := assistantBlocks(m)
			if err != nil {
				return nil, nil, err
			}
			if len(blocks) > 0 {
				out = append(out, sdk.NewAssistantMessage(blocks...))
			}
		case schema.Tool:
			var results []sdk.ContentBlockParamUnion
			for ; i < len(in) && in[i].Role == schema.Tool; i++ {
				results = append(results, sdk.NewToolResultBlock(in[i].ToolCallID, in[i].Content, false))
			}
			i-- // the loop's own i++ moves past the last tool message
			out = append(out, sdk.NewUserMessage(results...))
		default:
			return nil, nil, fmt.Errorf("anthropic: unsupported message role %q", m.Role)
		}
	}
	return system, out, nil
}

// assistantBlocks is the assistant's text followed by its tool calls as tool_use blocks.
func assistantBlocks(m *schema.Message) ([]sdk.ContentBlockParamUnion, error) {
	var blocks []sdk.ContentBlockParamUnion
	if m.Content != "" {
		blocks = append(blocks, sdk.NewTextBlock(m.Content))
	}
	for _, c := range m.ToolCalls {
		args := json.RawMessage(c.Function.Arguments)
		if len(args) == 0 {
			args = json.RawMessage("{}")
		}
		if !json.Valid(args) {
			return nil, fmt.Errorf("anthropic: arguments of tool call %s (%s) are not JSON", c.ID, c.Function.Name)
		}
		blocks = append(blocks, sdk.NewToolUseBlock(c.ID, args, c.Function.Name))
	}
	return blocks, nil
}

// toTools turns Eino tool definitions into Anthropic tools. The JSON schema of the parameters
// becomes the input schema: properties and required are the SDK's own fields, anything else
// (additionalProperties, $defs) rides along as extra fields.
func toTools(infos []*schema.ToolInfo) ([]sdk.ToolUnionParam, error) {
	var tools []sdk.ToolUnionParam
	for _, info := range infos {
		js, err := info.ParamsOneOf.ToJSONSchema()
		if err != nil {
			return nil, fmt.Errorf("anthropic: tool %s: %w", info.Name, err)
		}
		// A tool without parameters still sends an input schema: {"type":"object","properties":{}}.
		in := sdk.ToolInputSchemaParam{Properties: map[string]any{}}
		if js != nil {
			raw, err := json.Marshal(js)
			if err != nil {
				return nil, fmt.Errorf("anthropic: tool %s: %w", info.Name, err)
			}
			var extra map[string]any
			if err := json.Unmarshal(raw, &extra); err != nil {
				return nil, fmt.Errorf("anthropic: tool %s: %w", info.Name, err)
			}
			if p, ok := extra["properties"]; ok {
				in.Properties = p
			}
			in.Required = js.Required
			delete(extra, "type")
			delete(extra, "properties")
			delete(extra, "required")
			in.ExtraFields = extra
		}
		t := sdk.ToolParam{Name: info.Name, InputSchema: in}
		if info.Desc != "" {
			t.Description = sdk.String(info.Desc)
		}
		tools = append(tools, sdk.ToolUnionParam{OfTool: &t})
	}
	return tools, nil
}

// toMessage maps the API reply: text blocks joined, tool_use blocks as tool calls, the stop reason
// as the finish reason, and usage (the prompt count includes cached tokens, as Eino expects).
func toMessage(r *sdk.Message) *schema.Message {
	var text strings.Builder
	var calls []schema.ToolCall
	for _, b := range r.Content {
		switch b.Type {
		case "text":
			text.WriteString(b.Text)
		case "tool_use":
			calls = append(calls, schema.ToolCall{
				ID: b.ID, Type: "function",
				Function: schema.FunctionCall{Name: b.Name, Arguments: string(b.Input)},
			})
		}
	}
	u := r.Usage
	prompt := int(u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens)
	return &schema.Message{
		Role: schema.Assistant, Content: text.String(), ToolCalls: calls,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: string(r.StopReason),
			Usage: &schema.TokenUsage{
				PromptTokens:       prompt,
				PromptTokenDetails: schema.PromptTokenDetails{CachedTokens: int(u.CacheReadInputTokens), CacheWriteTokens: int(u.CacheCreationInputTokens)},
				CompletionTokens:   int(u.OutputTokens),
				TotalTokens:        prompt + int(u.OutputTokens),
			},
		},
	}
}
