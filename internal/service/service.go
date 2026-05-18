// Пакет service содержит бизнес-логику приложения.
//
// Взаимодействие с хранилищем осуществляется через интерфейс.
// Определите этот интерфейс здесь, по месту использования.

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"gopherledger/internal/auth"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"gopherledger/internal/domain"

	"golang.org/x/sync/errgroup"
)

// Service реализует бизнес-логику приложения.
// Замените поле repo в структуре на свой интерфейс.
//
// processingOrders хранит номера заказов, которые сейчас обрабатываются воркером.
// Защитите конкурентный доступ к этому полю самостоятельно.

type Repository interface {
	CreateUser(login, passwordHash string) (*domain.User, error)
	GetUserByLogin(login string) (*domain.User, error)
	CreateOrder(userID int64, number string) (*domain.Order, error)
	GetUserOrders(userID int64) ([]domain.Order, error)
	GetOrdersForProcessing() ([]domain.Order, error)
	UpdateOrderStatus(number, status string, accrual float64) error
	GetBalance(userID int64) (domain.Balance, error)
	Withdraw(userID int64, orderNumber string, sum float64) error
	GetWithdrawals(userID int64) ([]domain.Withdrawal, error)
}

type Service struct {
	repo             Repository
	processingOrders map[string]bool
	muProcessing     sync.RWMutex
}

func New(repo Repository) *Service {
	return &Service{
		repo:             repo,
		processingOrders: make(map[string]bool),
	}
}

// ---------------------------------------------------------------------------
// Методы бизнес-логики - реализуйте самостоятельно
// ---------------------------------------------------------------------------

// RegisterUser регистрирует нового пользователя и возвращает токен аутентификации.
// Хешируйте пароль перед сохранением с помощью crypto/sha256.
func (s *Service) RegisterUser(login, password string) (string, error) {
	hash := sha256.Sum256([]byte(password))
	hashString := hex.EncodeToString(hash[:])

	user, err := s.repo.CreateUser(login, hashString)
	if err != nil {
		return "", err
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// LoginUser проверяет учётные данные и возвращает токен аутентификации.
func (s *Service) LoginUser(login, password string) (string, error) {
	user, err := s.repo.GetUserByLogin(login)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(password))
	hashString := hex.EncodeToString(hash[:])

	if user.PasswordHash != hashString {
		return "", domain.ErrInvalidPassword
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// CreateOrder проверяет номер заказа по алгоритму Луна и сохраняет заказ.
func (s *Service) CreateOrder(userID int64, number string) (*domain.Order, error) {
	if !validateLuhn(number) {
		return nil, domain.ErrInvalidOrder
	}
	return s.repo.CreateOrder(userID, number)
}

// GetUserOrders возвращает все заказы пользователя.
func (s *Service) GetUserOrders(userID int64) ([]domain.Order, error) {
	return s.repo.GetUserOrders(userID)
}

// GetBalance возвращает текущий баланс пользователя.
func (s *Service) GetBalance(userID int64) (domain.Balance, error) {
	return s.repo.GetBalance(userID)
}

// Withdraw проверяет номер заказа по алгоритму Луна и списывает сумму с баланса.
func (s *Service) Withdraw(userID int64, orderNumber string, sum float64) error {
	orderNumber = strings.TrimSpace(orderNumber)
	if !validateLuhn(orderNumber) {
		return domain.ErrInvalidOrder
	}
	return s.repo.Withdraw(userID, orderNumber, sum)
}

// GetWithdrawals возвращает историю списаний пользователя.
func (s *Service) GetWithdrawals(userID int64) ([]domain.Withdrawal, error) {
	return s.repo.GetWithdrawals(userID)
}

// validateLuhn проверяет контрольную сумму номера заказа по алгоритму Луна.
// Вызывается при загрузке заказа и при списании баллов.
func validateLuhn(number string) bool {
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

// ---------------------------------------------------------------------------
// Воркер начислений
//
// StartAccrualWorker предоставлен. Реализуйте processAllPendingOrders
// и processOrder самостоятельно.
//
// Это самая интересная часть проекта: конкурентная обработка заказов.
// Подумайте, как защитить доступ к processingOrders из нескольких горутин.
// ---------------------------------------------------------------------------

// StartAccrualWorker запускает фоновый цикл, который каждые 3 секунды
// передаёт необработанные заказы в processAllPendingOrders.
// Останавливается при отмене ctx.
func (s *Service) StartAccrualWorker(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processAllPendingOrders(ctx)
		}
	}
}

// processAllPendingOrders получает заказы для обработки и запускает горутины.
// Реализуйте самостоятельно.
// processAllPendingOrders получает заказы для обработки и запускает горутины.
func (s *Service) processAllPendingOrders(ctx context.Context) {
	// TODO: замените interface{} на свой интерфейс и раскомментируйте
	orders, err := s.repo.GetOrdersForProcessing()
	if err != nil {
		log.Printf("воркер: не удалось получить заказы: %v", err)
		return
	}

	g, gctx := errgroup.WithContext(ctx)

	g.SetLimit(5)

	for _, order := range orders {
		number := order.Number

		if s.isProcessing(number) {
			continue
		}
		s.markProcessing(number)

		n := number
		g.Go(func() error {
			defer s.unmarkProcessing(n)

			s.processOrder(gctx, n)
			return nil
		})
	}
	// TODO: итерируйтесь по заказам, пропускайте те что уже в обработке,
	// для остальных запускайте s.processOrder через g.Go

	if err := g.Wait(); err != nil {
		log.Printf("воркер: ошибка группы: %v", err)
	}
}

// processOrder обрабатывает один заказ. Реализуйте самостоятельно.
// Используйте вспомогательные функции ниже для генерации случайных значений.
func (s *Service) processOrder(ctx context.Context, number string) {
	if err := s.repo.UpdateOrderStatus(number, domain.OrderStatusProcessing, 0); err != nil {
		log.Printf("воркер: не удалось обновить статус заказа %s: %v", number, err)
		return
	}
	delay := randomDelay()
	select {
	case <-ctx.Done():
		return
	case <-time.After(delay):
	}

	if isInvalid() {
		if err := s.repo.UpdateOrderStatus(number, domain.OrderStatusInvalid, 0); err != nil {
			log.Printf("воркер: INVALID")
		}
	}

	accural := randomAccrual()
	if err := s.repo.UpdateOrderStatus(number, domain.OrderStatusProcessed, accural); err != nil {
		log.Printf("воркер: PROCESSED не установился для %s: %v", number, err)
	}

}

// ---------------------------------------------------------------------------
// Вспомогательные функции - предоставлены
// ---------------------------------------------------------------------------

// randomAccrual возвращает случайное начисление от 10 до 500 баллов.
func randomAccrual() float64 {
	return float64(rand.Intn(491) + 10)
}

// randomDelay возвращает случайную задержку от 2 до 6 секунд.
func randomDelay() time.Duration {
	return time.Duration(rand.Intn(5)+2) * time.Second
}

// isInvalid возвращает true примерно в 10% случаев.
func isInvalid() bool {
	return rand.Intn(10) == 0
}

func (s *Service) markProcessing(number string) {
	s.muProcessing.Lock()
	defer s.muProcessing.Unlock()
	s.processingOrders[number] = true
}

func (s *Service) unmarkProcessing(number string) {
	s.muProcessing.Lock()
	defer s.muProcessing.Unlock()
	delete(s.processingOrders, number)
}

func (s *Service) isProcessing(number string) bool {
	s.muProcessing.Lock()
	defer s.muProcessing.Unlock()
	return s.processingOrders[number]
}
