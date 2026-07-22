package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// MemberRepository is the member persistence port.
type MemberRepository interface {
	GetByOpenID(ctx context.Context, openID string) (Member, error)
	GetByID(ctx context.Context, id int64) (Member, error)
	Create(ctx context.Context, m Member) (int64, error)
	BumpTokenVersion(ctx context.Context, id int64) error
}

// AccountRepository is the back-office account persistence port.
type AccountRepository interface {
	GetByUsername(ctx context.Context, username string) (Account, error)
	GetByID(ctx context.Context, id int64) (Account, error)
	BumpTokenVersion(ctx context.Context, id int64) error
}

type sqlMemberRepository struct{ db *platdb.DB }
type sqlAccountRepository struct{ db *platdb.DB }

// NewMemberRepository builds the MySQL member repository.
func NewMemberRepository(db *platdb.DB) MemberRepository { return &sqlMemberRepository{db: db} }

// NewAccountRepository builds the MySQL account repository.
func NewAccountRepository(db *platdb.DB) AccountRepository { return &sqlAccountRepository{db: db} }

const memberColumns = `id, COALESCE(wechat_openid,''), nickname, COALESCE(phone,''),
	COALESCE(invite_code,''), status, token_version, created_at, updated_at`

func scanMember(row interface{ Scan(...any) error }) (Member, error) {
	var m Member
	err := row.Scan(&m.ID, &m.WeChatOpenID, &m.Nickname, &m.Phone,
		&m.InviteCode, &m.Status, &m.TokenVersion, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (r *sqlMemberRepository) GetByOpenID(ctx context.Context, openID string) (Member, error) {
	const q = `SELECT ` + memberColumns + ` FROM members WHERE wechat_openid = ?`
	m, err := scanMember(r.db.QueryRowContext(ctx, q, openID))
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, apperr.NotFound("member not found")
	}
	if err != nil {
		return Member{}, apperr.Internal(err)
	}
	return m, nil
}

func (r *sqlMemberRepository) GetByID(ctx context.Context, id int64) (Member, error) {
	const q = `SELECT ` + memberColumns + ` FROM members WHERE id = ?`
	m, err := scanMember(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, apperr.NotFound("member not found")
	}
	if err != nil {
		return Member{}, apperr.Internal(err)
	}
	return m, nil
}

func (r *sqlMemberRepository) Create(ctx context.Context, m Member) (int64, error) {
	const q = `INSERT INTO members
		(wechat_openid, nickname, invite_code, status, token_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?)`
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, q, m.WeChatOpenID, m.Nickname, nullable(m.InviteCode), StatusActive, now, now)
	if err != nil {
		return 0, apperr.Internal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, apperr.Internal(err)
	}
	return id, nil
}

func (r *sqlMemberRepository) BumpTokenVersion(ctx context.Context, id int64) error {
	const q = `UPDATE members SET token_version = token_version + 1, updated_at = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, time.Now().UTC(), id); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

const accountColumns = `id, username, password_hash, display_name, role,
	COALESCE(store_id,0), status, token_version, created_at, updated_at`

func scanAccount(row interface{ Scan(...any) error }) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.DisplayName, &a.Role,
		&a.StoreID, &a.Status, &a.TokenVersion, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (r *sqlAccountRepository) GetByUsername(ctx context.Context, username string) (Account, error) {
	const q = `SELECT ` + accountColumns + ` FROM admin_accounts WHERE username = ?`
	a, err := scanAccount(r.db.QueryRowContext(ctx, q, username))
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, apperr.NotFound("account not found")
	}
	if err != nil {
		return Account{}, apperr.Internal(err)
	}
	return a, nil
}

func (r *sqlAccountRepository) GetByID(ctx context.Context, id int64) (Account, error) {
	const q = `SELECT ` + accountColumns + ` FROM admin_accounts WHERE id = ?`
	a, err := scanAccount(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, apperr.NotFound("account not found")
	}
	if err != nil {
		return Account{}, apperr.Internal(err)
	}
	return a, nil
}

func (r *sqlAccountRepository) BumpTokenVersion(ctx context.Context, id int64) error {
	const q = `UPDATE admin_accounts SET token_version = token_version + 1, updated_at = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, q, time.Now().UTC(), id); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
