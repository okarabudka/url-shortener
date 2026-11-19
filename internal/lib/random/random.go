// Package random generates aliases for short links.
package random

import "crypto/rand"

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// maxUnbiased is the largest multiple of len(alphabet) that fits in a byte. Bytes at or above it
// are skipped, so every character is equally likely.
const maxUnbiased = 256 / len(alphabet) * len(alphabet)

// NewRandomString returns a random alphanumeric string of the given size. It uses crypto/rand
// so that aliases cannot be predicted from earlier ones.
func NewRandomString(size int) string {
	out := make([]byte, 0, size)
	buf := make([]byte, size)

	for len(out) < size {
		// crypto/rand.Read never returns an error since Go 1.24; it crashes the program instead.
		_, _ = rand.Read(buf)

		for _, b := range buf {
			if int(b) >= maxUnbiased {
				continue
			}
			out = append(out, alphabet[int(b)%len(alphabet)])
			if len(out) == size {
				break
			}
		}
	}

	return string(out)
}
