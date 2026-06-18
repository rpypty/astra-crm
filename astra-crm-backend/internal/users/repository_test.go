package users

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapTraderWriteErrorMapsGlobalLoginConstraint(t *testing.T) {
	err := mapTraderWriteError(&pgconn.PgError{
		Code:           "23505",
		ConstraintName: "users_login_key",
	})

	if !errors.Is(err, ErrDuplicateLogin) {
		t.Fatalf("error = %v, want ErrDuplicateLogin", err)
	}
}
