package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"swiflow/config"
	"swiflow/model"
)

func StartChat(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)
	messages := []model.Message{}
	client := model.GetClient(&model.LLMConfig{
		Provider: "swiflow", TaskId: "test",

		ApiKey: config.Get("SWIFLOW_API_KEY"),
		ApiUrl: "http://localhost:8080/v1",
	})
	for {
		fmt.Print("Please Input Msg: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if strings.EqualFold(input, "EXIT") {
			break
		}

		messages = append(messages, model.Message{
			Role: "user", Content: strings.TrimSpace(input),
		})
		choices, err := client.Respond("chat", messages)
		if err == nil && len(choices) > 0 {
			fmt.Printf("Result: %s\n", choices[0].Message.Content)
		} else {
			fmt.Printf("Error: %v %d\n", err, len(choices))
			break
		}
	}
}

func StartStream(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)
	messages := []model.Message{}
	client := model.GetClient(&model.LLMConfig{
		Provider: "swiflow", TaskId: "test",

		ApiKey: config.Get("SWIFLOW_API_KEY"),
		ApiUrl: "http://localhost:8080/v1",
	})
	for {
		fmt.Print("Please Input Msg: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if strings.EqualFold(input, "EXIT") {
			break
		}

		messages = append(messages, model.Message{
			Role: "user", Content: strings.TrimSpace(input),
		})
		fmt.Print("Stream: ")
		var choice = new(model.Choice)
		err := client.Stream("chat", messages, func(choices []model.Choice) {
			fmt.Print(choices[0].Message.Content)
			choice.Message.Content += choices[0].Message.Content
		})
		fmt.Printf("\nResult: %s\n", choice.Message.Content)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			break
		}
	}
}
