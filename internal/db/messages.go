package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Message represents a single chat message in a session.
//
// OpenCode stores chat content in two tables:
//   - `message`: one row per message, holds role, agent, model,
//     tokens, cost, time, etc. (in JSON `data` column).
//   - `part`: one or more rows per message, holds the actual content
//     (type=text, step-start, tool, reasoning, etc.).
//
// This struct combines the two for convenient display in the TUI
// chat preview pane.
type Message struct {
	ID          string
	SessionID   string
	Role        string // "user" or "assistant"
	TimeCreated int64
	TimeCompleted int64
	Agent       string
	ModelID     string
	ProviderID  string
	Parts       []MessagePart
}

// MessagePart is one content chunk within a message.
type MessagePart struct {
	Type string // "text", "reasoning", "step-start", "tool", etc.
	Text string // populated for type=text and type=reasoning
}

// ListMessages returns up to `limit` most recent messages for the
// given session, with their parts attached. Messages are ordered
// oldest-first so the TUI can render them top-down in chronological
// order. The default limit of 0 means "no limit".
func ListMessages(ctx context.Context, database *sql.DB, sessionID string, limit int) ([]Message, error) {
	const msgCols = `id, session_id, data, time_created, time_updated`
	q := `SELECT ` + msgCols + ` FROM message WHERE session_id = ? ORDER BY time_created ASC`
	if limit > 0 {
		q += ` LIMIT ?`
	}

	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = database.QueryContext(ctx, q, sessionID, limit)
	} else {
		rows, err = database.QueryContext(ctx, q, sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var (
			m            Message
			data         string
			timeCreated  int64
			timeUpdated  int64
		)
		if err := rows.Scan(&m.ID, &m.SessionID, &data, &timeCreated, &timeUpdated); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.TimeCreated = timeCreated
		m.TimeCompleted = timeUpdated
		m.Role = jsonStringField(data, "role")
		m.Agent = jsonStringField(data, "agent")
		m.ModelID = jsonStringField(data, "modelID")
		m.ProviderID = jsonStringField(data, "providerID")
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	// Attach parts for each message. One query per message is
	// acceptable here because the typical preview window is small
	// (a few dozen messages at most).
	for i := range messages {
		parts, err := listParts(ctx, database, messages[i].ID)
		if err != nil {
			return nil, err
		}
		messages[i].Parts = parts
	}
	return messages, nil
}

func listParts(ctx context.Context, database *sql.DB, messageID string) ([]MessagePart, error) {
	rows, err := database.QueryContext(ctx,
		`SELECT data FROM part WHERE message_id = ? ORDER BY id ASC`, messageID)
	if err != nil {
		return nil, fmt.Errorf("query parts: %w", err)
	}
	defer rows.Close()

	var parts []MessagePart
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scan part: %w", err)
		}
		p := MessagePart{
			Type: jsonStringField(data, "type"),
			Text: jsonStringField(data, "text"),
		}
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

// jsonStringField extracts a top-level string field from a JSON
// object without depending on encoding/json (which would import
// the whole standard library JSON machinery). It returns "" if the
// field is absent, null, or non-string.
//
// The format we expect is `{"field":"value", ...}` with possible
// nested objects. We scan for the field name and read the next
// quoted string value. This is good enough for the opencode.db
// schema which has well-defined string fields.
func jsonStringField(raw, field string) string {
	key := `"` + field + `":`
	i := indexOf(raw, key)
	if i < 0 {
		return ""
	}
	i += len(key)
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t') {
		i++
	}
	if i >= len(raw) || raw[i] != '"' {
		return ""
	}
	i++
	j := i
	for j < len(raw) && raw[j] != '"' {
		if raw[j] == '\\' && j+1 < len(raw) {
			j += 2
			continue
		}
		j++
	}
	if j >= len(raw) {
		return ""
	}
	return raw[i:j]
}

// indexOf is a tiny case-sensitive substring search.
func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
