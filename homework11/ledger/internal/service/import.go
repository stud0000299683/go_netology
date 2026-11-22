package service

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"

	"ledger/internal/domain"
)

type importResult struct {
	index int
	err   error
}

// ImportTransactions проводит расходы через каналы jobs и results.
// При отмене контекста возвращает уже собранную сводку и ошибку контекста.
func (s *Ledger) ImportTransactions(ctx context.Context, items []domain.IndexedTransaction, workers int) (domain.ImportSummary, error) {
	summary := domain.ImportSummary{Errors: []domain.ImportError{}}
	if len(items) == 0 {
		return summary, nil
	}
	if workers < 1 {
		workers = DefaultImportWorkers
	}
	if workers > maxImportWorkers {
		workers = maxImportWorkers
	}
	if workers > len(items) {
		workers = len(items)
	}

	jobs := make(chan domain.IndexedTransaction, len(items))
	results := make(chan importResult, len(items))

	var workersWG sync.WaitGroup
	for i := 0; i < workers; i++ {
		workersWG.Add(1)
		go s.importWorker(ctx, jobs, results, &workersWG)
	}

	var accepted atomic.Int64
	var rejected atomic.Int64
	collectorDone := make(chan struct{})
	go func() {
		defer close(collectorDone)
		for result := range results {
			if result.err != nil {
				rejected.Add(1)
				summary.Errors = append(summary.Errors, domain.ImportError{
					Index: result.index,
					Error: result.err.Error(),
				})
				continue
			}
			accepted.Add(1)
		}
	}()

	go func() {
		defer close(jobs)
		for _, item := range items {
			select {
			case <-ctx.Done():
				return
			case jobs <- item:
			}
		}
	}()

	go func() {
		workersWG.Wait()
		close(results)
	}()

	<-collectorDone
	summary.Accepted = int(accepted.Load())
	summary.Rejected = int(rejected.Load())
	sort.Slice(summary.Errors, func(i, j int) bool {
		return summary.Errors[i].Index < summary.Errors[j].Index
	})
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	return summary, nil
}

func (s *Ledger) importWorker(ctx context.Context, jobs <-chan domain.IndexedTransaction, results chan<- importResult, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		var item domain.IndexedTransaction
		select {
		case <-ctx.Done():
			return
		case got, ok := <-jobs:
			if !ok {
				return
			}
			item = got
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		err := s.importItem(ctx, item.Transaction)
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case results <- importResult{index: item.Index, err: err}:
		}
	}
}

func (s *Ledger) importItem(ctx context.Context, tx domain.Transaction) error {
	_, err := s.AddTransaction(ctx, tx)
	return err
}
