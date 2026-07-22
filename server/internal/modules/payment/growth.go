package payment

// This file holds the recharge VIP-growth settlement policy: the pure tier
// resolution that maps a member's accumulated growth_value to the VIP tier they
// qualify for. The amounts themselves are never hard-coded here — the growth
// grant comes from recharge_products.growth_amount and the thresholds come from
// membership_tiers, both admin-configured (spec §1). Keeping the decision pure
// lets it be unit-tested without a live MySQL, matching the settlement spine
// test convention.

const (
	// growthAsset is the wallet_accounts.asset_type recharge growth credits into.
	// It mirrors wallet.AssetGrowthValue; the constant is duplicated locally to
	// keep the settlement package free of a wallet import, as coinsAsset already is.
	growthAsset = "growth_value"
	// growthSourceType tags the growth credit on the wallet ledger; combined with
	// the payment order id it forms the idempotency key, independent of the coins
	// credit so the two entitlements settle (and replay) independently.
	growthSourceType = "recharge_growth"
)

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
