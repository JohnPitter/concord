package sqlite

import (
	"context"
	"fmt"
)

// P2PMessage representa uma mensagem P2P persistida localmente.
type P2PMessage struct {
	ID        string `json:"id"`
	PeerID    string `json:"peer_id"`
	Direction string `json:"direction"` // "sent" | "received"
	Content   string `json:"content"`
	SentAt    string `json:"sent_at"`
}

// P2PRepo implementa persistência de mensagens P2P.
type P2PRepo struct {
	db *DB
}

// NewP2PRepo cria um novo repositório de mensagens P2P.
func NewP2PRepo(db *DB) *P2PRepo {
	return &P2PRepo{db: db}
}

// SaveMessage persiste uma mensagem P2P.
// Complexity: O(1).
func (r *P2PRepo) SaveMessage(ctx context.Context, msg P2PMessage) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO p2p_messages (id, peer_id, direction, content, sent_at)
		 VALUES (?, ?, ?, ?, ?)`,
		msg.ID, msg.PeerID, msg.Direction, msg.Content, msg.SentAt,
	)
	if err != nil {
		return fmt.Errorf("p2p_repo: save message: %w", err)
	}
	return nil
}

// GetMessages retorna mensagens com um peer ordenadas por sent_at ASC.
// Complexity: O(n) onde n = limit.
func (r *P2PRepo) GetMessages(ctx context.Context, peerID string, limit int) ([]P2PMessage, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, peer_id, direction, content, sent_at
		 FROM p2p_messages
		 WHERE peer_id = ?
		 ORDER BY sent_at ASC
		 LIMIT ?`,
		peerID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("p2p_repo: get messages: %w", err)
	}
	defer rows.Close()

	var msgs []P2PMessage
	for rows.Next() {
		var m P2PMessage
		if err := rows.Scan(&m.ID, &m.PeerID, &m.Direction, &m.Content, &m.SentAt); err != nil {
			return nil, fmt.Errorf("p2p_repo: scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("p2p_repo: rows: %w", err)
	}
	if msgs == nil {
		msgs = []P2PMessage{}
	}
	return msgs, nil
}
