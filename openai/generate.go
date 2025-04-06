// SPDX-FileCopyrightText: 2025 Masa Cento
// SPDX-License-Identifier: MIT

package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jumonmd/gengo/chat"
	"github.com/jumonmd/gengo/jsonschema"
	"github.com/sashabaranov/go-openai"
)

func Generate(ctx context.Context, r *chat.Request, opts ...chat.Option) (*chat.Response, error) {
	opt := chat.NewOptions(opts...)

	cfg := openai.DefaultConfig(os.Getenv("OPENAI_API_KEY"))
	if opt.BaseURL != "" {
		cfg.BaseURL = opt.BaseURL
	}
	client := openai.NewClientWithConfig(cfg)

	req := convertChatRequest(r)

	// tool call will not use stream for simplicity
	if opt.Streamer != nil && len(req.Tools) == 0 {
		resp, err := chatCompletionStream(ctx, client, req, opt.Streamer)
		if err != nil {
			return nil, fmt.Errorf("chat completion stream: %w", err)
		}
		opt.ModelCatalog.CalculateCost(r.Model, resp.Usage)
		return resp, nil
	}

	resp, err := chatCompletion(ctx, client, req)
	if err != nil {
		return nil, fmt.Errorf("chat completion: %w", err)
	}

	opt.ModelCatalog.CalculateCost(r.Model, resp.Usage)
	return resp, nil
}

func chatCompletion(ctx context.Context, client *openai.Client, r openai.ChatCompletionRequest) (*chat.Response, error) {
	resp, err := client.CreateChatCompletion(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("chat completion: %w", err)
	}
	msgs := []chat.Message{}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices")
	}

	content := resp.Choices[0].Message.Content
	if content != "" {
		msgs = append(msgs, chat.NewTextMessage(chat.MessageRoleAI, content))
	}
	for _, toolcall := range resp.Choices[0].Message.ToolCalls {
		msgs = append(msgs, chat.NewToolCallMessage(toolcall.Function.Name, toolcall.ID, toolcall.Function.Arguments))
	}

	chatresp := &chat.Response{
		Model:        r.Model,
		Messages:     msgs,
		FinishReason: convertFinishReason(resp.Choices[0].FinishReason),
		Usage: &chat.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
	return chatresp, nil
}

func chatCompletionStream(ctx context.Context, client *openai.Client, r openai.ChatCompletionRequest, streamer chat.Streamer) (*chat.Response, error) {
	r.Stream = true
	r.StreamOptions = &openai.StreamOptions{
		IncludeUsage: true,
	}
	stream, err := client.CreateChatCompletionStream(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("chat completion stream: %w", err)
	}
	defer stream.Close()

	usage := &chat.Usage{}
	content := ""
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				// chat completion stream is done
				return &chat.Response{
					Model:        r.Model,
					Messages:     []chat.Message{chat.NewTextMessage(chat.MessageRoleAI, content)},
					FinishReason: "stop",
					Usage:        usage,
				}, nil
			} else if err != nil {
				return nil, fmt.Errorf("chat completion stream recv: %w", err)
			}

			if response.Usage != nil {
				usage = chatUsage(response.Usage)
			}

			if len(response.Choices) == 0 {
				continue
			}

			// stream chunk content
			if c := response.Choices[0].Delta.Content; c != "" {
				content += c
				err := streamer(&chat.StreamResponse{
					Type:    "text",
					Content: c,
				})
				if err != nil {
					return nil, fmt.Errorf("stream: %w", err)
				}
			}
		}
	}
}

func chatUsage(usage *openai.Usage) *chat.Usage {
	return &chat.Usage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}
}

func convertChatRequest(r *chat.Request) openai.ChatCompletionRequest {
	msgs := []openai.ChatCompletionMessage{}
	for _, msg := range r.Messages {
		msgs = append(msgs, convertChatMessage(&msg))
	}

	tools := []openai.Tool{}
	for _, tool := range r.Tools {
		tools = append(tools, convertChatTool(&tool))
	}

	req := openai.ChatCompletionRequest{
		Model:    r.Model,
		Messages: msgs,
		Tools:    tools,
	}

	req.MaxTokens = int(r.Config.MaxTokens)
	req.Temperature = r.Config.Temperature
	req.TopP = r.Config.TopP
	req.FrequencyPenalty = r.Config.FrequencyPenalty
	req.PresencePenalty = r.Config.PresencePenalty
	req.Stop = r.Config.StopWords

	if r.ResponseSchema != nil {
		req.ResponseFormat = convertChatSchema(r.ResponseSchema)
	}

	if r.MustCallTool {
		req.ToolChoice = "required"
	}

	return req
}

func convertChatMessage(msg *chat.Message) openai.ChatCompletionMessage {
	parts := []openai.ChatMessagePart{}

	if msg.IsToolResponse() {
		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    msg.ToolResponse.Result,
			Name:       msg.ToolResponse.Name,
			ToolCallID: msg.ToolResponse.ID,
		}
	}
	for _, part := range msg.Content {
		parts = append(parts, convertContentPart(&part))
	}

	toolcalls := []openai.ToolCall{}
	if msg.IsToolCall() {
		toolcalls = append(toolcalls, openai.ToolCall{
			ID:   msg.ToolCall.ID,
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      msg.ToolCall.Name,
				Arguments: msg.ToolCall.Arguments,
			},
		})
	}
	return openai.ChatCompletionMessage{
		Role:         convertChatRole(msg.Role),
		MultiContent: parts,
		ToolCalls:    toolcalls,
	}
}

func convertChatRole(role chat.MessageRole) string {
	switch role {
	case chat.MessageRoleSystem:
		return openai.ChatMessageRoleSystem
	case chat.MessageRoleHuman:
		return openai.ChatMessageRoleUser
	case chat.MessageRoleAI:
		return openai.ChatMessageRoleAssistant
	case chat.MessageRoleTool:
		return openai.ChatMessageRoleTool
	}

	return openai.ChatMessageRoleUser
}

func convertContentPart(part *chat.ContentPart) openai.ChatMessagePart {
	if part.Type == "image" {
		return openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL:    part.DataURL,
				Detail: openai.ImageURLDetailAuto,
			},
		}
	}

	return openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: part.Text,
	}
}

func convertChatTool(tool *chat.Tool) openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Strict:      false,
			Parameters:  tool.InputSchema.JSON(),
		},
	}
}

func convertChatSchema(schema jsonschema.Schema) *openai.ChatCompletionResponseFormat {
	return &openai.ChatCompletionResponseFormat{
		Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
		JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
			Name:   "response",
			Schema: schema.JSON(),
		},
	}
}

func convertFinishReason(reason openai.FinishReason) chat.FinishReason {
	switch reason {
	case openai.FinishReasonStop:
		return chat.FinishReasonStop
	case openai.FinishReasonLength:
		return chat.FinishReasonMaxTokens
	case openai.FinishReasonFunctionCall,
		openai.FinishReasonToolCalls:
		return chat.FinishReasonToolUse
	case openai.FinishReasonContentFilter:
		return chat.FinishReasonSafety
	case openai.FinishReasonNull:
		return chat.FinishReasonUnknown
	default:
		return chat.FinishReasonUnknown
	}
}
