package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"ziziphus/pkg/model"
)

type ContactRequestRepo struct {
	pool DBPool
}

func NewContactRequestRepo(pool DBPool) *ContactRequestRepo {
	return &ContactRequestRepo{pool: pool}
}

// Insert creates a new contact request. On conflict it returns a false boolean.
func (r *ContactRequestRepo) Insert(ctx context.Context, req *model.ContactRequest) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO contact_requests (from_user_id, to_user_id, form_msg_id, status, message)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (from_user_id, to_user_id) DO NOTHING
		 RETURNING id`,
		req.FromUserID, req.ToUserID, req.FormMsgID, req.Status, req.Message).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil // conflict, no row inserted
		}
		return 0, err
	}
	return id, nil
}

// InsertTx is the transactional variant of Insert.
func (r *ContactRequestRepo) InsertTx(ctx context.Context, tx pgx.Tx, req *model.ContactRequest) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx,
		`INSERT INTO contact_requests (from_user_id, to_user_id, form_msg_id, status, message)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (from_user_id, to_user_id) DO NOTHING
		 RETURNING id`,
		req.FromUserID, req.ToUserID, req.FormMsgID, req.Status, req.Message).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

// UpdateFormMsgID sets the form message ID after the form has been sent.
func (r *ContactRequestRepo) UpdateFormMsgID(ctx context.Context, id, formMsgID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contact_requests SET form_msg_id = $1, updated_at = NOW()
		 WHERE id = $2`, formMsgID, id)
	return err
}

// UpdateFormMsgIDTx is the transactional variant.
func (r *ContactRequestRepo) UpdateFormMsgIDTx(ctx context.Context, tx pgx.Tx, id, formMsgID int64) error {
	_, err := tx.Exec(ctx,
		`UPDATE contact_requests SET form_msg_id = $1, updated_at = NOW()
		 WHERE id = $2`, formMsgID, id)
	return err
}

// LockByIDTx acquires a SELECT FOR UPDATE row lock and returns the request.
// Must be called within a transaction. Use for serializing concurrent FormResponse handling.
func (r *ContactRequestRepo) LockByIDTx(ctx context.Context, tx pgx.Tx, id int64) (*model.ContactRequest, error) {
	req := &model.ContactRequest{}
	var createdAt, updatedAt time.Time
	err := tx.QueryRow(ctx,
		`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at
		 FROM contact_requests WHERE id = $1 FOR UPDATE`, id).
		Scan(&req.ID, &req.FromUserID, &req.ToUserID, &req.FormMsgID, &req.Status, &req.Message, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	req.CreatedAt = createdAt.UnixMilli()
	req.UpdatedAt = updatedAt.UnixMilli()
	return req, nil
}

// UpdateStatus updates the status (non-transactional).
func (r *ContactRequestRepo) UpdateStatus(ctx context.Context, id int64, status model.ContactRequestStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contact_requests SET status = $1, updated_at = NOW()
		 WHERE id = $2`, status, id)
	return err
}

// UpdateStatusTx updates the status within an existing transaction.
func (r *ContactRequestRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id int64, status model.ContactRequestStatus) error {
	_, err := tx.Exec(ctx,
		`UPDATE contact_requests SET status = $1, updated_at = NOW()
		 WHERE id = $2`, status, id)
	return err
}

// GetByID returns a single contact request by primary key.
func (r *ContactRequestRepo) GetByID(ctx context.Context, id int64) (*model.ContactRequest, error) {
	req := &model.ContactRequest{}
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at
		 FROM contact_requests WHERE id = $1`, id).
		Scan(&req.ID, &req.FromUserID, &req.ToUserID, &req.FormMsgID, &req.Status, &req.Message, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	req.CreatedAt = createdAt.UnixMilli()
	req.UpdatedAt = updatedAt.UnixMilli()
	return req, nil
}

// GetByFormMsgID returns a contact request by the form message ID.
func (r *ContactRequestRepo) GetByFormMsgID(ctx context.Context, formMsgID int64) (*model.ContactRequest, error) {
	req := &model.ContactRequest{}
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at
		 FROM contact_requests WHERE form_msg_id = $1`, formMsgID).
		Scan(&req.ID, &req.FromUserID, &req.ToUserID, &req.FormMsgID, &req.Status, &req.Message, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	req.CreatedAt = createdAt.UnixMilli()
	req.UpdatedAt = updatedAt.UnixMilli()
	return req, nil
}

// GetByPair returns the contact request between two users (any status).
func (r *ContactRequestRepo) GetByPair(ctx context.Context, fromUserID, toUserID string) (*model.ContactRequest, error) {
	req := &model.ContactRequest{}
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at
		 FROM contact_requests WHERE from_user_id = $1 AND to_user_id = $2`, fromUserID, toUserID).
		Scan(&req.ID, &req.FromUserID, &req.ToUserID, &req.FormMsgID, &req.Status, &req.Message, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	req.CreatedAt = createdAt.UnixMilli()
	req.UpdatedAt = updatedAt.UnixMilli()
	return req, nil
}

// ListSent returns requests sent by a user, ordered newest first.
func (r *ContactRequestRepo) ListSent(ctx context.Context, userID string, page, size int) ([]*model.ContactRequest, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contact_requests WHERE from_user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at
		 FROM contact_requests WHERE from_user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var results []*model.ContactRequest
	for rows.Next() {
		req := &model.ContactRequest{}
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&req.ID, &req.FromUserID, &req.ToUserID, &req.FormMsgID, &req.Status, &req.Message, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		req.CreatedAt = createdAt.UnixMilli()
		req.UpdatedAt = updatedAt.UnixMilli()
		results = append(results, req)
	}
	return results, total, nil
}

// ListReceived returns requests received by a user, optionally filtered by status.
// Pass status = -1 to return all statuses.
func (r *ContactRequestRepo) ListReceived(ctx context.Context, userID string, status int, page, size int) ([]*model.ContactRequest, int, error) {
	var total int
	var countArgs []any
	var listArgs []any
	if status >= 0 {
		countArgs = []any{userID, status}
		listArgs = []any{userID, status, size, (page - 1) * size}
		if err := r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM contact_requests WHERE to_user_id = $1 AND status = $2`, countArgs...).Scan(&total); err != nil {
			return nil, 0, err
		}
	} else {
		countArgs = []any{userID}
		listArgs = []any{userID, size, (page - 1) * size}
		if err := r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM contact_requests WHERE to_user_id = $1`, countArgs...).Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at
		 FROM contact_requests WHERE to_user_id = $1`+func() string {
			if status >= 0 {
				return " AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4"
			}
			return " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		}(), listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var results []*model.ContactRequest
	for rows.Next() {
		req := &model.ContactRequest{}
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&req.ID, &req.FromUserID, &req.ToUserID, &req.FormMsgID, &req.Status, &req.Message, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		req.CreatedAt = createdAt.UnixMilli()
		req.UpdatedAt = updatedAt.UnixMilli()
		results = append(results, req)
	}
	return results, total, nil
}

// Delete removes a contact request by ID.
func (r *ContactRequestRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM contact_requests WHERE id = $1`, id)
	return err
}

// Exists checks if either user has the other in their contacts.
func (r *ContactRequestRepo) ExistsAnyDirection(ctx context.Context, userA, userB string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contacts
		 WHERE (user_id = $1 AND contact_id = $2) OR (user_id = $2 AND contact_id = $1)`,
		userA, userB).Scan(&count)
	return count > 0, err
}
