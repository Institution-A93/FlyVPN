// Package cron — периодические обязанности оркестратора над аккаунт-слоем MVP:
// принудительное истечение/квота, пороговые нотификации, ролл месячного периода,
// уборка OTP, мониторинг зависших платежей (docs/backend-requirements.md §9, §10).
package cron

import (
	"context"
	"log/slog"
	"time"
)

// Lifecycle — то, что воркеру нужно от стораджа (для тестируемости).
type Lifecycle interface {
	ExpireSubscriptions(ctx context.Context) (int64, error)
	RevokeExpiredCreds(ctx context.Context) (int64, error)
	EnforceQuota(ctx context.Context) (int64, error)
	RestoreUnderQuota(ctx context.Context) (int64, error)
	RollPeriods(ctx context.Context) (int64, error)
	EnqueueThresholdEvents(ctx context.Context) (int64, error)
	CleanupOTP(ctx context.Context) (int64, error)
	CountStalePendingPayments(ctx context.Context) (int64, error)
}

// Worker гоняет sweeps по расписанию.
type Worker struct {
	store Lifecycle
	log   *slog.Logger
}

// New собирает воркер.
func New(store Lifecycle, log *slog.Logger) *Worker {
	return &Worker{store: store, log: log}
}

// Run запускает два цикла: частый (sweep) и часовой (housekeeping). Первый прогон —
// сразу, затем по тикерам. Завершается по ctx.
func (w *Worker) Run(ctx context.Context, sweepEvery, houseEvery time.Duration) {
	w.sweep(ctx)
	w.housekeeping(ctx)

	sweepT := time.NewTicker(sweepEvery)
	houseT := time.NewTicker(houseEvery)
	defer sweepT.Stop()
	defer houseT.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sweepT.C:
			w.sweep(ctx)
		case <-houseT.C:
			w.housekeeping(ctx)
		}
	}
}

// sweep — частые обязанности (1–5 мин): истечение, квота, пороги.
func (w *Worker) sweep(ctx context.Context) {
	w.run(ctx, "expire_subscriptions", w.store.ExpireSubscriptions)
	w.run(ctx, "revoke_expired_creds", w.store.RevokeExpiredCreds)
	w.run(ctx, "enforce_quota", w.store.EnforceQuota)
	w.run(ctx, "restore_under_quota", w.store.RestoreUnderQuota)
	w.run(ctx, "threshold_events", w.store.EnqueueThresholdEvents)
}

// housekeeping — часовые обязанности: ролл периода (с догоном), уборка OTP, мониторинг платежей.
func (w *Worker) housekeeping(ctx context.Context) {
	// Ролл периода: повторяем, пока есть что продвигать (догон нескольких месяцев).
	for i := 0; i < 24; i++ {
		n, err := w.store.RollPeriods(ctx)
		if err != nil {
			w.log.Error("cron", "job", "roll_periods", "err", err)
			break
		}
		if n == 0 {
			break
		}
		w.log.Info("cron", "job", "roll_periods", "rows", n)
	}
	w.run(ctx, "cleanup_otp", w.store.CleanupOTP)

	if n, err := w.store.CountStalePendingPayments(ctx); err != nil {
		w.log.Error("cron", "job", "stale_payments", "err", err)
	} else if n > 0 {
		// Реальный poll статуса — за account-api (клиент Platega). Здесь только сигнал.
		w.log.Warn("cron", "job", "stale_payments", "count", n)
	}
}

func (w *Worker) run(ctx context.Context, job string, fn func(context.Context) (int64, error)) {
	n, err := fn(ctx)
	if err != nil {
		w.log.Error("cron", "job", job, "err", err)
		return
	}
	if n > 0 {
		w.log.Info("cron", "job", job, "rows", n)
	}
}
