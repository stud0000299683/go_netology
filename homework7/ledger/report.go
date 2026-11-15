package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const reportCacheTTL = 30 * time.Second

// ReportRow — сумма расходов одной категории за интервал дат.
type ReportRow struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// GetReportSummary считает суммы по категориям за включительный интервал.
// Повторный запрос с теми же датами в течение TTL берётся из Redis.
func GetReportSummary(ctx context.Context, from, to time.Time) ([]ReportRow, error) {
	if err := requireDB(); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("report:summary:%s:%s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	if raw, ok, err := redisCache.Get(ctx, key); err != nil {
		log.Printf("кеш отчёта недоступен, считаю по базе: %v", err)
	} else if ok {
		var cached []ReportRow
		if err := json.Unmarshal([]byte(raw), &cached); err != nil {
			log.Printf("кеш отчёта повреждён, считаю по базе: %v", err)
		} else {
			log.Printf("кеш отчёта: попадание, ключ %s", key)
			return cached, nil
		}
	}

	rows, err := database.QueryContext(ctx, `
		SELECT category, COALESCE(SUM(amount), 0)::float8
		FROM expenses
		WHERE date >= $1 AND date <= $2
		GROUP BY category
		ORDER BY category
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ReportRow, 0)
	for rows.Next() {
		var item ReportRow
		if err := rows.Scan(&item.Category, &item.Total); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if err := redisCache.Set(ctx, key, string(payload), reportCacheTTL); err != nil {
		log.Printf("не удалось записать кеш отчёта: %v", err)
	} else {
		log.Printf("кеш отчёта: промах, ключ %s", key)
	}
	return result, nil
}
