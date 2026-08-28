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
			res, err := tx.ExecContext(ctx, `INSERT INTO wallet_accounts(member_id,asset_type,available_amount,held_amount,version,created_at,updated_at) VALUES (?,?,?,0,0,?,?)`, member, asset, amount, nullableTime(created, now), nullableTime(updated, now))
			if err != nil {
				rows.Close()
				return fmt.Errorf("insert wallet %d/%s: %w", member, asset, err)
			}
			accountID, err := res.LastInsertId()
			if err != nil {
				rows.Close()
				return err
			}
			states[walletKey{member, asset}] = &walletState{accountID: accountID, targetBalance: amount}
			accounts++
		}
	}
	rows.Close()
	rows, err = i.source.QueryContext(ctx, `SELECT id,user_id,amount_type,trans_type,FLOOR(ABS(amount)),FLOOR(balance_after),COALESCE(description,''),related_id,created_at FROM transaction_records ORDER BY id`)
	if err != nil {
		return err
	}
	entries := int64(0)
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
		_, err = tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
			(id,account_id,member_id,asset_type,direction,amount,balance_after,reason,source_type,source_id,idem_key,created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, id, state.accountID, member, asset, direction, amount, balance, truncateUTF8(defaultString(description, "1.0交易记录"), 64), "v1_transaction_records", nullableInt(related), fmt.Sprintf("v1:transaction_records:%d", id), nullableTime(created, now))
		if err != nil {
			rows.Close()
			return fmt.Errorf("insert ledger %d: %w", id, err)
		}
		if err = i.mapID(ctx, tx, "transaction_records", id, "wallet_ledger_entries", id); err != nil {
			rows.Close()
			return err
		}
		state.sourceBalance = balance
		state.hasSource = true
		entries++
	}
	rows.Close()
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
		_, err = tx.ExecContext(ctx, `INSERT INTO wallet_ledger_entries
			(account_id,member_id,asset_type,direction,amount,balance_after,reason,source_type,idem_key,created_at)
			VALUES (?,?,?,?,?,?,?,'v1_opening_balance',?,UTC_TIMESTAMP())`, state.accountID, key.memberID, key.assetType, direction, amount, state.targetBalance, "1.0迁移余额校准", fmt.Sprintf("v1:opening:%d:%s", key.memberID, key.assetType))
		if err != nil {
			return err
		}
		corrections++
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
