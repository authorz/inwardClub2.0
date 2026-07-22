package auth

import (
	"crypto/rand"
	"encoding/binary"
)

// inviteCodeLen is the fixed length of a member invite code: 6 numeric digits.
const inviteCodeLen = 6

// newInviteCode returns a random 6-digit numeric invite code (leading zeros
// allowed, e.g. "004217"). Global uniqueness is enforced by the members
// .invite_code UNIQUE constraint; callers retry on a duplicate-key violation.
func newInviteCode() string {
	var b [8]byte
	// crypto/rand.Read never returns a short read; ignore err per stdlib guidance.
	_, _ = rand.Read(b[:])
	n := binary.BigEndian.Uint64(b[:]) % 1_000_000
	digits := []byte{'0', '0', '0', '0', '0', '0'}
	for i := inviteCodeLen - 1; i >= 0; i-- {
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits)
}
