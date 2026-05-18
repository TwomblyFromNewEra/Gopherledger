package service

import (
	"testing"

	"gopherledger/internal/domain"
)

type mockRepository struct {
	users       map[string]*domain.User
	orders      map[string]*domain.Order
	balances    map[int64]*domain.Balance
	withdrawals map[int64][]*domain.Withdrawal
	userIDSeq   int64
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:       make(map[string]*domain.User),
		orders:      make(map[string]*domain.Order),
		balances:    make(map[int64]*domain.Balance),
		withdrawals: make(map[int64][]*domain.Withdrawal),
	}
}

func (m *mockRepository) CreateUser(login, passwordHash string) (*domain.User, error) {
	if _, exists := m.users[login]; exists {
		return nil, domain.ErrUserExists
	}
	m.userIDSeq++
	user := &domain.User{
		ID:           m.userIDSeq,
		Login:        login,
		PasswordHash: passwordHash,
	}
	m.users[login] = user
	m.balances[user.ID] = &domain.Balance{Current: 0, Withdrawn: 0}
	return user, nil
}

func (m *mockRepository) GetUserByLogin(login string) (*domain.User, error) {
	user, exists := m.users[login]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) CreateOrder(userID int64, number string) (*domain.Order, error) {
	if order, exists := m.orders[number]; exists {
		if order.UserID == userID {
			return order, domain.ErrOrderOwnedByUser
		}
		return order, domain.ErrOrderExists
	}
	order := &domain.Order{
		ID:     int64(len(m.orders)) + 1,
		UserID: userID,
		Number: number,
		Status: domain.OrderStatusNew,
	}
	m.orders[number] = order
	return order, nil
}

func (m *mockRepository) GetUserOrders(userID int64) ([]domain.Order, error) {
	var result []domain.Order
	for _, order := range m.orders {
		if order.UserID == userID {
			result = append(result, *order)
		}
	}
	return result, nil
}

func (m *mockRepository) GetOrdersForProcessing() ([]domain.Order, error) {
	var result []domain.Order
	for _, order := range m.orders {
		if order.Status == domain.OrderStatusNew || order.Status == domain.OrderStatusProcessing {
			result = append(result, *order)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdateOrderStatus(number, status string, accrual float64) error {
	if order, exists := m.orders[number]; exists {
		order.Status = status
		order.Accrual = accrual
		return nil
	}
	return domain.ErrInvalidOrder
}

func (m *mockRepository) GetBalance(userID int64) (domain.Balance, error) {
	balance, exists := m.balances[userID]
	if !exists {
		return domain.Balance{}, domain.ErrUserNotFound
	}
	return *balance, nil
}

func (m *mockRepository) Withdraw(userID int64, orderNumber string, sum float64) error {
	balance, exists := m.balances[userID]
	if !exists {
		return domain.ErrUserNotFound
	}
	if balance.Current < sum {
		return domain.ErrInsufficientFunds
	}
	balance.Current -= sum
	balance.Withdrawn += sum
	return nil
}

func (m *mockRepository) GetWithdrawals(userID int64) ([]domain.Withdrawal, error) {
	withdrawals, exists := m.withdrawals[userID]
	if !exists {
		return []domain.Withdrawal{}, nil
	}
	result := make([]domain.Withdrawal, len(withdrawals))
	for i, w := range withdrawals {
		result[i] = *w
	}
	return result, nil
}

func TestRegisterUserSuccess(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	token, err := svc.RegisterUser("user1", "password123")
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}
	if token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestRegisterUserDuplicate(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	svc.RegisterUser("user1", "password123")
	_, err := svc.RegisterUser("user1", "password456")

	if err != domain.ErrUserExists {
		t.Errorf("Expected ErrUserExists, got %v", err)
	}
}

func TestLoginUserSuccess(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	svc.RegisterUser("user1", "password123")
	token, err := svc.LoginUser("user1", "password123")

	if err != nil {
		t.Fatalf("LoginUser failed: %v", err)
	}
	if token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestLoginUserInvalidPassword(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	svc.RegisterUser("user1", "password123")
	_, err := svc.LoginUser("user1", "wrongpassword")

	if err != domain.ErrInvalidPassword {
		t.Errorf("Expected ErrInvalidPassword, got %v", err)
	}
}

func TestLoginUserNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	_, err := svc.LoginUser("nonexistent", "password123")

	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestCreateOrderSuccess(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	order, err := svc.CreateOrder(1, "4561261212345467")
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}
	if order.Number != "4561261212345467" {
		t.Errorf("Expected 4561261212345467, got %s", order.Number)
	}
}

func TestCreateOrderInvalidLuhn(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	_, err := svc.CreateOrder(1, "1111111111111111")

	if err != domain.ErrInvalidOrder {
		t.Errorf("Expected ErrInvalidOrder, got %v", err)
	}
}

func TestCreateOrderDuplicate(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	svc.CreateOrder(1, "4561261212345467")
	_, err := svc.CreateOrder(1, "4561261212345467")

	if err != domain.ErrOrderOwnedByUser {
		t.Errorf("Expected ErrOrderOwnedByUser, got %v", err)
	}
}

func TestCreateOrderConflict(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	svc.CreateOrder(1, "4561261212345467")
	_, err := svc.CreateOrder(2, "4561261212345467")

	if err != domain.ErrOrderExists {
		t.Errorf("Expected ErrOrderExists, got %v", err)
	}
}

func TestGetUserOrders(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	// Первый валидный номер
	if _, err := svc.CreateOrder(1, "4561261212345467"); err != nil {
		t.Fatalf("Failed to create first order: %v", err)
	}

	// Второй валидный номер (Исправлено!)
	if _, err := svc.CreateOrder(1, "1234567812345670"); err != nil {
		t.Fatalf("Failed to create second order: %v", err)
	}

	orders, err := svc.GetUserOrders(1)
	if err != nil {
		t.Fatalf("GetUserOrders failed: %v", err)
	}

	if len(orders) != 2 {
		t.Errorf("Expected 2 orders, got %d", len(orders))
	}
}

func TestGetBalance(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	repo.balances[1] = &domain.Balance{
		Current:   0,
		Withdrawn: 0,
	}

	balance, err := svc.GetBalance(1)
	if err != nil {
		t.Fatalf("GetBalance failed: %v", err)
	}

	if balance.Current != 0 {
		t.Errorf("Expected current=0, got %f", balance.Current)
	}
}

func TestWithdrawSuccess(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	repo.balances[1] = &domain.Balance{Current: 100, Withdrawn: 0}

	err := svc.Withdraw(1, "4561261212345467", 50)
	if err != nil {
		t.Fatalf("Withdraw failed: %v", err)
	}
}

func TestWithdrawInsufficientFunds(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	repo.balances[1] = &domain.Balance{Current: 10, Withdrawn: 0}

	err := svc.Withdraw(1, "4561261212345467", 50)
	if err != domain.ErrInsufficientFunds {
		t.Errorf("Expected ErrInsufficientFunds, got %v", err)
	}
}

func TestWithdrawInvalidOrder(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	repo.balances[1] = &domain.Balance{Current: 100, Withdrawn: 0}

	err := svc.Withdraw(1, "1111111111111111", 50)
	if err != domain.ErrInvalidOrder {
		t.Errorf("Expected ErrInvalidOrder, got %v", err)
	}
}

func TestGetWithdrawals(t *testing.T) {
	repo := newMockRepository()
	svc := New(repo)

	withdrawals, err := svc.GetWithdrawals(1)
	if err != nil {
		t.Fatalf("GetWithdrawals failed: %v", err)
	}
	if len(withdrawals) != 0 {
		t.Errorf("Expected 0 withdrawals, got %d", len(withdrawals))
	}
}
