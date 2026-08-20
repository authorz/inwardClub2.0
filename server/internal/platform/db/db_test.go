package db

import (
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestIsDuplicateKey(t *testing.T) {
	err := fmt.Errorf("write member: %w", &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '13800000000' for key 'members.uq_members_phone'",
	})
	if !IsDuplicateKey(err, "uq_members_phone") {
		t.Fatal("expected named duplicate key to be detected through wrapping")
	}
	if IsDuplicateKey(err, "uq_members_openid") {
		t.Fatal("must not match a different unique key")
	}
}
