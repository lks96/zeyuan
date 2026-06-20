package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID int64 `json:"userId"`
	Exp    int64 `json:"exp"`
}

func SignToken(secret string, userID int64, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Exp:    time.Now().Add(ttl).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(secret, encodedPayload)
	return encodedPayload + "." + signature, nil
}

func ParseToken(secret string, token string) (Claims, error) {
	payload, signature, ok := strings.Cut(token, ".")
	if !ok || payload == "" || signature == "" {
		return Claims{}, ErrInvalidToken
	}

	expected := sign(secret, payload)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return Claims{}, ErrInvalidToken
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(rawPayload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	if claims.UserID <= 0 || claims.Exp < time.Now().Unix() {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}

func sign(secret string, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	mac.Write([]byte("."))
	mac.Write([]byte(strconv.Itoa(len(payload))))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
