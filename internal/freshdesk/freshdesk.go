package freshdesk

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-help-cs/internal/config"
	"go-help-cs/internal/database"
)

type Client struct {
	config *config.FreshdeskConfig
	client *http.Client
}

type Ticket struct {
	ID          int64     `json:"id"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	Status      int       `json:"status"`
	Priority    int       `json:"priority"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	RequesterID int64     `json:"requester_id"`
	Email       string    `json:"email"`
}

type Comment struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	Body      string    `json:"body"`
	IsPrivate bool      `json:"private"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateCommentRequest struct {
	Body      string `json:"body"`
	IsPrivate bool   `json:"private"`
}

func New(cfg *config.FreshdeskConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetNewTickets() ([]*Ticket, error) {
	url := fmt.Sprintf("%s/tickets", c.config.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tickets []*Ticket
	if err := json.NewDecoder(resp.Body).Decode(&tickets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tickets, nil
}

func (c *Client) GetTicketByID(id int64) (*Ticket, error) {
	url := fmt.Sprintf("%s/tickets/%d", c.config.BaseURL, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var ticket Ticket
	if err := json.NewDecoder(resp.Body).Decode(&ticket); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ticket, nil
}

func (c *Client) CreateComment(ticketID int64, body string, isPrivate bool) error {
	url := fmt.Sprintf("%s/tickets/%d/notes", c.config.BaseURL, ticketID)

	commentReq := CreateCommentRequest{
		Body:      body,
		IsPrivate: isPrivate,
	}

	jsonData, err := json.Marshal(commentReq)
	if err != nil {
		return fmt.Errorf("failed to marshal comment request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) setAuthHeader(req *http.Request) {
	auth := base64.StdEncoding.EncodeToString([]byte(c.config.APIKey + ":X"))
	req.Header.Set("Authorization", "Basic "+auth)
}

func (c *Client) IsSystemTicket(ticket *Ticket, systemEmails []string) bool {
	// Логика определения системного тикета по email
	for _, systemEmail := range systemEmails {
		if ticket.Email == systemEmail ||
			(len(ticket.Email) > len(systemEmail) && ticket.Email[:len(systemEmail)] == systemEmail) {
			return true
		}
	}
	return false
}

func (c *Client) ConvertToDatabaseTicket(ticket *Ticket, systemEmails []string) *database.Ticket {
	return &database.Ticket{
		ID:          ticket.ID,
		Subject:     ticket.Subject,
		Description: ticket.Description,
		Status:      ticket.Status,
		Priority:    ticket.Priority,
		Type:        ticket.Type,
		CreatedAt:   ticket.CreatedAt,
		UpdatedAt:   ticket.UpdatedAt,
		IsSystem:    c.IsSystemTicket(ticket, systemEmails),
	}
}
