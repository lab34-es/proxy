package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lab34-es/proxy/internal/models"
)

type GuardrailEventStore struct {
	db *sql.DB
}

func NewGuardrailEventStore(db *sql.DB) *GuardrailEventStore {
	return &GuardrailEventStore{db: db}
}

func (s *GuardrailEventStore) Record(guardrailID, apiKeyID, pattern, mode, inputText string) (*models.GuardrailEvent, error) {
	ev := &models.GuardrailEvent{
		ID:          uuid.New().String(),
		GuardrailID: guardrailID,
		APIKeyID:    apiKeyID,
		Pattern:     pattern,
		Mode:        mode,
		InputText:   inputText,
		CreatedAt:   time.Now().UTC(),
	}
	_, err := s.db.Exec(
		`INSERT INTO guardrail_events (id, guardrail_id, api_key_id, pattern, mode, input_text, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.GuardrailID, ev.APIKeyID, ev.Pattern, ev.Mode, ev.InputText, ev.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert guardrail event: %w", err)
	}
	return ev, nil
}

type GuardrailEventQuery struct {
	GuardrailID string
	APIKeyID    string
	Limit       int
	Offset      int
}

type GuardrailEventResult struct {
	Records []models.GuardrailEvent `json:"records"`
	Total   int                     `json:"total"`
}

func (s *GuardrailEventStore) List(q GuardrailEventQuery) (*GuardrailEventResult, error) {
	var where []string
	var args []interface{}

	if q.GuardrailID != "" {
		where = append(where, "guardrail_id = ?")
		args = append(args, q.GuardrailID)
	}
	if q.APIKeyID != "" {
		where = append(where, "api_key_id = ?")
		args = append(args, q.APIKeyID)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + join(where, " AND ")
	}

	var total int
	countQ := "SELECT COUNT(*) FROM guardrail_events" + whereClause
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count guardrail events: %w", err)
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	dataQ := "SELECT id, guardrail_id, api_key_id, pattern, mode, input_text, created_at FROM guardrail_events" +
		whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	dataArgs := append(args, limit, q.Offset)

	rows, err := s.db.Query(dataQ, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.GuardrailEvent
	for rows.Next() {
		var r models.GuardrailEvent
		if err := rows.Scan(&r.ID, &r.GuardrailID, &r.APIKeyID, &r.Pattern, &r.Mode, &r.InputText, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &GuardrailEventResult{Records: records, Total: total}, nil
}

func (s *GuardrailEventStore) GetByID(id string) (*models.GuardrailEvent, error) {
	var ev models.GuardrailEvent
	err := s.db.QueryRow(
		`SELECT id, guardrail_id, api_key_id, pattern, mode, input_text, created_at FROM guardrail_events WHERE id = ?`, id,
	).Scan(&ev.ID, &ev.GuardrailID, &ev.APIKeyID, &ev.Pattern, &ev.Mode, &ev.InputText, &ev.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (s *GuardrailEventStore) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM guardrail_events WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// join is a simple helper to avoid importing strings just for this.
func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += sep + p
	}
	return result
}
