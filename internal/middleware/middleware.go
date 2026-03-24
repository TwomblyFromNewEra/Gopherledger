// Пакет middleware содержит HTTP-middleware.
// Реализуйте Auth, Logging и Recover самостоятельно.
package middleware

import (
	"net/http"
)

// Auth проверяет токен из заголовка Authorization и помещает ID пользователя в контекст.
// Запросы без валидного токена получают ответ 401 Unauthorized.
//
// Что нужно сделать:
//   - прочитать токен из заголовка
//   - проверить токен через пакет auth
//   - поместить ID пользователя в контекст запроса
//   - передать управление следующему handler или вернуть 401
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// реализуйте самостоятельно
	})
}

// statusRecorder оборачивает http.ResponseWriter для перехвата статус-кода.
// Используйте эту структуру в Logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// Logging логирует метод, путь, статус ответа и время выполнения каждого запроса.
//
// Что нужно сделать:
//   - зафиксировать время начала запроса
//   - обернуть w в statusRecorder для перехвата статус-кода
//   - после выполнения handler записать лог
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// реализуйте самостоятельно
	})
}

// Recover перехватывает панику внутри handler, логирует её и возвращает
// клиенту ответ 500 Internal Server Error вместо того, чтобы уронить сервер.
//
// Что нужно сделать:
//   - добавить defer с вызовом recover()
//   - если паника произошла, залогировать её и отдать 500
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// реализуйте самостоятельно
	})
}