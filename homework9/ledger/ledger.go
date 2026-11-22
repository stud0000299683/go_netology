// Пакет ledger — единственная точка, которую импортирует Gateway.
// Репозитории и драйвер базы остаются внутри модуля.
package ledger

import (
	"context"

	"ledger/internal/app"
	"ledger/internal/domain"
	"ledger/internal/service"
)

// Transaction, Budget и ReportRow доступны Gateway для преобразования DTO.
type (
	Transaction = domain.Transaction
	Budget      = domain.Budget
	ReportRow   = domain.ReportRow
	Service     = service.Service
)

// ErrBudgetExceeded — та же ошибка, которую возвращает сервис при превышении лимита.
var ErrBudgetExceeded = domain.ErrBudgetExceeded

// New читает окружение, подключается к базе и Redis и отдаёт сервис вместе с закрытием ресурсов.
func New(ctx context.Context) (Service, func(), error) {
	return app.New(ctx)
}
