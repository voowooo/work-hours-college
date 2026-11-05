package graph

import (
	"context"

	"github.com/workhourscounter/backend/internal/db"
	"github.com/workhourscounter/backend/pkg/auth"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	DB *db.DB
}

// ensureUser создаёт пользователя при первом обращении.
func (r *Resolver) ensureUser(ctx context.Context, telegramID int64) error {
	username := auth.GetTelegramUsername(ctx)
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO users (telegram_id, username)
		VALUES ($1, NULLIF($2, ''))
		ON CONFLICT (telegram_id) DO UPDATE SET username = COALESCE(NULLIF($2, ''), users.username)
	`, telegramID, username)
	return err
}
