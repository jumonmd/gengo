package main

import (
	"context"
	"fmt"

	"github.com/jumonmd/gengo"
	"github.com/jumonmd/gengo/chat"
)

// set GOOGLE_API_KEY env.
func main() {
	ctx := context.Background()

	msgs := []chat.Message{
		chat.NewTextMessage(chat.MessageRoleHuman, "今年の東京の桜の開花日は何日でしたか？"),
	}

	// Simple text generation
	resp, err := gengo.Generate(ctx, &chat.Request{
		Model:    "gemini-2.0-flash",
		Messages: msgs,
	}, chat.WithSearch())
	if err != nil {
		panic(err)
	}

	// Print the response
	for _, msg := range resp.Messages {
		fmt.Printf("%+v\n", msg.ContentString())
	}
}
