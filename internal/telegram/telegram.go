package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BotToken string
	ChatID   string
	client   *http.Client
}

type Message struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type SendMessageResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int    `json:"message_id"`
		Text      string `json:"text"`
	} `json:"result"`
}

func New(botToken, chatID string) *Client {
	return &Client{
		BotToken: botToken,
		ChatID:   chatID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SendMessage(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.BotToken)

	message := Message{
		ChatID:    c.ChatID,
		Text:      text,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	var response SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !response.OK {
		return fmt.Errorf("telegram API returned error")
	}

	return nil
}

func (c *Client) SendAlert(ticketID int64, subject, description, customerInfo string) error {
	text := fmt.Sprintf(`🚨 <b>СИСТЕМНОЕ ПРЕДУПРЕЖДЕНИЕ</b>

📋 <b>Тикет:</b> #%d
📝 <b>Тема:</b> %s
📄 <b>Описание:</b> %s
👤 <b>Информация о клиенте:</b> %s

⏰ <b>Время:</b> %s

⚠️ Требует немедленного внимания!`,
		ticketID, subject, description, customerInfo, time.Now().Format("02.01.2006 15:04:05"))

	return c.SendMessage(text)
}

func (c *Client) SendNoOrdersAlert(ticketID int64, customerInfo string, daysWithoutOrders int) error {
	text := fmt.Sprintf(`📊 <b>ПРЕДУПРЕЖДЕНИЕ: НЕТ ЗАКАЗОВ</b>

📋 <b>Тикет:</b> #%d
👤 <b>Клиент:</b> %s
📅 <b>Дней без заказов:</b> %d

⏰ <b>Время:</b> %s

💡 Рекомендуется связаться с клиентом для выяснения причин.`,
		ticketID, customerInfo, daysWithoutOrders, time.Now().Format("02.01.2006 15:04:05"))

	return c.SendMessage(text)
}
