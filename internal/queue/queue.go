package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-help-cs/internal/config"

	"github.com/streadway/amqp"
)

type Client struct {
	config     *config.RabbitMQConfig
	connection *amqp.Connection
	channel    *amqp.Channel
}

type TicketMessage struct {
	TicketID    int64  `json:"ticket_id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
	Timestamp   int64  `json:"timestamp"`
}

func New(cfg *config.RabbitMQConfig) (*Client, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	client := &Client{
		config:     cfg,
		connection: conn,
		channel:    ch,
	}

	if err := client.setupQueue(); err != nil {
		return nil, fmt.Errorf("failed to setup queue: %w", err)
	}

	return client, nil
}

func (c *Client) setupQueue() error {
	// Объявляем очередь для тикетов
	_, err := c.channel.QueueDeclare(
		c.config.QueueName, // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Устанавливаем QoS для ограничения количества сообщений в обработке
	err = c.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	return nil
}

func (c *Client) PublishTicket(ticket *TicketMessage) error {
	body, err := json.Marshal(ticket)
	if err != nil {
		return fmt.Errorf("failed to marshal ticket message: %w", err)
	}

	err = c.channel.Publish(
		"",                 // exchange
		c.config.QueueName, // routing key
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published ticket %d to queue", ticket.TicketID)
	return nil
}

func (c *Client) ConsumeTickets(handler func(*TicketMessage) error) error {
	msgs, err := c.channel.Consume(
		c.config.QueueName, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Printf("Started consuming tickets from queue: %s", c.config.QueueName)

	go func() {
		for msg := range msgs {
			var ticketMsg TicketMessage
			if err := json.Unmarshal(msg.Body, &ticketMsg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				msg.Nack(false, false)
				continue
			}

			log.Printf("Processing ticket %d", ticketMsg.TicketID)

			if err := handler(&ticketMsg); err != nil {
				log.Printf("Failed to process ticket %d: %v", ticketMsg.TicketID, err)
				msg.Nack(false, true) // requeue
			} else {
				msg.Ack(false)
				log.Printf("Successfully processed ticket %d", ticketMsg.TicketID)
			}
		}
	}()

	return nil
}

func (c *Client) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return fmt.Errorf("failed to close channel: %w", err)
		}
	}

	if c.connection != nil {
		if err := c.connection.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	return nil
}

func NewTicketMessage(ticketID int64, subject, description string, isSystem bool) *TicketMessage {
	return &TicketMessage{
		TicketID:    ticketID,
		Subject:     subject,
		Description: description,
		IsSystem:    isSystem,
		Timestamp:   time.Now().Unix(),
	}
}
