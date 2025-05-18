// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/jumonmd/gengo/chat"
)

const structuredOutputPrompt = `
Please respond with json in the following json_schema:

%s

Make sure to return an instance of the JSON, not the schema itself.
`

func Generate(ctx context.Context, r *chat.Request, opts ...chat.Option) (*chat.Response, error) {
	opt := chat.NewOptions(opts...)

	options := []option.RequestOption{option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY"))}
	if opt.BaseURL != "" {
		options = append(options, option.WithBaseURL(opt.BaseURL))
	}

	client := anthropic.NewClient(options...)

	messages := []anthropic.MessageParam{}
	if r.ResponseSchema != nil {
		messages = append(messages,
			anthropic.NewUserMessage(anthropic.NewTextBlock(fmt.Sprintf(structuredOutputPrompt, string(r.ResponseSchema.JSON())))))
	}
	for _, msg := range r.Messages {
		param, err := convertMessage(&msg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}
		messages = append(messages, param)
	}

	params := convertChatRequest(r, messages)

	// tool call will not use stream for simplicity
	if opt.Streamer != nil && len(params.Tools) == 0 {
		resp, err := handleStreaming(ctx, client, params, opt.Streamer)
		if err != nil {
			return nil, fmt.Errorf("streaming error: %w", err)
		}
		opt.ModelCatalog.CalculateCost(r.Model, resp.Usage)
		return resp, nil
	}

	message, err := client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic message creation error: %w", err)
	}

	resp := messageToResponse(message)
	resp.Model = r.Model
	opt.ModelCatalog.CalculateCost(r.Model, resp.Usage)
	return resp, nil
}

func convertChatRequest(r *chat.Request, messages []anthropic.MessageParam) anthropic.MessageNewParams {
	params := anthropic.MessageNewParams{
		Model:    r.Model,
		Messages: messages,
	}

	if len(r.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, len(r.Tools))
		for i, tool := range r.Tools {
			toolParam := anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: tool.InputSchema.PropertiesJSON(),
				},
			}

			tools[i] = anthropic.ToolUnionParam{
				OfTool: &toolParam,
			}
		}
		params.Tools = tools
		if r.MustCallTool {
			params.ToolChoice = anthropic.ToolChoiceUnionParam{
				OfToolChoiceAny: &anthropic.ToolChoiceAnyParam{},
			}
		}
	}

	if r.Config.MaxTokens == 0 {
		params.MaxTokens = 2048
	} else {
		params.MaxTokens = int64(r.Config.MaxTokens)
	}

	if r.Config.Temperature != 0 {
		params.Temperature = anthropic.Float(float64(r.Config.Temperature))
	}
	if r.Config.TopP != 0 {
		params.TopP = anthropic.Float(float64(r.Config.TopP))
	}
	if len(r.Config.StopWords) > 0 {
		params.StopSequences = r.Config.StopWords
	}

	return params
}

func convertMessage(msg *chat.Message) (anthropic.MessageParam, error) {
	var blocks []anthropic.ContentBlockParamUnion
	switch {
	case msg.ToolResponse != nil:
		blocks = append(blocks, anthropic.NewToolResultBlock(msg.ToolResponse.ID, msg.ToolResponse.Result, false))
	case msg.ToolCall != nil:
		var input map[string]any
		if err := json.Unmarshal([]byte(msg.ToolCall.Arguments), &input); err != nil {
			return anthropic.MessageParam{}, fmt.Errorf("unmarshal tool call arguments: %w", err)
		}
		block := anthropic.ToolUseBlockParam{
			ID:    msg.ToolCall.ID,
			Input: input,
			Name:  msg.ToolCall.Name,
			Type:  "tool_use",
		}
		blocks = append(blocks, anthropic.ContentBlockParamUnion{OfRequestToolUseBlock: &block})
	default:
		blks, err := convertContentPart(msg)
		if err != nil {
			return anthropic.MessageParam{}, fmt.Errorf("convert content part: %w", err)
		}
		blocks = append(blocks, blks...)
	}

	if len(blocks) == 0 {
		return anthropic.MessageParam{}, fmt.Errorf("no valid content blocks")
	}

	switch msg.Role {
	case chat.MessageRoleSystem:
		return anthropic.NewUserMessage(anthropic.NewTextBlock("system: " + msg.Content[0].Text)), nil
	case chat.MessageRoleHuman:
		return anthropic.NewUserMessage(blocks...), nil
	case chat.MessageRoleAI:
		return anthropic.NewAssistantMessage(blocks...), nil
	case chat.MessageRoleTool:
		return anthropic.NewUserMessage(blocks...), nil
	default:
		return anthropic.MessageParam{}, fmt.Errorf("unknown message role: %s", msg.Role)
	}
}

func convertContentPart(msg *chat.Message) ([]anthropic.ContentBlockParamUnion, error) {
	blocks := []anthropic.ContentBlockParamUnion{}
	for _, part := range msg.Content {
		switch part.Type {
		case "text":
			blocks = append(blocks, anthropic.NewTextBlock(part.Text))
		case "image":
			if !chat.IsDataURL(part.DataURL) {
				return nil, fmt.Errorf("invalid image data URL: %s", part.DataURL)
			}

			mimeType, encodedData, err := chat.SplitDataURL(part.DataURL)
			if err != nil {
				return nil, fmt.Errorf("split image data URL: %w", err)
			}
			blocks = append(blocks, anthropic.NewImageBlockBase64(mimeType, encodedData))
		}
	}
	return blocks, nil
}

func convertFinishReason(reason anthropic.MessageStopReason) chat.FinishReason {
	switch reason {
	case anthropic.MessageStopReasonEndTurn,
		anthropic.MessageStopReasonStopSequence:
		return chat.FinishReasonStop
	case anthropic.MessageStopReasonMaxTokens:
		return chat.FinishReasonMaxTokens
	case anthropic.MessageStopReasonToolUse:
		return chat.FinishReasonToolUse
	default:
		return chat.FinishReasonUnknown
	}
}

func messageToResponse(message *anthropic.Message) *chat.Response {
	messages := []chat.Message{}

	for _, block := range message.Content {
		switch block := block.AsAny().(type) {
		case anthropic.TextBlock:
			messages = append(messages, chat.NewTextMessage(chat.MessageRoleAI, block.Text))
		case anthropic.ToolUseBlock:
			toolCall := chat.NewToolCallMessage(block.Name, block.ID, string(block.Input))
			messages = append(messages, toolCall)
		}
	}

	return &chat.Response{
		Messages:     messages,
		FinishReason: convertFinishReason(message.StopReason),
		Usage: &chat.Usage{
			InputTokens:  int(message.Usage.InputTokens),
			OutputTokens: int(message.Usage.OutputTokens),
			TotalTokens:  int(message.Usage.InputTokens + message.Usage.OutputTokens),
		},
	}
}

func handleStreaming(ctx context.Context, client anthropic.Client, params anthropic.MessageNewParams, streamer chat.Streamer) (*chat.Response, error) {
	stream := client.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	content := ""
	usage := &chat.Usage{}
	for stream.Next() {
		event := stream.Current()

		switch eventVariant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			if textDelta, ok := eventVariant.Delta.AsAny().(anthropic.TextDelta); ok {
				content += textDelta.Text
				err := streamer(&chat.StreamResponse{
					Type:    "text",
					Content: textDelta.Text,
				})
				if err != nil {
					return nil, fmt.Errorf("stream: %w", err)
				}
			}
		case anthropic.MessageStartEvent:
			usage.InputTokens = int(eventVariant.Message.Usage.InputTokens)
		case anthropic.MessageDeltaEvent:
			usage.OutputTokens += int(eventVariant.Usage.OutputTokens)
		}
	}

	if err := stream.Err(); err != nil {
		return nil, err
	}

	usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	return &chat.Response{
		Messages:     []chat.Message{chat.NewTextMessage(chat.MessageRoleAI, content)},
		FinishReason: "stop",
		Usage:        usage,
	}, nil
}
