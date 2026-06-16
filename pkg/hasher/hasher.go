// Package hasher реализует алгоритм сокращения URL через base62 кодирование.
package hasher

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Generate возвращает случайный короткий код заданной длины.
func Generate(length int) (string, error) {
	var sb strings.Builder
	sb.Grow(length)

	alphabetLen := big.NewInt(int64(len(alphabet)))

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		sb.WriteByte(alphabet[idx.Int64()])
	}

	return sb.String(), nil
}
