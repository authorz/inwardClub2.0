package wallet

import "errors"

// CentsPerCoin is the single exchange rule between fiat order amounts and the
// integer coin asset: one coin pays one yuan, which is one hundred cents.
const CentsPerCoin int64 = 100

var (
	ErrCoinAmountNotPositive  = errors.New("coin payment amount must be positive")
	ErrCoinAmountNotWholeYuan = errors.New("coin payment amount must be whole yuan")
)

// CoinsRequired converts an integer-cent fiat amount into the integer number
// of coins required to pay or refund it. Coin payments never round: a fractional
// yuan amount must use another payment method.
func CoinsRequired(amountCent int64) (int64, error) {
	if amountCent <= 0 {
		return 0, ErrCoinAmountNotPositive
	}
	if amountCent%CentsPerCoin != 0 {
		return 0, ErrCoinAmountNotWholeYuan
	}
	return amountCent / CentsPerCoin, nil
}
