// Fiat amounts use integer cents; wallet coin amounts use integer coin counts.
// One coin pays one yuan (100 cents). Never round fractional-yuan coin payments.
const CENTS_PER_COIN = 100;

function coinsRequired(amountCent) {
  const amount = Number(amountCent);
  if (!Number.isSafeInteger(amount) || amount <= 0 || amount % CENTS_PER_COIN !== 0) return null;
  return amount / CENTS_PER_COIN;
}

module.exports = { CENTS_PER_COIN, coinsRequired };
