package wallet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"github.com/inwardclub/server/internal/modules/printer"
	platdb "github.com/inwardclub/server/internal/platform/db"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func TestWithdrawPointsIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_INTEGRATION") != "1" {
		t.Skip("set RUN_MYSQL_INTEGRATION=1 to run")
	}
	ctx := context.Background()
	_, sourceFile, _, _ := runtime.Caller(0)
	env, err := godotenv.Read(filepath.Join(filepath.Dir(sourceFile), "../../../.env"))
	if err != nil {
		t.Fatalf("load .env: %v", err)
	}
	database, err := platdb.Open(ctx, env["MYSQL_DSN"])
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()

	now := time.Now().UTC().Truncate(time.Second)
	suffix := now.UnixNano()
	storeResult, err := database.ExecContext(ctx, `INSERT INTO stores
		(name, status, created_at, updated_at) VALUES (?, 'active', ?, ?)`,
		fmt.Sprintf("points-withdrawal-store-%d", suffix), now, now)
	if err != nil {
		t.Fatalf("insert store: %v", err)
	}
	storeID, _ := storeResult.LastInsertId()
	var tierID int64
	var vipLevel int
	if err := database.QueryRowContext(ctx, `SELECT id, level FROM membership_tiers
		WHERE status = 'active' ORDER BY level DESC, id DESC LIMIT 1`).Scan(&tierID, &vipLevel); err != nil {
		database.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, storeID)
		t.Fatalf("select membership tier: %v", err)
	}
	phone := fmt.Sprintf("176%08d", suffix%100000000)
	memberResult, err := database.ExecContext(ctx, `INSERT INTO members
		(nickname, phone, current_tier_id, profile_completed, status, created_at, updated_at)
		VALUES ('积分提取会员', ?, ?, 1, 'active', ?, ?)`, phone, tierID, now, now)
	if err != nil {
		database.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, storeID)
		t.Fatalf("insert member: %v", err)
	}
	memberID, _ := memberResult.LastInsertId()
	accountResult, err := database.ExecContext(ctx, `INSERT INTO wallet_accounts
		(member_id, asset_type, available_amount, held_amount, version, created_at, updated_at)
		VALUES (?, 'points', 1000, 0, 0, ?, ?)`, memberID, now, now)
	if err != nil {
		database.ExecContext(ctx, `DELETE FROM members WHERE id = ?`, memberID)
		database.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, storeID)
		t.Fatalf("insert points account: %v", err)
	}
	accountID, _ := accountResult.LastInsertId()
	printerResult, err := database.ExecContext(ctx, `INSERT INTO printer_devices
		(store_id, name, provider, device_sn, device_key, status, created_at, updated_at)
		VALUES (?, '积分提取集成测试打印机', 'xpyun', ?, '', 'active', ?, ?)`,
		storeID, fmt.Sprintf("POINT-WITHDRAW-%d", suffix), now, now)
	if err != nil {
		database.ExecContext(ctx, `DELETE FROM wallet_accounts WHERE id = ?`, accountID)
		database.ExecContext(ctx, `DELETE FROM members WHERE id = ?`, memberID)
		database.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, storeID)
		t.Fatalf("insert printer: %v", err)
	}
	printerID, _ := printerResult.LastInsertId()
	idemKey := fmt.Sprintf("point-withdrawal-integration-%d", suffix)
	insufficientIdemKey := idemKey + "-insufficient"
	var withdrawalID int64
	defer func() {
		if withdrawalID > 0 {
			printIdemKey := fmt.Sprintf("point-withdrawal:%d:printer:%d:print-receipt", withdrawalID, printerID)
			database.ExecContext(ctx, `DELETE FROM outbox_events WHERE idem_key = ?`, printIdemKey)
			database.ExecContext(ctx, `DELETE FROM print_jobs WHERE idem_key = ?`, printIdemKey)
		}
		database.ExecContext(ctx, `DELETE FROM wallet_ledger_entries WHERE idem_key IN (?, ?)`, idemKey, insufficientIdemKey)
		database.ExecContext(ctx, `DELETE FROM point_withdrawals WHERE idem_key IN (?, ?)`, idemKey, insufficientIdemKey)
		database.ExecContext(ctx, `DELETE FROM printer_devices WHERE id = ?`, printerID)
		database.ExecContext(ctx, `DELETE FROM wallet_accounts WHERE id = ?`, accountID)
		database.ExecContext(ctx, `DELETE FROM members WHERE id = ?`, memberID)
		database.ExecContext(ctx, `DELETE FROM stores WHERE id = ?`, storeID)
	}()

	repo := NewPointsRepository(database, nil)
	result, err := repo.WithdrawPoints(ctx, memberID, storeID, 300, idemKey)
	if err != nil {
		t.Fatalf("withdraw points: %v", err)
	}
	withdrawalID = result.RequestID
	if result.AssetType != AssetPoints || result.Amount != 300 || result.BalanceAfter != 700 || result.Status != "approved" {
		t.Fatalf("unexpected withdrawal result: %+v", result)
	}

	var balance int64
	if err := database.QueryRowContext(ctx, `SELECT available_amount FROM wallet_accounts WHERE id = ?`, accountID).
		Scan(&balance); err != nil || balance != 700 {
		t.Fatalf("points balance = %d, err=%v; want 700", balance, err)
	}
	var direction, reason, sourceType string
	var ledgerAmount, ledgerBalance, sourceID int64
	if err := database.QueryRowContext(ctx, `SELECT direction, amount, balance_after, reason, source_type, source_id
		FROM wallet_ledger_entries WHERE idem_key = ?`, idemKey).Scan(
		&direction, &ledgerAmount, &ledgerBalance, &reason, &sourceType, &sourceID,
	); err != nil {
		t.Fatalf("select withdrawal ledger: %v", err)
	}
	if direction != "debit" || ledgerAmount != 300 || ledgerBalance != 700 ||
		reason != "point_withdrawal" || sourceType != "point_withdrawal" || sourceID != withdrawalID {
		t.Fatalf("unexpected withdrawal ledger: direction=%s amount=%d balance=%d reason=%s source=%s/%d",
			direction, ledgerAmount, ledgerBalance, reason, sourceType, sourceID)
	}

	printIdemKey := fmt.Sprintf("point-withdrawal:%d:printer:%d:print-receipt", withdrawalID, printerID)
	var template, receiptContent string
	if err := database.QueryRowContext(ctx, `SELECT template, JSON_UNQUOTE(JSON_EXTRACT(payload, '$.Content'))
		FROM print_jobs WHERE idem_key = ?`, printIdemKey).Scan(&template, &receiptContent); err != nil {
		t.Fatalf("select withdrawal receipt: %v", err)
	}
	maskedPhone := phone[:3] + "****" + phone[len(phone)-4:]
	if template != printer.PointWithdrawalReceiptTemplate {
		t.Fatalf("receipt template = %q, want %q", template, printer.PointWithdrawalReceiptTemplate)
	}
	for _, want := range []string{
		"手机尾号  " + maskedPhone, fmt.Sprintf("会员等级  VIP%d", vipLevel),
		"提取积分  300", "剩余积分  700", "合计  300",
		"请工作人员仔细检查核验！",
	} {
		if !strings.Contains(receiptContent, want) {
			t.Fatalf("withdrawal receipt missing %q:\n%s", want, receiptContent)
		}
	}

	retried, err := repo.WithdrawPoints(ctx, memberID, storeID, 300, idemKey)
	if err != nil {
		t.Fatalf("retry withdrawal: %v", err)
	}
	if retried != result {
		t.Fatalf("retry result = %+v, want %+v", retried, result)
	}
	for table, query := range map[string]string{
		"withdrawals": `SELECT COUNT(*) FROM point_withdrawals WHERE idem_key = ?`,
		"ledger":      `SELECT COUNT(*) FROM wallet_ledger_entries WHERE idem_key = ?`,
		"print jobs":  `SELECT COUNT(*) FROM print_jobs WHERE idem_key = ?`,
	} {
		key := idemKey
		if table == "print jobs" {
			key = printIdemKey
		}
		var count int
		if err := database.QueryRowContext(ctx, query, key).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s count = %d, err=%v; want 1", table, count, err)
		}
	}
	if err := database.QueryRowContext(ctx, `SELECT available_amount FROM wallet_accounts WHERE id = ?`, accountID).
		Scan(&balance); err != nil || balance != 700 {
		t.Fatalf("points balance after retry = %d, err=%v; want 700", balance, err)
	}

	if _, err := repo.WithdrawPoints(ctx, memberID, storeID, 701, insufficientIdemKey); err == nil ||
		apperr.From(err).Code != apperr.CodeInsufficientBalance {
		t.Fatalf("insufficient withdrawal error = %v, want INSUFFICIENT_BALANCE", err)
	}
	var insufficientCount int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM point_withdrawals WHERE idem_key = ?`, insufficientIdemKey).
		Scan(&insufficientCount); err != nil || insufficientCount != 0 {
		t.Fatalf("insufficient withdrawal count = %d, err=%v; want 0", insufficientCount, err)
	}
}
