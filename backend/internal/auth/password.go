package auth

import (
	"crypto/sha256"
	"fmt"
)

func HashPassword(username string, password string) string {
	sum := sha256.Sum256([]byte(username + ":" + password))
	return fmt.Sprintf("%x", sum)
}
