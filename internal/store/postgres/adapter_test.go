package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplacePlaceholders_NoPlaceholders(t *testing.T) {
	assert.Equal(t, "SELECT 1", replacePlaceholders("SELECT 1"))
}

func TestReplacePlaceholders_SinglePlaceholder(t *testing.T) {
	assert.Equal(t, "SELECT * FROM users WHERE id = $1", replacePlaceholders("SELECT * FROM users WHERE id = ?"))
}

func TestReplacePlaceholders_MultiplePlaceholders(t *testing.T) {
	input := "INSERT INTO users (id, name, email) VALUES (?, ?, ?)"
	expected := "INSERT INTO users (id, name, email) VALUES ($1, $2, $3)"
	assert.Equal(t, expected, replacePlaceholders(input))
}

func TestReplacePlaceholders_QuestionMarkInString(t *testing.T) {
	// ? inside single quotes should NOT be replaced
	input := "SELECT * FROM users WHERE name = 'what?' AND id = ?"
	expected := "SELECT * FROM users WHERE name = 'what?' AND id = $1"
	assert.Equal(t, expected, replacePlaceholders(input))
}

func TestReplacePlaceholders_ComplexQuery(t *testing.T) {
	input := `INSERT INTO auth_sessions (id, user_id, refresh_token_hash, encrypted_refresh, expires_at, created_at)
        VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
	expected := `INSERT INTO auth_sessions (id, user_id, refresh_token_hash, encrypted_refresh, expires_at, created_at)
        VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)`
	assert.Equal(t, expected, replacePlaceholders(input))
}

func TestReplacePlaceholders_SubqueryWithPlaceholders(t *testing.T) {
	input := `SELECT * FROM messages WHERE channel_id = ? AND created_at < (SELECT created_at FROM messages WHERE id = ?) LIMIT ?`
	expected := `SELECT * FROM messages WHERE channel_id = $1 AND created_at < (SELECT created_at FROM messages WHERE id = $2) LIMIT $3`
	assert.Equal(t, expected, replacePlaceholders(input))
}

func TestReplacePlaceholders_EmptyQuery(t *testing.T) {
	assert.Equal(t, "", replacePlaceholders(""))
}

func TestReplacePlaceholders_OnlyPlaceholders(t *testing.T) {
	assert.Equal(t, "$1$2$3", replacePlaceholders("???"))
}

func TestReplacePlaceholders_MixedQuotesAndPlaceholders(t *testing.T) {
	input := "UPDATE servers SET name = ? WHERE invite_code = 'test' AND id = ?"
	expected := "UPDATE servers SET name = $1 WHERE invite_code = 'test' AND id = $2"
	assert.Equal(t, expected, replacePlaceholders(input))
}

func TestNewAdapter(t *testing.T) {
	// Verify NewAdapter doesn't panic with nil (it shouldn't validate at construction time)
	adapter := NewAdapter(nil)
	assert.NotNil(t, adapter)
	assert.Nil(t, adapter.DB())
}
