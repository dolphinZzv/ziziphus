package db

import (
	"context"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"ziziphus/pkg/model"
)

func TestMFARepo_Upsert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMFARepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO user_mfa (user_id, mfa_type, enabled, secret, updated_at) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (user_id) DO UPDATE SET mfa_type = $2, enabled = $3, secret = $4, updated_at = NOW()`)).
		WithArgs("user_1", model.MFATOTP, true, "testsecret").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	m := &model.UserMFA{
		UserID:  "user_1",
		MFAType: model.MFATOTP,
		Enabled: true,
		Secret:  "testsecret",
	}
	err = repo.Upsert(context.Background(), m)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMFARepo_Disable(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMFARepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_mfa SET enabled = FALSE, updated_at = NOW() WHERE user_id = $1`)).
		WithArgs("user_1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Disable(context.Background(), "user_1")
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
