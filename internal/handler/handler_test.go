package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gopherledger/internal/domain"
)

// mockService - простой mock для тестирования
type mockService struct {
	registerFn       func(login, password string) (string, error)
	loginFn          func(login, password string) (string, error)
	createOrderFn    func(userID int64, number string) (*domain.Order, error)
	getUserOrdersFn  func(userID int64) ([]domain.Order, error)
	getBalanceFn     func(userID int64) (domain.Balance, error)
	withdrawFn       func(userID int64, orderNumber string, sum float64) error
	getWithdrawalsFn func(userID int64) ([]domain.Withdrawal, error)
}

func (m *mockService) RegisterUser(login, password string) (string, error) {
	if m.registerFn != nil {
		return m.registerFn(login, password)
	}
	return "", nil
}

func (m *mockService) LoginUser(login, password string) (string, error) {
	if m.loginFn != nil {
		return m.loginFn(login, password)
	}
	return "", nil
}

func (m *mockService) CreateOrder(userID int64, number string) (*domain.Order, error) {
	if m.createOrderFn != nil {
		return m.createOrderFn(userID, number)
	}
	return nil, nil
}

func (m *mockService) GetUserOrders(userID int64) ([]domain.Order, error) {
	if m.getUserOrdersFn != nil {
		return m.getUserOrdersFn(userID)
	}
	return nil, nil
}

func (m *mockService) GetBalance(userID int64) (domain.Balance, error) {
	if m.getBalanceFn != nil {
		return m.getBalanceFn(userID)
	}
	return domain.Balance{}, nil
}

func (m *mockService) Withdraw(userID int64, orderNumber string, sum float64) error {
	if m.withdrawFn != nil {
		return m.withdrawFn(userID, orderNumber, sum)
	}
	return nil
}

func (m *mockService) GetWithdrawals(userID int64) ([]domain.Withdrawal, error) {
	if m.getWithdrawalsFn != nil {
		return m.getWithdrawalsFn(userID)
	}
	return nil, nil
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           LoginRequest
		mockFn         func(login, password string) (string, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "успешная регистрация",
			body: LoginRequest{Login: "user1", Password: "pass123"},
			mockFn: func(login, password string) (string, error) {
				return "token123", nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "пользователь уже существует",
			body: LoginRequest{Login: "user1", Password: "pass123"},
			mockFn: func(login, password string) (string, error) {
				return "", domain.ErrUserExists
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "некорректные данные - пустой логин",
			body:           LoginRequest{Login: "", Password: "pass123"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "некорректные данные - пустой пароль",
			body:           LoginRequest{Login: "user1", Password: ""},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{svc: &mockService{registerFn: tt.mockFn}}
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ожидаемый статус %d, получен %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		body           LoginRequest
		mockFn         func(login, password string) (string, error)
		expectedStatus int
	}{
		{
			name: "успешная авторизация",
			body: LoginRequest{Login: "user1", Password: "pass123"},
			mockFn: func(login, password string) (string, error) {
				return "token123", nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "неверный логин или пароль",
			body: LoginRequest{Login: "user1", Password: "wrong"},
			mockFn: func(login, password string) (string, error) {
				return "", domain.ErrInvalidPassword
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "пользователь не найден",
			body: LoginRequest{Login: "unknown", Password: "pass"},
			mockFn: func(login, password string) (string, error) {
				return "", domain.ErrUserNotFound
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{svc: &mockService{loginFn: tt.mockFn}}
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ожидаемый статус %d, получен %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		orderNumber    string
		userID         int64
		mockFn         func(userID int64, number string) (*domain.Order, error)
		expectedStatus int
	}{
		{
			name:        "успешное создание заказа",
			orderNumber: "4561261212345467",
			userID:      1,
			mockFn: func(userID int64, number string) (*domain.Order, error) {
				return &domain.Order{
					Number:     number,
					Status:     "NEW",
					Accrual:    0,
					UploadedAt: time.Now(),
				}, nil
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:        "заказ уже загружен этим пользователем",
			orderNumber: "4561261212345467",
			userID:      1,
			mockFn: func(userID int64, number string) (*domain.Order, error) {
				return nil, domain.ErrOrderOwnedByUser
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "заказ принадлежит другому пользователю",
			orderNumber: "4561261212345467",
			userID:      1,
			mockFn: func(userID int64, number string) (*domain.Order, error) {
				return nil, domain.ErrOrderExists
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:        "неверный номер заказа (проверка Луна)",
			orderNumber: "1234567890123456",
			userID:      1,
			mockFn: func(userID int64, number string) (*domain.Order, error) {
				return nil, domain.ErrInvalidOrder
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "пустой номер заказа",
			orderNumber:    "",
			userID:         1,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{svc: &mockService{createOrderFn: tt.mockFn}}
			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.orderNumber))
			req = req.WithContext(context.WithValue(req.Context(), CtxKeyUserID, tt.userID))
			w := httptest.NewRecorder()

			h.CreateOrder(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ожидаемый статус %d, получен %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetOrders(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		mockFn         func(userID int64) ([]domain.Order, error)
		expectedStatus int
	}{
		{
			name:   "заказы найдены",
			userID: 1,
			mockFn: func(userID int64) ([]domain.Order, error) {
				return []domain.Order{
					{Number: "4561261212345467", Status: "NEW", UploadedAt: time.Now()},
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "заказов нет",
			userID: 1,
			mockFn: func(userID int64) ([]domain.Order, error) {
				return []domain.Order{}, nil
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{svc: &mockService{getUserOrdersFn: tt.mockFn}}
			req := httptest.NewRequest(http.MethodGet, "/orders", nil)
			req = req.WithContext(context.WithValue(req.Context(), CtxKeyUserID, tt.userID))
			w := httptest.NewRecorder()

			h.GetOrders(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ожидаемый статус %d, получен %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetBalance(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		mockFn         func(userID int64) (domain.Balance, error)
		expectedStatus int
	}{
		{
			name:   "баланс получен",
			userID: 1,
			mockFn: func(userID int64) (domain.Balance, error) {
				return domain.Balance{Current: 100.5, Withdrawn: 50.0}, nil
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{svc: &mockService{getBalanceFn: tt.mockFn}}
			req := httptest.NewRequest(http.MethodGet, "/balance", nil)
			req = req.WithContext(context.WithValue(req.Context(), CtxKeyUserID, tt.userID))
			w := httptest.NewRecorder()

			h.GetBalance(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ожидаемый статус %d, получен %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_Withdraw(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		withdrawReq    WithdrawRequest
		mockFn         func(userID int64, orderNumber string, sum float64) error
		expectedStatus int
	}{
		{
			name:        "успешное списание",
			userID:      1,
			withdrawReq: WithdrawRequest{OrderNumber: "4561261212345467", Sum: 50.0},
			mockFn: func(userID int64, orderNumber string, sum float64) error {
				return nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "недостаточно баллов",
			userID:      1,
			withdrawReq: WithdrawRequest{OrderNumber: "4561261212345467", Sum: 1000.0},
			mockFn: func(userID int64, orderNumber string, sum float64) error {
				return domain.ErrInsufficientFunds
			},
			expectedStatus: http.StatusPaymentRequired,
		},
		{
			name:        "неверный номер заказа",
			userID:      1,
			withdrawReq: WithdrawRequest{OrderNumber: "1234", Sum: 50.0},
			mockFn: func(userID int64, orderNumber string, sum float64) error {
				return domain.ErrInvalidOrder
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "некорректные данные - пустой номер",
			userID:         1,
			withdrawReq:    WithdrawRequest{OrderNumber: "", Sum: 50.0},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "некорректные данные - нулевая сумма",
			userID:         1,
			withdrawReq:    WithdrawRequest{OrderNumber: "4561261212345467", Sum: 0},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{svc: &mockService{withdrawFn: tt.mockFn}}
			body, _ := json.Marshal(tt.withdrawReq)
			req := httptest.NewRequest(http.MethodPost, "/withdraw", bytes.NewReader(body))
			req = req.WithContext(context.WithValue(req.Context(), CtxKeyUserID, tt.userID))
			w := httptest.NewRecorder()

			h.Withdraw(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ожидаемый статус %d, получен %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetWithdrawals(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		mockFn         func(userID int64) ([]domain.Withdrawal, error)
		expectedStatus int
	}{
		{
			name:   "списания найдены",
			userID: 1,
			mockFn: func(userID int64) ([]domain.Withdrawal, error) {
				return []domain.Withdrawal{
					{OrderNumber: "4561261212345467", Sum: 50.0, ProcessedAt: time.Now()},
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "списаний нет",
			userID: 1,
			mockFn: func(userID int64) ([]domain.Withdrawal, error) {
				return []domain.Withdrawal{}, nil
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{svc: &mockService{getWithdrawalsFn: tt.mockFn}}
			req := httptest.NewRequest(http.MethodGet, "/withdrawals", nil)
			req = req.WithContext(context.WithValue(req.Context(), CtxKeyUserID, tt.userID))
			w := httptest.NewRecorder()

			h.GetWithdrawals(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("ожидаемый статус %d, получен %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestUserIDFromContext(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		want  int64
		want1 bool
	}{
		{
			name:  "userID найден",
			ctx:   context.WithValue(context.Background(), CtxKeyUserID, int64(42)),
			want:  42,
			want1: true,
		},
		{
			name:  "userID не найден",
			ctx:   context.Background(),
			want:  0,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := UserIDFromContext(tt.ctx)
			if got != tt.want {
				t.Errorf("UserIDFromContext() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("UserIDFromContext() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_writeError(t *testing.T) {
	type args struct {
		w           http.ResponseWriter
		status      int
		code        string
		userMsg     string
		internalErr error
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeError(tt.args.w, tt.args.status, tt.args.code, tt.args.userMsg, tt.args.internalErr)
		})
	}
}

func Test_writeJSON(t *testing.T) {
	type args struct {
		w      http.ResponseWriter
		status int
		v      any
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeJSON(tt.args.w, tt.args.status, tt.args.v)
		})
	}
}
