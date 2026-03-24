// Пакет router собирает маршруты и middleware в единый HTTP-обработчик.
// Реализуйте этот пакет самостоятельно.
package router

import (
	"net/http"

	"gopherledger/internal/handler"
)

// New создаёт и возвращает HTTP-обработчик со всеми маршрутами.
//
// Публичные маршруты (без авторизации):
//
//	POST /api/user/register
//	POST /api/user/login
//
// Защищённые маршруты (требуют токен):
//
//	POST /api/user/orders
//	GET  /api/user/orders
//	GET  /api/user/balance
//	POST /api/user/balance/withdraw
//	GET  /api/user/withdrawals
//	POST /api/stats/export
func New(h *handler.Handler) http.Handler {
	panic("не реализовано")
}
