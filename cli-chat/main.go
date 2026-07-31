package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	LLMURL     = "http://127.0.0.1:8080/v1/chat/completions"
	ModelName  = "./models/Devstral-Small-2-24B"
	ColorReset = "\033[0m"
	ColorUser  = "\033[1;32m" // Яскраво-зелений
	ColorLLM   = "\033[1;36m" // Яскраво-блакитний
	ColorInfo  = "\033[1;33m" // Жовтий
	ColorDim   = "\033[2m"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

type StreamToken struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func sendChatRequest(messages []Message) string {
	request := ChatRequest{
		Model:       ModelName,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.2,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("\nПомилка створення JSON: %v\n", err)
		return ""
	}

	req, err := http.NewRequest("POST", LLMURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("\nПомилка створення HTTP запиту: %v\n", err)
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("\nНе вдалося з'єднатися з локальною LLM (%s): %v\n", LLMURL, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("\nСервер повернув помилку HTTP %d: %s\n", resp.StatusCode, string(body))
		return ""
	}

	var fullResponse strings.Builder
	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("\nПомилка вичитування потоку: %v\n", err)
			break
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var token StreamToken
			if err := json.Unmarshal([]byte(data), &token); err == nil {
				if len(token.Choices) > 0 && token.Choices[0].Delta.Content != "" {
					chunk := token.Choices[0].Delta.Content
					fmt.Print(chunk)
					os.Stdout.Sync() // Одразу виводимо токен на екран!
					fullResponse.WriteString(chunk)
				}
			}
		}
	}

	fmt.Println()
	return fullResponse.String()
}

func main() {
	fmt.Printf("%s==================================================%s\n", ColorInfo, ColorReset)
	fmt.Printf("%s 🚀 Локальний CLI-чат (Devstral 24B via Apple Silicon) %s\n", ColorInfo, ColorReset)
	fmt.Printf("%s Напишіть 'exit' або 'quit' для виходу %s\n", ColorDim, ColorReset)
	fmt.Printf("%s==================================================%s\n\n", ColorInfo, ColorReset)

	scanner := bufio.NewScanner(os.Stdin)
	var messages []Message

	// Системна інструкція
	messages = append(messages, Message{
		Role:    "system",
		Content: "Ти — корисний штучний інтелект асистент. Відповідай ввічливо, точно та українською мовою.",
	})

	for {
		fmt.Printf("%sВи > %s", ColorUser, ColorReset)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			fmt.Printf("%sДо побачення! 👋%s\n", ColorInfo, ColorReset)
			break
		}

		messages = append(messages, Message{
			Role:    "user",
			Content: input,
		})

		fmt.Printf("%sLLM > %s", ColorLLM, ColorReset)
		assistantReply := sendChatRequest(messages)

		if assistantReply != "" {
			messages = append(messages, Message{
				Role:    "assistant",
				Content: assistantReply,
			})
		}
		fmt.Println()
	}
}
