// Пакет handler содержит HTTP-обработчики.
//
// Взаимодействие с бизнес-логикой осуществляется через интерфейс.
// Определите этот интерфейс здесь, по месту использования.
// Реализуйте все обработчики самостоятельно.

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gopherledger/internal/domain"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Handler хранит зависимость от бизнес-логики.
// Замените interface{} на свой интерфейс.

type ServiceInterface interface {
	RegisterUser(login, password string) (string, error)
	LoginUser(login, password string) (string, error)
	CreateOrder(userID int64, number string) (*domain.Order, error)
	GetUserOrders(userID int64) ([]domain.Order, error)
	GetBalance(userID int64) (domain.Balance, error)
	Withdraw(userID int64, orderNumber string, sum float64) error
	GetWithdrawals(userID int64) ([]domain.Withdrawal, error)
}
type Handler struct {
	svc ServiceInterface
}

type contextKey string

const CtxKeyUserID contextKey = "userID"

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type OrderResponse struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type BalanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawRequest struct {
	OrderNumber string  `json:"order_number"`
	Sum         float64 `json:"sum"`
}

type WithdrawalResponse struct {
	OrderNumber string    `json:"order_number"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

type StatsResponse struct {
	UsersCount     int64            `json:"users_count"`
	OrdersCount    int64            `json:"orders_count"`
	OrdersStatus   map[string]int64 `json:"orders_status"`
	AccrualTotal   float64          `json:"accrual_total"`
	WithdrawnTotal float64          `json:"withdrawn_total"`
	GeneratedAt    time.Time        `json:"generated_at"`
}

// New создаёт Handler.
func New(svc ServiceInterface) *Handler {
	return &Handler{
		svc: svc,
	}
}

// ---------------------------------------------------------------------------
// Вспомогательные функции для ответов - предоставлены
// ---------------------------------------------------------------------------

// writeError записывает JSON-ответ с ошибкой.
// Клиент видит только userMsg. Внутренние детали пишутся только в лог.
// Прочитайте ТЗ и создайте структуру тела ответа самостоятельно.
func writeError(w http.ResponseWriter, status int, code, userMsg string, internalErr error) {
	if internalErr != nil {
		log.Printf("ошибка code=%s status=%d: %v", code, status, internalErr)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// TODO: создайте структуру ответа и сериализуйте её
	json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: userMsg,
	})
}

// writeJSON записывает успешный JSON-ответ.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Обработчики - реализуйте самостоятельно
// ---------------------------------------------------------------------------

// Register обрабатывает POST /api/user/register.
// При успехе: 200 OK, заголовок Authorization с токеном.
// При дублировании логина: 409 Conflict.
// При некорректных данных: 400 Bad Request.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" || req.Login == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Некорректные данные", err)
		return
	}

	token, err := h.svc.RegisterUser(req.Login, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserExists) {
			writeError(w, http.StatusConflict, "user_exists", "Пользователь с таким логином уже существует", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", err)
		return

	}

	w.Header().Set("Authorization", token)
	writeJSON(w, http.StatusOK, AuthResponse{Token: token})

}

// Login обрабатывает POST /api/user/login.
// При успехе: 200 OK, заголовок Authorization с токеном.
// При неверных данных: 401 Unauthorized.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" || req.Login == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Некорректные данные", err)
		return
	}
	token, err := h.svc.LoginUser(req.Login, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrInvalidPassword) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Неверный логин или пароль", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", err)
		return
	}
	w.Header().Set("Authorization", token)
	writeJSON(w, http.StatusOK, AuthResponse{Token: token})
}

// CreateOrder обрабатывает POST /api/user/orders.
// Тело запроса: номер заказа в виде обычного текста.
// 202 Accepted  - новый заказ принят в обработку.
// 200 OK        - заказ уже загружен этим пользователем.
// 409 Conflict  - заказ принадлежит другому пользователю.
// 422 Unprocessable Entity - номер не прошёл проверку Луна.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Пользователь не авторизован", nil)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Не удалось прочитать тело запроса", err)
		return
	}
	defer r.Body.Close()

	o := strings.TrimSpace(string(body))
	if o == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Номер заказа не может быть пустым", nil)
		return
	}

	order, err := h.svc.CreateOrder(userID, o)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOrderOwnedByUser):
			w.WriteHeader(http.StatusOK)
			return
		case errors.Is(err, domain.ErrOrderExists):
			writeError(w, http.StatusConflict, "order_conflict", "Номер заказа уже из другого аккаунта", err)
			return
		case errors.Is(err, domain.ErrInvalidOrder):
			writeError(w, http.StatusUnprocessableEntity, "invalid_order", "Неверный формат номера (проверка Луна)", err)
			return
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", err)
			return
		}
	}
	writeJSON(w, http.StatusAccepted, OrderResponse{
		Number:     order.Number,
		Status:     order.Status,
		Accrual:    order.Accrual,
		UploadedAt: order.UploadedAt,
	})
}

// GetOrders обрабатывает GET /api/user/orders.
// 200 OK с JSON-массивом заказов или 204 No Content если заказов нет.
func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Пользователь не авторизован", nil)
		return
	}
	orders, err := h.svc.GetUserOrders(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", err)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	res := make([]OrderResponse, len(orders))
	for i, o := range orders {
		res[i] = OrderResponse{
			Number:     o.Number,
			Status:     o.Status,
			Accrual:    o.Accrual,
			UploadedAt: o.UploadedAt,
		}
	}
	writeJSON(w, http.StatusOK, res)
}

// GetBalance обрабатывает GET /api/user/balance.
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Пользователь не авторизован", nil)
		return
	}
	balance, err := h.svc.GetBalance(userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Пользователь не найден", err)
			return
		}
	}
	writeJSON(w, http.StatusOK, BalanceResponse{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	})
}

// Withdraw обрабатывает POST /api/user/balance/withdraw.
// 200 OK при успехе.
// 402 Payment Required при нехватке баллов.
// 422 Unprocessable Entity при неверном номере заказа.
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Пользователь не авторизован", nil)
		return
	}
	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Некорректные данные", err)
		return
	}
	if req.OrderNumber == "" || req.Sum <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Номер заказа и сумма должны быть указаны", nil)
		return
	}
	err := h.svc.Withdraw(userID, req.OrderNumber, req.Sum)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidOrder):
			writeError(w, http.StatusUnprocessableEntity, "invalid_order", "Неверный формат номера заказа", err)
			return
		case errors.Is(err, domain.ErrInsufficientFunds):
			writeError(w, http.StatusPaymentRequired, "insufficient_funds", "Недостаточно баллов для списания", err)
			return
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", err)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

}

// GetWithdrawals обрабатывает GET /api/user/withdrawals.
// 200 OK с массивом или 204 No Content если списаний нет.
func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Пользователь не авторизован", nil)
		return
	}
	withdrawn, err := h.svc.GetWithdrawals(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", err)
		return
	}
	if len(withdrawn) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	withdrawns := make([]WithdrawalResponse, len(withdrawn))
	for i, w := range withdrawn {
		withdrawns[i] = WithdrawalResponse{
			OrderNumber: w.OrderNumber,
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt,
		}
	}
	writeJSON(w, http.StatusOK, withdrawns)
}

// ExportStats обрабатывает POST /api/stats/export.
// Собирает статистику системы и записывает её в текстовый файл stats.txt
// в корне проекта. Возвращает 200 OK при успехе.
//
// Файл должен содержать:
//   - общее число зарегистрированных пользователей
//   - общее число заказов и их распределение по статусам
//   - суммарное количество начисленных баллов
//   - суммарное количество списанных баллов
//   - время генерации отчёта
//
// Для работы с файлами используйте пакет os (неделя 8).
func (h *Handler) ExportStats(w http.ResponseWriter, r *http.Request) {
	// 1. Проверяем авторизацию и получаем ID текущего пользователя
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Пользователь не авторизован", nil)
		return
	}

	// 2. Получаем все заказы этого пользователя
	orders, err := h.svc.GetUserOrders(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить заказы", err)
		return
	}

	// 3. Получаем баланс этого пользователя (чтобы узнать списания)
	balance, err := h.svc.GetBalance(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось получить баланс", err)
		return
	}

	// 4. Считаем метрики по заказам прямо здесь, на лету
	var accrualTotal float64
	ordersStatus := make(map[string]int64)

	for _, o := range orders {
		ordersStatus[o.Status]++
		accrualTotal += o.Accrual
	}

	// 5. Собираем красивую текстовую строку через fmt.Sprintf (без string(rune()))
	content := "=== Личная статистика пользователя ===\n"
	content += fmt.Sprintf("ID пользователя: %d\n", userID)
	content += fmt.Sprintf("Заказов суммарно: %d\n", len(orders))
	content += "Распределение по статусам:\n"

	// Если заказов нет, мапа будет пустой, обработаем красиво
	if len(orders) == 0 {
		content += "  - Нет загруженных заказов\n"
	} else {
		for status, count := range ordersStatus {
			content += fmt.Sprintf("  - %s: %d\n", status, count)
		}
	}

	content += fmt.Sprintf("Начислено баллов суммарно: %.2f\n", accrualTotal)
	content += fmt.Sprintf("Списано баллов суммарно: %.2f\n", balance.Withdrawn)
	content += fmt.Sprintf("Текущий баланс: %.2f\n", balance.Current)
	content += fmt.Sprintf("Время генерации отчета: %s\n", time.Now().Format(time.RFC3339))

	// 6. Записываем файл stats.txt в корень проекта
	err = os.WriteFile("stats.txt", []byte(content), 0666)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось записать файл", err)
		return
	}

	// 7. Возвращаем чистый 200 OK без тела, как просит ТЗ
	w.WriteHeader(http.StatusOK)
}

// ---------------------------------------------------------------------------
// Вспомогательная функция для работы с контекстом - предоставлена
// ---------------------------------------------------------------------------

// UserIDFromContext извлекает ID аутентифицированного пользователя из контекста.
// Возвращает 0, false если значение отсутствует.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	// реализуйте самостоятельно
	userID, ok := ctx.Value(CtxKeyUserID).(int64)
	return userID, ok
}
