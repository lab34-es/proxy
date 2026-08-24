package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lab34-es/proxy/internal/models"
)

type GuardrailStore struct {
	db *sql.DB
}

func NewGuardrailStore(db *sql.DB) *GuardrailStore {
	return &GuardrailStore{db: db}
}

func (s *GuardrailStore) Create(pattern, mode, replaceBy string) (*models.Guardrail, error) {
	g := &models.Guardrail{
		ID:        uuid.New().String(),
		Pattern:   pattern,
		Mode:      mode,
		ReplaceBy: replaceBy,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(
		`INSERT INTO guardrails (id, pattern, mode, replace_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		g.ID, g.Pattern, g.Mode, g.ReplaceBy, g.CreatedAt, g.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert guardrail: %w", err)
	}
	return g, nil
}

func (s *GuardrailStore) List() ([]models.Guardrail, error) {
	rows, err := s.db.Query(`SELECT id, pattern, mode, replace_by, created_at, updated_at FROM guardrails ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Guardrail
	for rows.Next() {
		var g models.Guardrail
		if err := rows.Scan(&g.ID, &g.Pattern, &g.Mode, &g.ReplaceBy, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *GuardrailStore) GetByID(id string) (*models.Guardrail, error) {
	var g models.Guardrail
	err := s.db.QueryRow(
		`SELECT id, pattern, mode, replace_by, created_at, updated_at FROM guardrails WHERE id = ?`, id,
	).Scan(&g.ID, &g.Pattern, &g.Mode, &g.ReplaceBy, &g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *GuardrailStore) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM guardrails WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
