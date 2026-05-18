// Пакет store реализует хранилище данных в памяти.
// Используйте отдельные мьютексы для независимых групп данных.
// Реализуйте этот пакет самостоятельно.
package store

import (
	"gopherledger/internal/domain"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Store хранит все данные приложения в памяти.
// Добавьте средства защиты конкурентного доступа самостоятельно.
type Store struct {
	// users хранит пользователей по их ID
	users map[int64]*domain.User

	// usersByLogin хранит пользователей по логину - для быстрого поиска при авторизации
	usersByLogin map[string]*domain.User

	// orders хранит заказы по номеру заказа
	orders map[string]*domain.Order

	// balances хранит текущий баланс каждого пользователя по его ID
	balances map[int64]*domain.Balance

	// withdrawals хранит историю списаний для каждого пользователя по его ID
	withdrawals map[int64][]*domain.Withdrawal

	// nextID используется для генерации уникальных числовых ID
	nextID int64

	muUsers       sync.RWMutex
	muOrders      sync.RWMutex
	muBalances    sync.RWMutex
	muWithdrawals sync.RWMutex
}

// New создаёт и возвращает новое пустое хранилище.
func New() *Store {
	return &Store{
		users:        make(map[int64]*domain.User),
		usersByLogin: make(map[string]*domain.User),
		orders:       make(map[string]*domain.Order),
		balances:     make(map[int64]*domain.Balance),
		withdrawals:  make(map[int64][]*domain.Withdrawal),
	}
}

func (s *Store) next() int64 {
	return atomic.AddInt64(&s.nextID, 1)
}

// CreateUser добавляет нового пользователя.
// Возвращает domain.ErrUserExists если логин уже занят.
func (s *Store) CreateUser(login, passwordHash string) (*domain.User, error) {
	s.muUsers.Lock()
	defer s.muUsers.Unlock()

	if _, ok := s.usersByLogin[login]; ok {
		return nil, domain.ErrUserExists
	}
	id := s.next()
	user := &domain.User{
		ID:           id,
		Login:        login,
		PasswordHash: passwordHash,
	}
	s.users[id] = user
	s.usersByLogin[login] = user

	s.muBalances.Lock()
	if _, ok := s.balances[id]; !ok {
		s.balances[id] = &domain.Balance{}
	}
	s.muBalances.Unlock()

	return user, nil
}

// GetUserByLogin возвращает пользователя по логину.
// Возвращает domain.ErrUserNotFound если пользователь не найден.
func (s *Store) GetUserByLogin(login string) (*domain.User, error) {
	s.muUsers.RLock()
	defer s.muUsers.RUnlock()

	user, ok := s.usersByLogin[login]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

// CreateOrder добавляет новый заказ для пользователя.
// Возвращает domain.ErrOrderOwnedByUser если этот пользователь уже загружал этот номер.
// Возвращает domain.ErrOrderExists если номер принадлежит другому пользователю.
func (s *Store) CreateOrder(userID int64, number string) (*domain.Order, error) {
	s.muOrders.Lock()

	defer s.muOrders.Unlock()

	if existing, ok := s.orders[number]; ok {
		if existing.UserID == userID {
			return existing, domain.ErrOrderOwnedByUser
		}
		return existing, domain.ErrOrderExists
	}

	id := s.next()
	order := &domain.Order{
		ID:         id,
		UserID:     userID,
		Number:     number,
		Status:     domain.OrderStatusNew,
		Accrual:    0,
		UploadedAt: time.Now(),
	}

	s.orders[number] = order

	s.muBalances.Lock()
	if _, ok := s.balances[userID]; !ok {
		s.balances[userID] = &domain.Balance{}
	}
	s.muBalances.Unlock()

	return order, nil
}

// GetUserOrders возвращает все заказы пользователя, сначала новые.
func (s *Store) GetUserOrders(userID int64) ([]domain.Order, error) {
	s.muOrders.RLock()
	defer s.muOrders.RUnlock()

	res := make([]domain.Order, 0, len(s.orders))
	for _, order := range s.orders {
		if order.UserID == userID {
			res = append(res, *order)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].UploadedAt.After(res[j].UploadedAt)
	})

	return res, nil
}

// GetOrdersForProcessing возвращает все заказы в статусе NEW или PROCESSING.
func (s *Store) GetOrdersForProcessing() ([]domain.Order, error) {
	s.muOrders.RLock()
	defer s.muOrders.RUnlock()

	res := make([]domain.Order, 0, len(s.orders))
	for _, order := range s.orders {
		if order.Status == domain.OrderStatusProcessing || order.Status == domain.OrderStatusNew {
			res = append(res, *order)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].UploadedAt.After(res[j].UploadedAt)
	})
	return res, nil
}

// UpdateOrderStatus обновляет статус и начисление заказа.
// Если статус PROCESSED и accrual > 0, баланс пользователя пополняется.
func (s *Store) UpdateOrderStatus(number, status string, accrual float64) error {
	s.muOrders.Lock()
	defer s.muOrders.Unlock()

	order, ok := s.orders[number]
	if !ok {
		return domain.ErrInvalidOrder
	}

	order.Status = status
	order.Accrual = accrual

	if status == domain.OrderStatusProcessed && accrual > 0 {
		s.muBalances.Lock()
		balance, ok := s.balances[order.UserID]
		if !ok {
			balance = &domain.Balance{}
			s.balances[order.UserID] = balance
		}
		balance.Current += accrual
		s.muBalances.Unlock()
	}

	order.Status = status
	order.Accrual = accrual

	return nil
}

// GetBalance возвращает баланс пользователя.
func (s *Store) GetBalance(userID int64) (domain.Balance, error) {
	s.muBalances.RLock()
	defer s.muBalances.RUnlock()

	balance, ok := s.balances[userID]
	if !ok {
		return domain.Balance{}, domain.ErrUserNotFound
	}

	res := domain.Balance{
		Current:   balance.Current,
		Withdrawn: balance.Withdrawn,
	}

	return res, nil

}

// Withdraw списывает сумму с баланса и записывает операцию.
// Возвращает domain.ErrInsufficientFunds если баланса не хватает.
// Обе операции должны быть атомарны: либо обе успешны, либо ни одна.
func (s *Store) Withdraw(userID int64, orderNumber string, sum float64) error {
	s.muBalances.Lock()
	defer s.muBalances.Unlock()

	balance, ok := s.balances[userID]
	if !ok {
		return domain.ErrInsufficientFunds
	}

	if balance.Current < sum {
		return domain.ErrInsufficientFunds
	}
	balance.Current -= sum
	balance.Withdrawn += sum

	s.muWithdrawals.Lock()

	id := s.next()
	with := &domain.Withdrawal{
		ID:          id,
		UserID:      userID,
		OrderNumber: orderNumber,
		Sum:         sum,
		ProcessedAt: time.Now(),
	}
	s.withdrawals[userID] = append(s.withdrawals[userID], with)
	s.muWithdrawals.Unlock()

	return nil
}

// GetWithdrawals возвращает историю списаний пользователя, сначала новые.
func (s *Store) GetWithdrawals(userID int64) ([]domain.Withdrawal, error) {
	s.muWithdrawals.RLock()
	defer s.muWithdrawals.RUnlock()

	res := make([]domain.Withdrawal, 0, len(s.withdrawals))

	userWithdrawals, ok := s.withdrawals[userID]
	if !ok || len(userWithdrawals) == 0 {
		return nil, nil
	}

	for _, withdrawal := range userWithdrawals {
		res = append(res, *withdrawal)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ProcessedAt.After(res[j].ProcessedAt)
	})

	return res, nil
}
