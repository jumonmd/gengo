package main

import (
	"context"
	"fmt"

	"github.com/jumonmd/gengo"
	"github.com/jumonmd/gengo/chat"
)

// set OPENAI_API_KEY env.
func main() {
	ctx := context.Background()

	// Simple text generation
	resp, err := gengo.Generate(ctx, &chat.Request{
		Model: "gpt-4o-mini", // or gemini-2.0-flash, claude-3-5-haiku-latest
		Messages: []chat.Message{
			chat.NewTextMessage(chat.MessageRoleHuman, "Hello, how are you?"),
		},
	})
	if err != nil {
		panic(err)
	}

	// Print the response
	for _, msg := range resp.Messages {
		fmt.Println(msg.String())
	}
}
