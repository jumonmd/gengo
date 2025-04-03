// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package anthropic

import (
	"context"
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
	client := anthropic.NewClient(
		option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
	)

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

	for _, part := range msg.Content {
		if part.Type == "text" {
			blocks = append(blocks, anthropic.NewTextBlock(part.Text))
		} else if part.Type == "image" {
			if !chat.IsDataURL(part.DataURL) {
				return anthropic.MessageParam{}, fmt.Errorf("invalid image data URL: %s", part.DataURL)
			}

			mimeType, encodedData, err := chat.SplitDataURL(part.DataURL)
			if err != nil {
				return anthropic.MessageParam{}, fmt.Errorf("split image data URL: %w", err)
			}
			blocks = append(blocks, anthropic.NewImageBlockBase64(mimeType, encodedData))
		}
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
		// check this spec
		return anthropic.NewAssistantMessage(blocks...), nil
	default:
		return anthropic.MessageParam{}, fmt.Errorf("unknown message role: %s", msg.Role)
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
		Messages: messages,
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
				streamer(&chat.StreamResponse{
					Type:    "text",
					Content: textDelta.Text,
				})
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
		Messages: []chat.Message{chat.NewTextMessage(chat.MessageRoleAI, content)},
		Usage:    usage,
	}, nil
}
