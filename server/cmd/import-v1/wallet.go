package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type walletKey struct {
	memberID  int64
	assetType string
}
type walletState struct {
	accountID, sourceBalance, targetBalance int64
	hasSource                               bool
}

func (i *importer) migrateWallet(ctx context.Context, tx *sql.Tx) error {
	states := map[walletKey]*walletState{}
	accountRows := [][]any{}
	rows, err := i.source.QueryContext(ctx, `SELECT id,FLOOR(COALESCE(points,0)),FLOOR(COALESCE(balance,0)),FLOOR(COALESCE(all_balance,0)),created_at,updated_at FROM users ORDER BY id`)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	accounts := int64(0)
	for rows.Next() {
		var member, points, coins, growth int64
		var created, updated sql.NullTime
		if err = rows.Scan(&member, &points, &coins, &growth, &created, &updated); err != nil {
			rows.Close()
			return err
		}
		for asset, amount := range map[string]int64{"points": points, "coins": coins, "growth_value": growth} {
			assetCode := map[string]int64{"points": 1, "coins": 2, "growth_value": 3}[asset]
			accountID := member*10 + assetCode
			accountRows = append(accountRows, []any{accountID, member, asset, amount, int64(0), int64(0), nullableTime(created, now), nullableTime(updated, now)})
			states[walletKey{member, asset}] = &walletState{accountID: accountID, targetBalance: amount}
			accounts++
		}
	}
	rows.Close()
	if err := execBatches(ctx, tx, `INSERT INTO wallet_accounts
		(id,member_id,asset_type,available_amount,held_amount,version,created_at,updated_at)`, 8, accountRows); err != nil {
		return err
	}
	rows, err = i.source.QueryContext(ctx, `SELECT id,user_id,amount_type,trans_type,FLOOR(ABS(amount)),FLOOR(balance_after),COALESCE(description,''),related_id,created_at FROM transaction_records ORDER BY id`)
	if err != nil {
		return err
	}
	entries := int64(0)
	ledgerRows := [][]any{}
	for rows.Next() {
		var id, member, amount, balance int64
		var amountType string
		var transType int
		var description string
		var related sql.NullInt64
		var created sql.NullTime
		if err = rows.Scan(&id, &member, &amountType, &transType, &amount, &balance, &description, &related, &created); err != nil {
			rows.Close()
			return err
		}
		asset := map[string]string{"balance": "coins", "points": "points", "all_balance": "growth_value"}[amountType]
		state := states[walletKey{member, asset}]
		if state == nil {
			rows.Close()
			return fmt.Errorf("missing wallet for transaction %d", id)
		}
		direction := "credit"
		if transType == 2 {
			direction = "debit"
		}
		ledgerRows = append(ledgerRows, []any{id, state.accountID, member, asset, direction, amount, balance, truncateUTF8(defaultString(description, "1.0交易记录"), 64), "v1_transaction_records", nullableInt(related), fmt.Sprintf("v1:transaction_records:%d", id), nullableTime(created, now)})
		if err = i.mapID(ctx, tx, "transaction_records", id, "wallet_ledger_entries", id); err != nil {
			rows.Close()
			return err
		}
		state.sourceBalance = balance
		state.hasSource = true
		entries++
	}
	rows.Close()
	correctionID := int64(40_000)
	corrections := int64(0)
	for key, state := range states {
		base := int64(0)
		if state.hasSource {
			base = state.sourceBalance
		}
		diff := state.targetBalance - base
		if diff == 0 {
			continue
		}
		direction := "credit"
		amount := diff
		if diff < 0 {
			direction = "debit"
			amount = -diff
		}
		correctionID++
		ledgerRows = append(ledgerRows, []any{correctionID, state.accountID, key.memberID, key.assetType, direction, amount, state.targetBalance, "1.0迁移余额校准", "v1_opening_balance", nil, fmt.Sprintf("v1:opening:%d:%s", key.memberID, key.assetType), time.Now().UTC()})
		corrections++
	}
	if err := execBatches(ctx, tx, `INSERT INTO wallet_ledger_entries
		(id,account_id,member_id,asset_type,direction,amount,balance_after,reason,source_type,source_id,idem_key,created_at)`, 12, ledgerRows); err != nil {
		return err
	}
	i.metrics["walletAccountsImported"] = accounts
	i.metrics["walletLedgerEntriesImported"] = entries
	i.metrics["walletOpeningCorrections"] = corrections
	var sourceCoinRaw string
	if err = i.source.QueryRowContext(ctx, `SELECT CAST(SUM(COALESCE(balance,0)) AS CHAR) FROM users`).Scan(&sourceCoinRaw); err == nil {
		cent, parseErr := centsFromString(sourceCoinRaw)
		if parseErr == nil {
			i.metrics["sourceCoinBalanceHundredths"] = cent
		}
	}
	var targetCoins int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(available_amount),0) FROM wallet_accounts WHERE asset_type='coins'`).Scan(&targetCoins); err != nil {
		return err
	}
	i.metrics["targetCoinBalanceFloored"] = targetCoins
	return nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
