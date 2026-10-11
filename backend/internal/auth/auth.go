package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	username, password string
	secret             []byte
}

func New(username, password, secret string) *Manager {
	return &Manager{username, password, []byte(secret)}
}
func (m *Manager) Login(username, password string) (string, error) {
	a := subtle.ConstantTimeCompare([]byte(username), []byte(m.username))
	b := subtle.ConstantTimeCompare([]byte(password), []byte(m.password))
	if a&b != 1 {
		return "", errors.New("invalid credentials")
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%s:%d:%s", m.username, time.Now().Add(24*time.Hour).Unix(), base64.RawURLEncoding.EncodeToString(nonce))
	body := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return body + "." + m.sign(body), nil
}
func (m *Manager) sign(body string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (m *Manager) Verify(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	expected := m.sign(parts[0])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[1])) != 1 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	fields := strings.Split(string(decoded), ":")
	if len(fields) != 3 || fields[0] != m.username {
		return false
	}
	exp, err := strconv.ParseInt(fields[1], 10, 64)
	return err == nil && time.Now().Unix() < exp
}
