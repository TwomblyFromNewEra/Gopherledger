package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"gopherledger/internal/domain"
	"reflect"
	"sync"
	"testing"
	"time"
)

// mockRepository - mock для тестирования
type mockRepository struct {
	createUserFn             func(login, passwordHash string) (*domain.User, error)
	getUserByLoginFn         func(login string) (*domain.User, error)
	createOrderFn            func(userID int64, number string) (*domain.Order, error)
	getUserOrdersFn          func(userID int64) ([]domain.Order, error)
	getOrdersForProcessingFn func() ([]domain.Order, error)
	updateOrderStatusFn      func(number, status string, accrual float64) error
	getBalanceFn             func(userID int64) (domain.Balance, error)
	withdrawFn               func(userID int64, orderNumber string, sum float64) error
	getWithdrawalsFn         func(userID int64) ([]domain.Withdrawal, error)
}

func (m *mockRepository) CreateUser(login, passwordHash string) (*domain.User, error) {
	if m.createUserFn != nil {
		return m.createUserFn(login, passwordHash)
	}
	return nil, nil
}

func (m *mockRepository) GetUserByLogin(login string) (*domain.User, error) {
	if m.getUserByLoginFn != nil {
		return m.getUserByLoginFn(login)
	}
	return nil, nil
}

func (m *mockRepository) CreateOrder(userID int64, number string) (*domain.Order, error) {
	if m.createOrderFn != nil {
		return m.createOrderFn(userID, number)
	}
	return nil, nil
}

func (m *mockRepository) GetUserOrders(userID int64) ([]domain.Order, error) {
	if m.getUserOrdersFn != nil {
		return m.getUserOrdersFn(userID)
	}
	return nil, nil
}

func (m *mockRepository) GetOrdersForProcessing() ([]domain.Order, error) {
	if m.getOrdersForProcessingFn != nil {
		return m.getOrdersForProcessingFn()
	}
	return nil, nil
}

func (m *mockRepository) UpdateOrderStatus(number, status string, accrual float64) error {
	if m.updateOrderStatusFn != nil {
		return m.updateOrderStatusFn(number, status, accrual)
	}
	return nil
}

func (m *mockRepository) GetBalance(userID int64) (domain.Balance, error) {
	if m.getBalanceFn != nil {
		return m.getBalanceFn(userID)
	}
	return domain.Balance{}, nil
}

func (m *mockRepository) Withdraw(userID int64, orderNumber string, sum float64) error {
	if m.withdrawFn != nil {
		return m.withdrawFn(userID, orderNumber, sum)
	}
	return nil
}

func (m *mockRepository) GetWithdrawals(userID int64) ([]domain.Withdrawal, error) {
	if m.getWithdrawalsFn != nil {
		return m.getWithdrawalsFn(userID)
	}
	return nil, nil
}

func TestNew(t *testing.T) {
	type args struct {
		repo Repository
	}
	tests := []struct {
		name string
		args args
		want *Service
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.repo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_CreateOrder(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		userID int64
		number string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.Order
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			got, err := s.CreateOrder(tt.args.userID, tt.args.number)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateOrder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_GetBalance(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		userID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.Balance
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			got, err := s.GetBalance(tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBalance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBalance() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_GetUserOrders(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		userID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domain.Order
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			got, err := s.GetUserOrders(tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserOrders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUserOrders() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_GetWithdrawals(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		userID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domain.Withdrawal
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			got, err := s.GetWithdrawals(tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWithdrawals() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWithdrawals() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_LoginUser(t *testing.T) {
	password := "password123"
	hash := sha256.Sum256([]byte(password))
	hashString := hex.EncodeToString(hash[:])

	tests := []struct {
		name     string
		login    string
		password string
		mockFn   func(login string) (*domain.User, error)
		wantErr  bool
	}{
		{
			name:     "успешная авторизация",
			login:    "testuser",
			password: password,
			mockFn: func(login string) (*domain.User, error) {
				return &domain.User{ID: 1, Login: login, PasswordHash: hashString}, nil
			},
			wantErr: false,
		},
		{
			name:     "неверный пароль",
			login:    "testuser",
			password: "wrongpass",
			mockFn: func(login string) (*domain.User, error) {
				return &domain.User{ID: 1, Login: login, PasswordHash: hashString}, nil
			},
			wantErr: true,
		},
		{
			name:     "пользователь не найден",
			login:    "unknown",
			password: password,
			mockFn: func(login string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{repo: &mockRepository{getUserByLoginFn: tt.mockFn}}
			got, err := s.LoginUser(tt.login, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoginUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("LoginUser() получен пустой токен")
			}
		})
	}
}

func TestService_RegisterUser(t *testing.T) {
	tests := []struct {
		name     string
		login    string
		password string
		mockFn   func(login, passwordHash string) (*domain.User, error)
		wantErr  bool
	}{
		{
			name:     "успешная регистрация",
			login:    "testuser",
			password: "password123",
			mockFn: func(login, passwordHash string) (*domain.User, error) {
				return &domain.User{ID: 1, Login: login, PasswordHash: passwordHash}, nil
			},
			wantErr: false,
		},
		{
			name:     "пользователь уже существует",
			login:    "testuser",
			password: "password123",
			mockFn: func(login, passwordHash string) (*domain.User, error) {
				return nil, domain.ErrUserExists
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{repo: &mockRepository{createUserFn: tt.mockFn}}
			got, err := s.RegisterUser(tt.login, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("RegisterUser() получен пустой токен")
			}
		})
	}
}

func TestService_Withdraw(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		userID      int64
		orderNumber string
		sum         float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			if err := s.Withdraw(tt.args.userID, tt.args.orderNumber, tt.args.sum); (err != nil) != tt.wantErr {
				t.Errorf("Withdraw() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_markProcessing(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		number string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			s.markProcessing(tt.args.number)
		})
	}
}

func TestService_processAllPendingOrders(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			s.processAllPendingOrders(tt.args.ctx)
		})
	}
}

func TestService_processOrder(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		ctx    context.Context
		number string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			s.processOrder(tt.args.ctx, tt.args.number)
		})
	}
}

func TestService_unmarkProcessing(t *testing.T) {
	type fields struct {
		repo             Repository
		processingOrders map[string]bool
		muProcessing     sync.RWMutex
	}
	type args struct {
		number string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				repo:             tt.fields.repo,
				processingOrders: tt.fields.processingOrders,
				muProcessing:     tt.fields.muProcessing,
			}
			s.unmarkProcessing(tt.args.number)
		})
	}
}

func Test_isInvalid(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInvalid(); got != tt.want {
				t.Errorf("isInvalid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_randomAccrual(t *testing.T) {
	tests := []struct {
		name string
		want float64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := randomAccrual(); got != tt.want {
				t.Errorf("randomAccrual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_randomDelay(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := randomDelay(); got != tt.want {
				t.Errorf("randomDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateLuhn(t *testing.T) {
	type args struct {
		number string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateLuhn(tt.args.number); got != tt.want {
				t.Errorf("validateLuhn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_isProcessing(t *testing.T) {
	s := New(&mockRepository{})

	s.markProcessing("order1")

	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{
			name:   "заказ в обработке",
			number: "order1",
			want:   true,
		},
		{
			name:   "заказ не в обработке",
			number: "order2",
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.isProcessing(tt.number); got != tt.want {
				t.Errorf("isProcessing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_markAndUnmarkProcessing(t *testing.T) {
	s := New(&mockRepository{})

	t.Run("отметить в обработке", func(t *testing.T) {
		s.markProcessing("order1")
		if !s.isProcessing("order1") {
			t.Errorf("заказ должен быть отмечен как в обработке")
		}
	})

	t.Run("снять отметку", func(t *testing.T) {
		s.unmarkProcessing("order1")
		if s.isProcessing("order1") {
			t.Errorf("заказ должен быть снят с обработки")
		}
	})
}

func TestService_StartAccrualWorker(t *testing.T) {
	t.Run("воркер запускается и останавливается", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		s := New(&mockRepository{
			getOrdersForProcessingFn: func() ([]domain.Order, error) {
				return []domain.Order{}, nil
			},
		})

		// Воркер должен завершиться после отмены контекста
		s.StartAccrualWorker(ctx)

		// Если мы здесь, то воркер корректно остановился
		if ctx.Err() == nil {
			t.Error("контекст должен быть отменен")
		}
	})
}
