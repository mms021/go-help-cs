package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-help-cs/internal/config"
)

type Client struct {
	config *config.OllamaConfig
	client *http.Client
}

type GenerateRequest struct {
	Model   string  `json:"model"`
	Prompt  string  `json:"prompt"`
	Stream  bool    `json:"stream"`
	Options Options `json:"options,omitempty"`
}

type Options struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	MaxTokens   int     `json:"num_predict,omitempty"`
}

type GenerateResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

func New(cfg *config.OllamaConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) GenerateResponse(prompt string) (string, error) {
	return c.GenerateResponseWithModel(prompt, c.config.ModelCS)
}

func (c *Client) GenerateResponseWithModel(prompt, model string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", c.config.BaseURL)

	req := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: Options{
			Temperature: 0.7,
			TopP:        0.9,
			MaxTokens:   1000,
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Response, nil
}

func (c *Client) GenerateTicketResponse(ticketSubject, ticketDescription string, isSystem bool, modelCS, modelSupport string) (string, error) {
	var model string

	if isSystem {
		model = modelSupport
		prompt := c.buildSystemTicketPrompt(ticketSubject, ticketDescription)
		return c.GenerateResponseWithModel(prompt, model)
	} else {
		model = modelCS
		prompt := c.buildCustomerTicketPrompt(ticketSubject, ticketDescription)
		return c.GenerateResponseWithModel(prompt, model)
	}
}

func (c *Client) buildCustomerTicketPrompt(subject, description string) string {
	return fmt.Sprintf(`Ты - помощник службы поддержки. Ответь на тикет клиента профессионально и дружелюбно.

Тема тикета: %s
Описание: %s

Сгенерируй ответ, который:
1. Показывает понимание проблемы клиента
2. Предлагает конкретные шаги для решения
3. Написан профессионально, но дружелюбно
4. Не превышает 200-300 слов

Ответ:`, subject, description)
}

func (c *Client) buildSystemTicketPrompt(subject, description string) string {
	return fmt.Sprintf(`Ты - системный администратор. Обработай системный тикет.

Тема: %s
Описание: %s

Сгенерируй технический ответ, который:
1. Анализирует системную проблему
2. Предлагает технические решения
3. Включает возможные причины и диагностику
4. Написан для технических специалистов

Ответ:`, subject, description)
}
