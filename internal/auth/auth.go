// Пакет auth отвечает за генерацию и проверку токенов аутентификации.
// Токен - это случайная уникальная строка (например, UUID или hex-строка),
// которая однозначно связана с конкретным пользователем.
//
// Внутри пакета нужно хранить соответствие токен -> userID.
// Используйте для этого map с защитой от конкурентного доступа.
// Реализуйте этот пакет самостоятельно.

package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"sync"
)

// ErrInvalidToken возвращается, если токен не найден или недействителен.
var ErrInvalidToken = errors.New("недействительный токен")

type tokenStore struct {
	mu     sync.RWMutex
	tokens map[string]int64
}

var store = &tokenStore{tokens: make(map[string]int64)}

// GenerateToken создаёт новый токен для пользователя с указанным ID
// и сохраняет связь токен -> userID внутри пакета.
func GenerateToken(userID int64) (string, error) {
	nums := make([]byte, 16)
	if _, err := rand.Read(nums); err != nil {
		return "", err
	}
	token := hex.EncodeToString(nums)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.tokens[token] = userID
	return token, nil
}

// ValidateToken проверяет токен и возвращает ID пользователя.
// Возвращает ErrInvalidToken если токен не найден.
func ValidateToken(token string) (int64, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	id, ok := store.tokens[token]
	if !ok {
		return 0, ErrInvalidToken
	}
	return id, nil
}
