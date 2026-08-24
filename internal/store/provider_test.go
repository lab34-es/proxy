package store

import (
	"database/sql"
	"testing"

	"github.com/lab34-es/proxy/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	return database
}

func TestProviderStore_Create(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	p, err := s.Create("openai", "https://api.openai.com", "sk-test")
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)
	assert.Equal(t, "openai", p.Name)
	assert.Equal(t, "https://api.openai.com", p.BaseURL)
	assert.Equal(t, "sk-test", p.APIKey)
	assert.False(t, p.CreatedAt.IsZero())
	assert.False(t, p.UpdatedAt.IsZero())
}

func TestProviderStore_Create_DuplicateName(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	_, err := s.Create("openai", "https://api.openai.com", "sk-1")
	require.NoError(t, err)

	_, err = s.Create("openai", "https://api.openai.com", "sk-2")
	assert.Error(t, err) // UNIQUE constraint on name
}

func TestProviderStore_List(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	// Empty list.
	list, err := s.List()
	require.NoError(t, err)
	assert.Empty(t, list)

	// Add providers.
	_, err = s.Create("provider1", "https://p1.com", "key1")
	require.NoError(t, err)
	_, err = s.Create("provider2", "https://p2.com", "key2")
	require.NoError(t, err)

	list, err = s.List()
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestProviderStore_GetByID(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	created, err := s.Create("test", "https://test.com", "key")
	require.NoError(t, err)

	p, err := s.GetByID(created.ID)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, created.ID, p.ID)
	assert.Equal(t, "test", p.Name)
}

func TestProviderStore_GetByID_NotFound(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	p, err := s.GetByID("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestProviderStore_Update(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	created, err := s.Create("original", "https://orig.com", "key1")
	require.NoError(t, err)

	newName := "updated"
	newURL := "https://updated.com"
	newKey := "key2"

	updated, err := s.Update(created.ID, &newName, &newURL, &newKey)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "updated", updated.Name)
	assert.Equal(t, "https://updated.com", updated.BaseURL)
	assert.Equal(t, "key2", updated.APIKey)
}

func TestProviderStore_Update_PartialFields(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	created, err := s.Create("original", "https://orig.com", "key1")
	require.NoError(t, err)

	newName := "updated"
	updated, err := s.Update(created.ID, &newName, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "updated", updated.Name)
	assert.Equal(t, "https://orig.com", updated.BaseURL) // unchanged
	assert.Equal(t, "key1", updated.APIKey)              // unchanged
}

func TestProviderStore_Update_NotFound(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	name := "x"
	p, err := s.Update("nonexistent", &name, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestProviderStore_Delete(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	created, err := s.Create("todelete", "https://del.com", "key")
	require.NoError(t, err)

	err = s.Delete(created.ID)
	assert.NoError(t, err)

	// Verify deleted.
	p, err := s.GetByID(created.ID)
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestProviderStore_Delete_NotFound(t *testing.T) {
	database := setupTestDB(t)
	s := NewProviderStore(database)

	err := s.Delete("nonexistent")
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestProviderStore_Delete_CascadesUsageAndEvents(t *testing.T) {
	database := setupTestDB(t)
	ps := NewProviderStore(database)
	ks := NewAPIKeyStore(database)
	us := NewUsageStore(database)
	gs := NewGuardrailStore(database)
	ges := NewGuardrailEventStore(database)

	// Create a provider, key, usage record, and guardrail event.
	p, err := ps.Create("cascade-test", "https://cascade.com", "key")
	require.NoError(t, err)

	k, _, err := ks.Create("test-key", p.ID, 60)
	require.NoError(t, err)

	err = us.Record(k.ID, p.ID, "gpt-4", 100, 50, 150)
	require.NoError(t, err)

	g, err := gs.Create("secret-\\d+", "reject", "")
	require.NoError(t, err)

	_, err = ges.Record(g.ID, k.ID, "secret-\\d+", "reject", "secret-123")
	require.NoError(t, err)

	// Deleting the provider should succeed (cascade: provider → api_keys → usage + guardrail_events).
	err = ps.Delete(p.ID)
	assert.NoError(t, err)

	// Verify provider is gone.
	got, err := ps.GetByID(p.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}
