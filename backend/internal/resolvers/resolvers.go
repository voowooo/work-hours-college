package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/workhourscounter/backend/internal/db"
	"github.com/workhourscounter/backend/internal/models"
)

type Resolver struct {
	DB *db.DB
}

func (r *Resolver) Me(ctx context.Context) (*models.User, error) {
	telegramID, ok := ctx.Value("telegramID").(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	user := &models.User{}
	err := r.DB.QueryRowContext(ctx,
		"SELECT id, telegram_id, username, created_at FROM users WHERE telegram_id = $1",
		telegramID,
	).Scan(&user.ID, &user.TelegramID, &user.Username, &user.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

func (r *Resolver) WorkEntry(ctx context.Context, date string) (*models.WorkEntry, error) {
	telegramID, ok := ctx.Value("telegramID").(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	entry := &models.WorkEntry{}
	err := r.DB.QueryRowContext(ctx, `
		SELECT we.id, we.user_id, we.date::text, we.arrived_at::text, we.left_at::text, 
		       COALESCE(we.hours_worked, 0), we.created_at, we.updated_at
		FROM work_entries we
		JOIN users u ON u.id = we.user_id
		WHERE u.telegram_id = $1 AND we.date = $2
	`, telegramID, date).Scan(
		&entry.ID, &entry.UserID, &entry.Date, &entry.ArrivedAt, &entry.LeftAt,
		&entry.HoursWorked, &entry.CreatedAt, &entry.UpdatedAt,
	)

	if err != nil {
		return nil, nil
	}

	return entry, nil
}

func (r *Resolver) WorkEntries(ctx context.Context, startDate, endDate string) ([]models.WorkEntry, error) {
	telegramID, ok := ctx.Value("telegramID").(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT we.id, we.user_id, we.date::text, we.arrived_at::text, we.left_at::text, 
		       COALESCE(we.hours_worked, 0), we.created_at, we.updated_at
		FROM work_entries we
		JOIN users u ON u.id = we.user_id
		WHERE u.telegram_id = $1 AND we.date BETWEEN $2 AND $3
		ORDER BY we.date
	`, telegramID, startDate, endDate)

	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []models.WorkEntry
	for rows.Next() {
		var entry models.WorkEntry
		if err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Date, &entry.ArrivedAt, &entry.LeftAt,
			&entry.HoursWorked, &entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (r *Resolver) MonthStats(ctx context.Context, year, month int) (*models.MonthStats, error) {
	_, ok := ctx.Value("telegramID").(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	startDate := fmt.Sprintf("%d-%02d-01", year, month)
	endDate := fmt.Sprintf("%d-%02d-31", year, month)

	entries, err := r.WorkEntries(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	var totalHours float64
	for _, entry := range entries {
		totalHours += entry.HoursWorked
	}

	monthName := time.Month(month).String()

	return &models.MonthStats{
		Month:      monthName,
		TotalHours: totalHours,
		Entries:    entries,
	}, nil
}

func (r *Resolver) UpsertWorkEntry(ctx context.Context, input models.UpsertWorkEntryInput) (*models.WorkEntry, error) {
	telegramID, ok := ctx.Value("telegramID").(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	entry := &models.WorkEntry{}
	err := r.DB.QueryRowContext(ctx, `
		INSERT INTO work_entries (user_id, date, arrived_at, left_at)
		SELECT u.id, $2, $3, $4
		FROM users u WHERE u.telegram_id = $1
		ON CONFLICT (user_id, date) 
		DO UPDATE SET arrived_at = COALESCE(EXCLUDED.arrived_at, work_entries.arrived_at),
		              left_at = COALESCE(EXCLUDED.left_at, work_entries.left_at),
		              updated_at = NOW()
		RETURNING id, user_id, date::text, arrived_at::text, left_at::text, 
		          COALESCE(hours_worked, 0), created_at, updated_at
	`, telegramID, input.Date, input.ArrivedAt, input.LeftAt).Scan(
		&entry.ID, &entry.UserID, &entry.Date, &entry.ArrivedAt, &entry.LeftAt,
		&entry.HoursWorked, &entry.CreatedAt, &entry.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert entry: %w", err)
	}

	return entry, nil
}

func (r *Resolver) DeleteWorkEntry(ctx context.Context, date string) (bool, error) {
	telegramID, ok := ctx.Value("telegramID").(int64)
	if !ok {
		return false, fmt.Errorf("unauthorized")
	}

	result, err := r.DB.ExecContext(ctx, `
		DELETE FROM work_entries 
		WHERE user_id = (SELECT id FROM users WHERE telegram_id = $1) 
		AND date = $2
	`, telegramID, date)

	if err != nil {
		return false, fmt.Errorf("failed to delete entry: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}
