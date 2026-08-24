package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lab34-es/proxy/internal/models"
)

type ProviderStore struct {
	db *sql.DB
}

func NewProviderStore(db *sql.DB) *ProviderStore {
	return &ProviderStore{db: db}
}

func (s *ProviderStore) Create(name, baseURL, apiKey string) (*models.Provider, error) {
	p := &models.Provider{
		ID:        uuid.New().String(),
		Name:      name,
		BaseURL:   baseURL,
		APIKey:    apiKey,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := s.db.Exec(
		`INSERT INTO providers (id, name, base_url, api_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.BaseURL, p.APIKey, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert provider: %w", err)
	}
	return p, nil
}

func (s *ProviderStore) List() ([]models.Provider, error) {
	rows, err := s.db.Query(`SELECT id, name, base_url, api_key, created_at, updated_at FROM providers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Provider
	for rows.Next() {
		var p models.Provider
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProviderStore) GetByID(id string) (*models.Provider, error) {
	var p models.Provider
	err := s.db.QueryRow(
		`SELECT id, name, base_url, api_key, created_at, updated_at FROM providers WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *ProviderStore) Update(id string, name, baseURL, apiKey *string) (*models.Provider, error) {
	p, err := s.GetByID(id)
	if err != nil || p == nil {
		return p, err
	}
	if name != nil {
		p.Name = *name
	}
	if baseURL != nil {
		p.BaseURL = *baseURL
	}
	if apiKey != nil {
		p.APIKey = *apiKey
	}
	p.UpdatedAt = time.Now().UTC()

	_, err = s.db.Exec(
		`UPDATE providers SET name=?, base_url=?, api_key=?, updated_at=? WHERE id=?`,
		p.Name, p.BaseURL, p.APIKey, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update provider: %w", err)
	}
	return p, nil
}

func (s *ProviderStore) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
