package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lab34-es/proxy/internal/models"
)

type APIKeyStore struct {
	db *sql.DB
}

func NewAPIKeyStore(db *sql.DB) *APIKeyStore {
	return &APIKeyStore{db: db}
}

// Create generates a new proxy API key. It returns the model (with hash) and
// the raw key string. The raw key is never stored.
func (s *APIKeyStore) Create(name, providerID string, rateLimitRPM int) (*models.APIKey, string, error) {
	rawKey, err := generateKey()
	if err != nil {
		return nil, "", fmt.Errorf("generate key: %w", err)
	}
	hash := hashKey(rawKey)

	k := &models.APIKey{
		ID:           uuid.New().String(),
		Name:         name,
		KeyHash:      hash,
		ProviderID:   providerID,
		RateLimitRPM: rateLimitRPM,
		CreatedAt:    time.Now().UTC(),
	}

	_, err = s.db.Exec(
		`INSERT INTO api_keys (id, name, key_hash, provider_id, rate_limit_rpm, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		k.ID, k.Name, k.KeyHash, k.ProviderID, k.RateLimitRPM, k.CreatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("insert api key: %w", err)
	}
	return k, rawKey, nil
}

func (s *APIKeyStore) List() ([]models.APIKey, error) {
	rows, err := s.db.Query(
		`SELECT id, name, key_hash, provider_id, rate_limit_rpm, created_at, revoked_at FROM api_keys ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.APIKey
	for rows.Next() {
		var k models.APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.ProviderID, &k.RateLimitRPM, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// Lookup finds a non-revoked API key by its raw key string.
func (s *APIKeyStore) Lookup(rawKey string) (*models.APIKey, error) {
	hash := hashKey(rawKey)
	var k models.APIKey
	err := s.db.QueryRow(
		`SELECT id, name, key_hash, provider_id, rate_limit_rpm, created_at, revoked_at
		   FROM api_keys WHERE key_hash = ? AND revoked_at IS NULL`, hash,
	).Scan(&k.ID, &k.Name, &k.KeyHash, &k.ProviderID, &k.RateLimitRPM, &k.CreatedAt, &k.RevokedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *APIKeyStore) Revoke(id string) error {
	now := time.Now().UTC()
	res, err := s.db.Exec(`UPDATE api_keys SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, now, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// generateKey creates a "llmp-" prefixed random key.
func generateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "llmp-" + hex.EncodeToString(b), nil
}

func hashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
