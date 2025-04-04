package main

import (
	"context"
	"fmt"

	"github.com/jumonmd/gengo"
	"github.com/jumonmd/gengo/chat"
	"github.com/jumonmd/gengo/jsonschema"
)

// set OPENAI_API_KEY env.
func main() {
	ctx := context.Background()

	msgs := []chat.Message{
		chat.NewTextMessage(chat.MessageRoleHuman, "What is the weather in Tokyo?"),
	}

	// Simple text generation
	resp, err := gengo.Generate(ctx, &chat.Request{
		Model:    "gpt-4o-mini", // or gemini-2.0-flash, claude-3-5-haiku-latest
		Messages: msgs,
		Tools: []chat.Tool{
			{
				Name:        "get_current_weather",
				Description: "Get the current weather in a given location",
				InputSchema: jsonschema.MustParseJSONString(`{"type": "object", "properties": {"location": {"type": "string"}}}`),
			},
		},
		MustCallTool: true,
	})
	if err != nil {
		panic(err)
	}

	msgs = append(msgs, resp.Messages...)

	for _, msg := range resp.ToolCalls() {
		msgs = append(msgs, chat.NewToolResponseMessage("get_current_weather", msg.ToolCall.ID, "Rainy"))
	}

	resp, err = gengo.Generate(ctx, &chat.Request{
		Model:    "gpt-4o-mini", // or gemini-2.0-flash, claude-3-5-haiku-latest
		Messages: msgs,
	})
	if err != nil {
		panic(err)
	}

	// Print the response
	for _, msg := range resp.Messages {
		fmt.Printf("%+v\n", msg.ContentString())
	}
}
