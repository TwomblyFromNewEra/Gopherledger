package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gopherledger/internal/domain"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// FAKE SERVICE IMPLEMENTATION
// ============================================================================

type fakeService struct {
	mu          sync.Mutex
	users       map[string]string // login -> password
	userIDs     map[string]int64  // login -> userID
	tokens      map[string]int64  // token -> userID
	orders      map[string]*domain.Order
	balances    map[int64]domain.Balance
	withdrawals map[int64][]domain.Withdrawal
	userIDSeq   int64
	tokenIDSeq  int64
}

func newFakeService() *fakeService {
	return &fakeService{
		users:       make(map[string]string),
		userIDs:     make(map[string]int64),
		tokens:      make(map[string]int64),
		orders:      make(map[string]*domain.Order),
		balances:    make(map[int64]domain.Balance),
		withdrawals: make(map[int64][]domain.Withdrawal),
	}
}

func (f *fakeService) RegisterUser(login, password string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.users[login]; exists {
		return "", domain.ErrUserExists
	}

	f.userIDSeq++
	f.users[login] = password
	f.userIDs[login] = f.userIDSeq
	f.balances[f.userIDSeq] = domain.Balance{Current: 0, Withdrawn: 0}

	f.tokenIDSeq++
	token := fmt.Sprintf("token_%d", f.tokenIDSeq)
	f.tokens[token] = f.userIDSeq

	return token, nil
}

func (f *fakeService) LoginUser(login, password string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	pwd, exists := f.users[login]
	if !exists {
		return "", domain.ErrUserNotFound
	}

	if pwd != password {
		return "", domain.ErrInvalidPassword
	}

	f.tokenIDSeq++
	token := fmt.Sprintf("token_%d", f.tokenIDSeq)
	f.tokens[token] = f.userIDs[login]

	return token, nil
}

func (f *fakeService) CreateOrder(userID int64, number string) (*domain.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if order, exists := f.orders[number]; exists {
		if order.UserID == userID {
			return order, domain.ErrOrderOwnedByUser
		}
		return order, domain.ErrOrderExists
	}

	if !isValidLuhn(number) {
		return nil, domain.ErrInvalidOrder
	}

	order := &domain.Order{
		ID:         int64(len(f.orders)) + 1,
		UserID:     userID,
		Number:     number,
		Status:     domain.OrderStatusNew,
		Accrual:    0,
		UploadedAt: time.Now(),
	}

	f.orders[number] = order
	return order, nil
}

func (f *fakeService) GetUserOrders(userID int64) ([]domain.Order, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var result []domain.Order
	for _, order := range f.orders {
		if order.UserID == userID {
			result = append(result, *order)
		}
	}
	return result, nil
}

func (f *fakeService) GetBalance(userID int64) (domain.Balance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	balance, exists := f.balances[userID]
	if !exists {
		return domain.Balance{}, domain.ErrUserNotFound
	}
	return balance, nil
}

func (f *fakeService) Withdraw(userID int64, orderNumber string, sum float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !isValidLuhn(orderNumber) {
		return domain.ErrInvalidOrder
	}

	balance, exists := f.balances[userID]
	if !exists {
		return domain.ErrUserNotFound
	}

	if balance.Current < sum {
		return domain.ErrInsufficientFunds
	}

	balance.Current -= sum
	balance.Withdrawn += sum
	f.balances[userID] = balance

	withdrawal := domain.Withdrawal{
		ID:          int64(len(f.withdrawals[userID])) + 1,
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}
	f.withdrawals[userID] = append(f.withdrawals[userID], withdrawal)

	return nil
}

func (f *fakeService) GetWithdrawals(userID int64) ([]domain.Withdrawal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	withdrawals, exists := f.withdrawals[userID]
	if !exists {
		return []domain.Withdrawal{}, nil
	}

	result := make([]domain.Withdrawal, len(withdrawals))
	copy(result, withdrawals)
	return result, nil
}

func isValidLuhn(number string) bool {
	if number == "" {
		return false
	}

	var luhn int
	for i, j := 0, len(number)-1; j >= 0; i, j = i+1, j-1 {
		if number[j] < '0' || number[j] > '9' {
			return false
		}

		cur := int(number[j] - '0')

		if i%2 == 1 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
	}

	return luhn%10 == 0
}

// ============================================================================
// UNIT TESTS FOR HTTP HANDLERS
// ============================================================================

// --- REGISTER TESTS ---

func TestRegisterSuccess(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	body := LoginRequest{Login: "user1", Password: "pass123"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/user/register", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if w.Header().Get("Authorization") == "" {
		t.Error("Expected Authorization header to be set")
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode body: %v", err)
	}
	if resp.Token == "" {
		t.Error("Expected non-empty token in JSON response")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	body := LoginRequest{Login: "user1", Password: "pass123"}
	bodyBytes, _ := json.Marshal(body)

	req1 := httptest.NewRequest("POST", "/api/user/register", bytes.NewReader(bodyBytes))
	w1 := httptest.NewRecorder()
	h.Register(w1, req1)

	req2 := httptest.NewRequest("POST", "/api/user/register", bytes.NewReader(bodyBytes))
	w2 := httptest.NewRecorder()
	h.Register(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("Expected 409 Conflict, got %d", w2.Code)
	}
}

func TestRegisterEmptyFields(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	body := LoginRequest{Login: "", Password: ""}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/user/register", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}
}

// --- LOGIN TESTS ---

func TestLoginSuccess(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	_, _ = svc.RegisterUser("user1", "pass123")

	body := LoginRequest{Login: "user1", Password: "pass123"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/user/login", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	if w.Header().Get("Authorization") == "" {
		t.Error("Expected Authorization header")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	_, _ = svc.RegisterUser("user1", "pass123")

	body := LoginRequest{Login: "user1", Password: "wrong_password"}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/user/login", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

// --- CREATE ORDER TESTS ---

func TestCreateOrderSuccess(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	req := httptest.NewRequest("POST", "/api/user/orders", bytes.NewReader([]byte("4561261212345467"))) // Валидный Лун
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateOrder(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected 202 Accepted, got %d", w.Code)
	}
}

func TestCreateOrderInvalidLuhn(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	req := httptest.NewRequest("POST", "/api/user/orders", bytes.NewReader([]byte("1234567890123456"))) // Невалидный Лун
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.CreateOrder(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected 422 Unprocessable Entity, got %d", w.Code)
	}
}

func TestCreateOrderConflicts(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	// Предзаполняем заказ в базе от имени UserID = 1
	svc.orders["4561261212345467"] = &domain.Order{
		UserID: 1,
		Number: "4561261212345467",
		Status: domain.OrderStatusNew,
	}

	// 1. Повторный запрос от ТОГО ЖЕ юзера -> 200 OK
	ctx1 := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	req1 := httptest.NewRequest("POST", "/api/user/orders", bytes.NewReader([]byte("4561261212345467")))
	w1 := httptest.NewRecorder()
	h.CreateOrder(w1, req1.WithContext(ctx1))

	if w1.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for same user conflict, got %d", w1.Code)
	}

	// 2. Запрос на этот же номер от ДРУГОГО юзера -> 409 Conflict
	ctx2 := context.WithValue(context.Background(), CtxKeyUserID, int64(2))
	req2 := httptest.NewRequest("POST", "/api/user/orders", bytes.NewReader([]byte("4561261212345467")))
	w2 := httptest.NewRecorder()
	h.CreateOrder(w2, req2.WithContext(ctx2))

	if w2.Code != http.StatusConflict {
		t.Errorf("Expected 409 Conflict for other user, got %d", w2.Code)
	}
}

// --- GET ORDERS TESTS ---

func TestGetOrdersEmpty(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	req := httptest.NewRequest("GET", "/api/user/orders", nil)
	w := httptest.NewRecorder()

	h.GetOrders(w, req.WithContext(ctx))

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204 No Content, got %d", w.Code)
	}
}

func TestGetOrdersWithContent(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	svc.orders["4561261212345467"] = &domain.Order{
		UserID:     1,
		Number:     "4561261212345467",
		Status:     domain.OrderStatusNew,
		UploadedAt: time.Now(),
	}

	req := httptest.NewRequest("GET", "/api/user/orders", nil)
	w := httptest.NewRecorder()

	h.GetOrders(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	var resp []OrderResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode body: %v", err)
	}

	if len(resp) != 1 || resp[0].Number != "4561261212345467" {
		t.Error("Response slice elements mismatch or empty")
	}
}

// --- GET BALANCE TESTS ---

func TestGetBalanceSuccess(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	svc.balances[1] = domain.Balance{Current: 450.50, Withdrawn: 120.00}

	req := httptest.NewRequest("GET", "/api/user/balance", nil)
	w := httptest.NewRecorder()

	h.GetBalance(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	var resp BalanceResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if resp.Current != 450.50 || resp.Withdrawn != 120.00 {
		t.Errorf("Expected currents 450.50 and 120.00, got %f and %f", resp.Current, resp.Withdrawn)
	}
}

// --- WITHDRAW TESTS ---

func TestWithdrawSuccess(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	svc.balances[1] = domain.Balance{Current: 100, Withdrawn: 0}

	body := WithdrawRequest{OrderNumber: "4561261212345467", Sum: 40}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/user/balance/withdraw", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.Withdraw(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	// Доп проверка баланса после списания
	if svc.balances[1].Current != 60 || svc.balances[1].Withdrawn != 40 {
		t.Error("Balance fields didn't update inside fake service state")
	}
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	svc.balances[1] = domain.Balance{Current: 10, Withdrawn: 0}

	body := WithdrawRequest{OrderNumber: "4561261212345467", Sum: 50}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/user/balance/withdraw", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.Withdraw(w, req.WithContext(ctx))

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("Expected 402 Payment Required, got %d", w.Code)
	}
}

// --- GET WITHDRAWALS TESTS ---

func TestGetWithdrawalsEmpty(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	req := httptest.NewRequest("GET", "/api/user/withdrawals", nil)
	w := httptest.NewRecorder()

	h.GetWithdrawals(w, req.WithContext(ctx))

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204 No Content, got %d", w.Code)
	}
}

func TestGetWithdrawalsWithContent(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	svc.withdrawals[1] = []domain.Withdrawal{
		{OrderNumber: "4561261212345467", Sum: 100, ProcessedAt: time.Now()},
	}

	req := httptest.NewRequest("GET", "/api/user/withdrawals", nil)
	w := httptest.NewRecorder()

	h.GetWithdrawals(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	var resp []WithdrawalResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if len(resp) != 1 || resp[0].OrderNumber != "4561261212345467" || resp[0].Sum != 100 {
		t.Error("Withdrawal response fields array parsing mismatch")
	}
}

// --- EXPORT STATS TESTS ---

func TestExportStatsSuccess(t *testing.T) {
	svc := newFakeService()
	h := New(svc)

	ctx := context.WithValue(context.Background(), CtxKeyUserID, int64(1))
	svc.balances[1] = domain.Balance{Current: 50, Withdrawn: 50}
	svc.orders["4561261212345467"] = &domain.Order{
		UserID: 1, Number: "4561261212345467", Status: domain.OrderStatusProcessed, Accrual: 100,
	}

	req := httptest.NewRequest("POST", "/api/stats/export", nil)
	w := httptest.NewRecorder()

	// Безопасное удаление тестового артефакта с диска
	t.Cleanup(func() {
		_ = os.Remove("stats.txt")
	})

	h.ExportStats(w, req.WithContext(ctx))

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}

	// Проверяем физическое существование файла на диске
	if _, err := os.Stat("stats.txt"); os.IsNotExist(err) {
		t.Error("Expected stats.txt file to be written into project root directory")
	}
}
