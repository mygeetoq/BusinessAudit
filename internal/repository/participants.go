package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"business-pulse/internal/models"
)

var ErrAccountHasData = errors.New("счёт уже содержит операции: смена счёта смешала бы данные разных организаций")
var ErrAccountInUse = errors.New("этот счёт уже подключён к другому профилю")

const participantCols = `id, phone, account_id, bank_name, name, group_type, telegram_chat_id, created_at`

func scanParticipant(row pgx.Row) (*models.Participant, error) {
	var p models.Participant
	err := row.Scan(&p.ID, &p.Phone, &p.AccountID, &p.BankName, &p.Name, &p.GroupType, &p.TelegramChatID, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpsertParticipant создаёт участника или обновляет его данные по телефону.
func (r *Repository) UpsertParticipant(ctx context.Context, p models.Participant) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO participants (phone, account_id, name, group_type, bank_name)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'Демо-банк'))
		ON CONFLICT (phone) DO UPDATE SET
			account_id = EXCLUDED.account_id,
			name       = EXCLUDED.name,
			group_type = EXCLUDED.group_type,
			bank_name  = EXCLUDED.bank_name
		RETURNING id`,
		p.Phone, p.AccountID, p.Name, p.GroupType, p.BankName).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert участника %s: %w", p.Phone, err)
	}
	return id, nil
}

func (r *Repository) GetParticipantByID(ctx context.Context, id uuid.UUID) (*models.Participant, error) {
	return scanParticipant(r.pool.QueryRow(ctx,
		`SELECT `+participantCols+` FROM participants WHERE id = $1`, id))
}

func (r *Repository) GetParticipantByPhone(ctx context.Context, phone string) (*models.Participant, error) {
	return scanParticipant(r.pool.QueryRow(ctx,
		`SELECT `+participantCols+` FROM participants WHERE phone = $1`, phone))
}

// EnsureParticipantForPhone создаёт профиль без подключённого счёта.
// Повторный вызов не меняет уже существующий профиль или его данные.
func (r *Repository) EnsureParticipantForPhone(ctx context.Context, phone string) (*models.Participant, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO participants (phone, account_id, bank_name, name, group_type)
		VALUES ($1, 'pending-' || gen_random_uuid()::text, '', '', 'B')
		ON CONFLICT (phone) DO NOTHING`, phone)
	if err != nil {
		return nil, fmt.Errorf("создание участника: %w", err)
	}
	return r.GetParticipantByPhone(ctx, phone)
}

func (r *Repository) GetParticipantByAccountID(ctx context.Context, accountID string) (*models.Participant, error) {
	return scanParticipant(r.pool.QueryRow(ctx,
		`SELECT `+participantCols+` FROM participants WHERE account_id = $1`, accountID))
}

func (r *Repository) GetParticipantByChatID(ctx context.Context, chatID int64) (*models.Participant, error) {
	return scanParticipant(r.pool.QueryRow(ctx,
		`SELECT `+participantCols+` FROM participants WHERE telegram_chat_id = $1`, chatID))
}

func (r *Repository) SetTelegramChatID(ctx context.Context, id uuid.UUID, chatID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE participants SET telegram_chat_id = $2 WHERE id = $1`, id, chatID)
	return err
}

// ConnectBankAccount позволяет впервые указать счёт и после загрузки CSV.
// Замена уже указанного счёта после появления операций запрещена.
func (r *Repository) ConnectBankAccount(ctx context.Context, id uuid.UUID, bankName, accountID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var current, currentBank string
	err = tx.QueryRow(ctx, `SELECT account_id, bank_name FROM participants WHERE id = $1 FOR UPDATE`, id).Scan(&current, &currentBank)
	if err != nil {
		return err
	}
	if currentBank != "" && current != accountID {
		var hasData bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE participant_id=$1) OR EXISTS(SELECT 1 FROM daily_metrics WHERE participant_id=$1)`, id).Scan(&hasData)
		if err != nil {
			return err
		}
		if hasData {
			return ErrAccountHasData
		}
	}
	_, err = tx.Exec(ctx, `UPDATE participants SET bank_name=$2, account_id=$3 WHERE id=$1`, id, bankName, accountID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAccountInUse
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListParticipants(ctx context.Context) ([]models.Participant, error) {
	return r.listParticipants(ctx,
		`SELECT `+participantCols+` FROM participants ORDER BY created_at`)
}

// ListParticipantsByGroup — участники заданной группы (для пересчёта берём 'B').
func (r *Repository) ListParticipantsByGroup(ctx context.Context, group string) ([]models.Participant, error) {
	return r.listParticipants(ctx,
		`SELECT `+participantCols+` FROM participants WHERE group_type = $1 ORDER BY created_at`, group)
}

func (r *Repository) listParticipants(ctx context.Context, sql string, args ...any) ([]models.Participant, error) {
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Participant
	for rows.Next() {
		var p models.Participant
		if err := rows.Scan(&p.ID, &p.Phone, &p.AccountID, &p.BankName, &p.Name, &p.GroupType, &p.TelegramChatID, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
