package database

import (
	"database/sql"
	"fmt"
	"time"

	"go-help-cs/internal/config"

	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

type Ticket struct {
	ID          int64
	Subject     string
	Description string
	Status      int
	Priority    int
	Type        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsSystem    bool
}

type TicketComment struct {
	ID        int64
	TicketID  int64
	Body      string
	IsPrivate bool
	CreatedAt time.Time
}

func New(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return database, nil
}

func (d *Database) createTables() error {
	ticketsTable := `
	CREATE TABLE IF NOT EXISTS tickets (
		id BIGINT PRIMARY KEY,
		subject TEXT NOT NULL,
		description TEXT,
		status INTEGER NOT NULL,
		priority INTEGER NOT NULL,
		type VARCHAR(50),
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		is_system BOOLEAN DEFAULT FALSE,
		processed_at TIMESTAMP
	);`

	commentsTable := `
	CREATE TABLE IF NOT EXISTS ticket_comments (
		id BIGINT PRIMARY KEY,
		ticket_id BIGINT NOT NULL,
		body TEXT NOT NULL,
		is_private BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP NOT NULL,
		FOREIGN KEY (ticket_id) REFERENCES tickets(id)
	);`

	if _, err := d.db.Exec(ticketsTable); err != nil {
		return err
	}

	if _, err := d.db.Exec(commentsTable); err != nil {
		return err
	}

	return nil
}

func (d *Database) SaveTicket(ticket *Ticket) error {
	query := `
	INSERT INTO tickets (id, subject, description, status, priority, type, created_at, updated_at, is_system)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT (id) DO UPDATE SET
		subject = EXCLUDED.subject,
		description = EXCLUDED.description,
		status = EXCLUDED.status,
		priority = EXCLUDED.priority,
		type = EXCLUDED.type,
		updated_at = EXCLUDED.updated_at,
		is_system = EXCLUDED.is_system`

	_, err := d.db.Exec(query, ticket.ID, ticket.Subject, ticket.Description,
		ticket.Status, ticket.Priority, ticket.Type, ticket.CreatedAt, ticket.UpdatedAt, ticket.IsSystem)
	return err
}

func (d *Database) SaveComment(comment *TicketComment) error {
	query := `
	INSERT INTO ticket_comments (id, ticket_id, body, is_private, created_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id) DO NOTHING`

	_, err := d.db.Exec(query, comment.ID, comment.TicketID, comment.Body, comment.IsPrivate, comment.CreatedAt)
	return err
}

func (d *Database) GetTicketByID(id int64) (*Ticket, error) {
	query := `SELECT id, subject, description, status, priority, type, created_at, updated_at, is_system 
			  FROM tickets WHERE id = $1`

	var ticket Ticket
	err := d.db.QueryRow(query, id).Scan(
		&ticket.ID, &ticket.Subject, &ticket.Description, &ticket.Status,
		&ticket.Priority, &ticket.Type, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.IsSystem)

	if err != nil {
		return nil, err
	}

	return &ticket, nil
}

func (d *Database) GetUnprocessedTickets() ([]*Ticket, error) {
	query := `SELECT id, subject, description, status, priority, type, created_at, updated_at, is_system 
			  FROM tickets WHERE processed_at IS NULL ORDER BY created_at ASC`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*Ticket
	for rows.Next() {
		var ticket Ticket
		err := rows.Scan(
			&ticket.ID, &ticket.Subject, &ticket.Description, &ticket.Status,
			&ticket.Priority, &ticket.Type, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.IsSystem)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, &ticket)
	}

	return tickets, nil
}

func (d *Database) MarkTicketProcessed(id int64) error {
	query := `UPDATE tickets SET processed_at = NOW() WHERE id = $1`
	_, err := d.db.Exec(query, id)
	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}
