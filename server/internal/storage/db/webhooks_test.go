package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"ziziphus/pkg/model"
)

func TestNewWebhookRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	if repo == nil {
		t.Fatal("NewWebhookRepo returned nil")
	}
}

func TestWebhookRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	now := time.Now()

	headers := []model.WebhookHeader{{Key: "X-Custom", Value: "val"}}
	cidr := []string{"10.0.0.0/8"}

	rows := pgxmock.NewRows([]string{"id", "conv_id", "name", "api_key_plain", "api_key_hash", "callback_url", "headers", "cidr_whitelist", "created_by", "created_at"}).
		AddRow(int64(1), "conv_1", "my-webhook", "key-plain", "key-hash", "https://example.com/cb", headers, cidr, "u1", now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO conv_webhooks (conv_id, name, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, created_by, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING `+scanCols,
	)).
		WithArgs("conv_1", "my-webhook", "key-plain", "key-hash", "https://example.com/cb", headers, cidr, "u1", AnyTime{}).
		WillReturnRows(rows)

	wh := &model.ConvWebhook{
		ConvID:        "conv_1",
		Name:          "my-webhook",
		APIKeyPlain:   "key-plain",
		APIKeyHash:    "key-hash",
		CallbackURL:   "https://example.com/cb",
		Headers:       headers,
		CIDRWhitelist: cidr,
		CreatedBy:     "u1",
	}
	created, err := repo.Create(context.Background(), wh)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created == nil {
		t.Fatal("Create returned nil")
	}
	if created.ID != 1 {
		t.Errorf("ID = %d, want 1", created.ID)
	}
	if created.Name != "my-webhook" {
		t.Errorf("Name = %q, want %q", created.Name, "my-webhook")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestWebhookRepo_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "conv_id", "name", "api_key_plain", "api_key_hash", "callback_url", "headers", "cidr_whitelist", "created_by", "created_at"}).
		AddRow(int64(1), "conv_1", "my-webhook", "key-plain", "key-hash", "https://example.com/cb", []model.WebhookHeader{}, []string{}, "u1", now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT ` + scanCols + ` FROM conv_webhooks WHERE id = $1`,
	)).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	wh, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if wh == nil {
		t.Fatal("GetByID returned nil")
	}
	if wh.ID != 1 {
		t.Errorf("ID = %d, want 1", wh.ID)
	}
	if wh.ConvID != "conv_1" {
		t.Errorf("ConvID = %q, want %q", wh.ConvID, "conv_1")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestWebhookRepo_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT ` + scanCols + ` FROM conv_webhooks WHERE id = $1`,
	)).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	wh, err := repo.GetByID(context.Background(), 999)
	if err == nil {
		t.Fatal("GetByID expected error, got nil")
	}
	if wh != nil {
		t.Errorf("GetByID expected nil, got %v", wh)
	}
}

func TestWebhookRepo_GetByAPIKey(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "conv_id", "name", "api_key_plain", "api_key_hash", "callback_url", "headers", "cidr_whitelist", "created_by", "created_at"}).
		AddRow(int64(1), "conv_1", "my-webhook", "test-api-key", "hash", "", []model.WebhookHeader{}, []string{}, "u1", now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT ` + scanCols + ` FROM conv_webhooks WHERE api_key_plain = $1`,
	)).
		WithArgs("test-api-key").
		WillReturnRows(rows)

	wh, err := repo.GetByAPIKey(context.Background(), "test-api-key")
	if err != nil {
		t.Fatalf("GetByAPIKey: %v", err)
	}
	if wh.APIKeyPlain != "test-api-key" {
		t.Errorf("APIKeyPlain = %q, want %q", wh.APIKeyPlain, "test-api-key")
	}
}

func TestWebhookRepo_ListByConvID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "conv_id", "name", "api_key_plain", "api_key_hash", "callback_url", "headers", "cidr_whitelist", "created_by", "created_at"}).
		AddRow(int64(1), "conv_1", "wh1", "k1", "h1", "", []model.WebhookHeader{}, []string{}, "u1", now).
		AddRow(int64(2), "conv_1", "wh2", "k2", "h2", "", []model.WebhookHeader{}, []string{}, "u2", now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT ` + scanCols + ` FROM conv_webhooks WHERE conv_id = $1 ORDER BY created_at ASC`,
	)).
		WithArgs("conv_1").
		WillReturnRows(rows)

	list, err := repo.ListByConvID(context.Background(), "conv_1")
	if err != nil {
		t.Fatalf("ListByConvID: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d webhooks, want 2", len(list))
	}
	if list[0].Name != "wh1" {
		t.Errorf("list[0].Name = %q, want %q", list[0].Name, "wh1")
	}
	if list[1].Name != "wh2" {
		t.Errorf("list[1].Name = %q, want %q", list[1].Name, "wh2")
	}
}

func TestWebhookRepo_ListByConvID_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)

	rows := pgxmock.NewRows([]string{"id", "conv_id", "name", "api_key_plain", "api_key_hash", "callback_url", "headers", "cidr_whitelist", "created_by", "created_at"})
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT ` + scanCols + ` FROM conv_webhooks WHERE conv_id = $1 ORDER BY created_at ASC`,
	)).
		WithArgs("conv_empty").
		WillReturnRows(rows)

	list, err := repo.ListByConvID(context.Background(), "conv_empty")
	if err != nil {
		t.Fatalf("ListByConvID: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d webhooks, want 0", len(list))
	}
}

func TestWebhookRepo_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE conv_webhooks SET name = $1, callback_url = $2, headers = $3, cidr_whitelist = $4 WHERE id = $5`,
	)).
		WithArgs("newname", "https://new-cb.com", []model.WebhookHeader{}, []string{}, int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	wh := &model.ConvWebhook{
		ID:            1,
		Name:          "newname",
		CallbackURL:   "https://new-cb.com",
		Headers:       []model.WebhookHeader{},
		CIDRWhitelist: []string{},
	}
	err = repo.Update(context.Background(), wh)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestWebhookRepo_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM conv_webhooks WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestWebhookRepo_GetByConvIDAndName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "conv_id", "name", "api_key_plain", "api_key_hash", "callback_url", "headers", "cidr_whitelist", "created_by", "created_at"}).
		AddRow(int64(1), "conv_1", "my-webhook", "key", "hash", "", []model.WebhookHeader{}, []string{}, "u1", now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT `+scanCols+` FROM conv_webhooks WHERE conv_id = $1 AND name = $2`,
	)).
		WithArgs("conv_1", "my-webhook").
		WillReturnRows(rows)

	wh, err := repo.GetByConvIDAndName(context.Background(), "conv_1", "my-webhook")
	if err != nil {
		t.Fatalf("GetByConvIDAndName: %v", err)
	}
	if wh.ID != 1 {
		t.Errorf("ID = %d, want 1", wh.ID)
	}
	if wh.Name != "my-webhook" {
		t.Errorf("Name = %q, want %q", wh.Name, "my-webhook")
	}
}

func TestWebhookRepo_GetAPIKeyHash(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT api_key_hash FROM conv_webhooks WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"api_key_hash"}).AddRow("hash-value"))

	hash, err := repo.GetAPIKeyHash(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetAPIKeyHash: %v", err)
	}
	if hash != "hash-value" {
		t.Errorf("hash = %q, want %q", hash, "hash-value")
	}
}

func TestWebhookRepo_UpdateAPIKeyHash(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conv_webhooks SET api_key_hash = $1 WHERE id = $2`)).
		WithArgs("new-hash", int64(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateAPIKeyHash(context.Background(), 1, "new-hash")
	if err != nil {
		t.Fatalf("UpdateAPIKeyHash: %v", err)
	}
}

func TestScanWebhookHeaders_Nil(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	headers, err := repo.scanWebhookHeaders(nil)
	if err != nil {
		t.Fatalf("scanWebhookHeaders(nil): %v", err)
	}
	if headers != nil {
		t.Errorf("expected nil, got %+v", headers)
	}
}

func TestScanWebhookHeaders_Valid(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	src := []byte(`[{"key":"X-Custom","value":"val1"},{"key":"Authorization","value":"Bearer tok"}]`)
	headers, err := repo.scanWebhookHeaders(src)
	if err != nil {
		t.Fatalf("scanWebhookHeaders: %v", err)
	}
	if len(headers) != 2 {
		t.Fatalf("got %d headers, want 2", len(headers))
	}
	if headers[0].Key != "X-Custom" || headers[0].Value != "val1" {
		t.Errorf("header[0] = %+v", headers[0])
	}
	if headers[1].Key != "Authorization" || headers[1].Value != "Bearer tok" {
		t.Errorf("header[1] = %+v", headers[1])
	}
}

func TestScanWebhookHeaders_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	src := []byte(`not json`)
	_, err = repo.scanWebhookHeaders(src)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestScanWebhookHeaders_NonBytes(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	headers, err := repo.scanWebhookHeaders("string type")
	if err != nil {
		t.Fatalf("scanWebhookHeaders(string): %v", err)
	}
	if headers != nil {
		t.Errorf("expected nil for non-[]byte src, got %+v", headers)
	}
}

func TestScanStringSlice_Nil(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	s, err := repo.scanStringSlice(nil)
	if err != nil {
		t.Fatalf("scanStringSlice(nil): %v", err)
	}
	if s != nil {
		t.Errorf("expected nil, got %+v", s)
	}
}

func TestScanStringSlice_Valid(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	src := []byte(`["10.0.0.0/8","192.168.0.0/16"]`)
	s, err := repo.scanStringSlice(src)
	if err != nil {
		t.Fatalf("scanStringSlice: %v", err)
	}
	if len(s) != 2 {
		t.Fatalf("got %d items, want 2", len(s))
	}
	if s[0] != "10.0.0.0/8" || s[1] != "192.168.0.0/16" {
		t.Errorf("got %+v", s)
	}
}

func TestScanStringSlice_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	src := []byte(`[invalid`)
	_, err = repo.scanStringSlice(src)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestScanStringSlice_NonBytes(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewWebhookRepo(mock)
	s, err := repo.scanStringSlice(42)
	if err != nil {
		t.Fatalf("scanStringSlice(int): %v", err)
	}
	if s != nil {
		t.Errorf("expected nil for non-[]byte src, got %+v", s)
	}
}
