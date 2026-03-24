// Пакет store реализует хранилище данных в памяти.
// Используйте отдельные мьютексы для независимых групп данных.
// Реализуйте этот пакет самостоятельно.
package store

import (
	"gopherledger/internal/domain"
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
}

// New создаёт и возвращает новое пустое хранилище.
func New() *Store {
	panic("не реализовано")
}

// CreateUser добавляет нового пользователя.
// Возвращает domain.ErrUserExists если логин уже занят.
func (s *Store) CreateUser(login, passwordHash string) (*domain.User, error) {
	panic("не реализовано")
}

// GetUserByLogin возвращает пользователя по логину.
// Возвращает domain.ErrUserNotFound если пользователь не найден.
func (s *Store) GetUserByLogin(login string) (*domain.User, error) {
	panic("не реализовано")
}

// CreateOrder добавляет новый заказ для пользователя.
// Возвращает domain.ErrOrderOwnedByUser если этот пользователь уже загружал этот номер.
// Возвращает domain.ErrOrderExists если номер принадлежит другому пользователю.
func (s *Store) CreateOrder(userID int64, number string) (*domain.Order, error) {
	panic("не реализовано")
}

// GetUserOrders возвращает все заказы пользователя, сначала новые.
func (s *Store) GetUserOrders(userID int64) ([]domain.Order, error) {
	panic("не реализовано")
}

// GetOrdersForProcessing возвращает все заказы в статусе NEW или PROCESSING.
func (s *Store) GetOrdersForProcessing() ([]domain.Order, error) {
	panic("не реализовано")
}

// UpdateOrderStatus обновляет статус и начисление заказа.
// Если статус PROCESSED и accrual > 0, баланс пользователя пополняется.
func (s *Store) UpdateOrderStatus(number, status string, accrual float64) error {
	panic("не реализовано")
}

// GetBalance возвращает баланс пользователя.
func (s *Store) GetBalance(userID int64) (domain.Balance, error) {
	panic("не реализовано")
}

// Withdraw списывает сумму с баланса и записывает операцию.
// Возвращает domain.ErrInsufficientFunds если баланса не хватает.
// Обе операции должны быть атомарны: либо обе успешны, либо ни одна.
func (s *Store) Withdraw(userID int64, orderNumber string, sum float64) error {
	panic("не реализовано")
}

// GetWithdrawals возвращает историю списаний пользователя, сначала новые.
func (s *Store) GetWithdrawals(userID int64) ([]domain.Withdrawal, error) {
	panic("не реализовано")
}