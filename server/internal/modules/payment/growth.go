package payment

// This file holds the WeChat-payment VIP-growth settlement policy: ¥1 of the
// business-order amount earns 1 growth value, and the accumulated balance
// resolves the highest qualifying configured membership tier. Keeping the
// calculations pure lets them be unit-tested without a live MySQL.

const (
	// growthAsset is the wallet_accounts.asset_type payment growth credits into.
	// It mirrors wallet.AssetGrowthValue; the constant is duplicated locally to
	// keep the settlement package free of a wallet import, as coinsAsset already is.
	growthAsset = "growth_value"
	// wechatGrowthSourceType tags the growth credit on the wallet ledger; combined with
	// the payment order id it forms the idempotency key, independent of the coins
	// credit so the two entitlements settle (and replay) independently.
	wechatGrowthSourceType = "wechat_payment_growth"
)

// wechatGrowthAmount applies the rule shared by settlement, member benefits and
// rankings: ¥1 of a successful WeChat business-order amount earns 1 growth
// value. Order amounts are stored in cents and growth values are integers.
func wechatGrowthAmount(amountCent int64) int64 {
	if amountCent <= 0 {
		return 0
	}
	return amountCent / 100
}

// tierRow is the minimal membership_tiers projection the resolver needs: the id
// to persist and the level/threshold to rank by.
type tierRow struct {
	id        int64
	level     int
	threshold int64
}

// resolveTier returns the highest-ranked active tier the member qualifies for
// at the given growth_value balance, i.e. the tier with the greatest threshold
// that does not exceed the balance. Ties on threshold break toward the higher
// level. It reports ok=false when no tier qualifies (e.g. every configured tier
// has a threshold above the balance, or there are no tiers at all), in which
// case the caller must leave the member's current tier untouched.
//
// The input need not be pre-sorted; callers pass whatever membership_tiers
// returns. Only the qualifying comparison and the level/threshold tie-breaks
// matter, so the scan is order-independent.
func resolveTier(tiers []tierRow, balance int64) (tierRow, bool) {
	var best tierRow
	found := false
	for _, t := range tiers {
		if t.threshold > balance {
			continue // threshold not yet met
		}
		if !found ||
			t.threshold > best.threshold ||
			(t.threshold == best.threshold && t.level > best.level) {
			best = t
			found = true
		}
	}
	return best, found
}
