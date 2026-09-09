package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type MarketNotificationStore interface {
	ActiveLineUserIDs(context.Context) ([]string, error)
	LogError(context.Context, string, string, map[string]any) error
}

type MarketNotificationWorker struct {
	repository MarketNotificationStore
	messenger  LinePusher
	redis      *redis.Client
	interval   time.Duration
	now        func() time.Time
}

func NewMarketNotificationWorker(repository MarketNotificationStore, messenger LinePusher, redisClient *redis.Client, interval time.Duration) *MarketNotificationWorker {
	return &MarketNotificationWorker{repository: repository, messenger: messenger, redis: redisClient, interval: interval, now: time.Now}
}

func (w *MarketNotificationWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		w.check(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *MarketNotificationWorker) check(ctx context.Context) {
	event, eventTime, ok := marketTransition(w.now(), w.interval+time.Minute)
	if !ok {
		return
	}
	key := "market-notification:US:" + event + ":" + eventTime.Format("2006-01-02")
	claimed, err := w.redis.SetNX(ctx, key, "sending", 14*24*time.Hour).Result()
	if err != nil || !claimed {
		return
	}
	recipients, err := w.repository.ActiveLineUserIDs(ctx)
	if err != nil {
		_ = w.redis.Del(ctx, key).Err()
		_ = w.repository.LogError(ctx, "RECIPIENTS_FAILED", err.Error(), nil)
		return
	}
	message := marketTransitionMessage(event, eventTime)
	for _, lineUserID := range recipients {
		if pushErr := w.messenger.PushText(ctx, lineUserID, message); pushErr != nil {
			_ = w.repository.LogError(ctx, "LINE_PUSH_FAILED", pushErr.Error(), map[string]any{"event": event})
		}
	}
	log.Printf("sent US market %s notification to %d LINE users", event, len(recipients))
}

func marketTransition(now time.Time, window time.Duration) (string, time.Time, bool) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		return "", time.Time{}, false
	}
	local := now.In(location)
	if !isUSTradingDay(local) {
		return "", time.Time{}, false
	}
	open := time.Date(local.Year(), local.Month(), local.Day(), 9, 30, 0, 0, location)
	closeAt := time.Date(local.Year(), local.Month(), local.Day(), 16, 0, 0, 0, location)
	if !local.Before(open) && local.Sub(open) <= window {
		return "OPEN", open.UTC(), true
	}
	if !local.Before(closeAt) && local.Sub(closeAt) <= window {
		return "CLOSED", closeAt.UTC(), true
	}
	return "", time.Time{}, false
}

func marketTransitionMessage(event string, eventTime time.Time) string {
	bangkok, _ := time.LoadLocation("Asia/Bangkok")
	local := eventTime.In(bangkok).Format("Mon 2 Jan 2006, 15:04")
	if event == "OPEN" {
		return fmt.Sprintf("🟢 US stock market is now OPEN\nตลาดหุ้นสหรัฐฯ เปิดแล้ว\nเวลาไทย/ลาว: %s น.", local)
	}
	return fmt.Sprintf("🔴 US stock market is now CLOSED\nตลาดหุ้นสหรัฐฯ ปิดแล้ว\nเวลาไทย/ลาว: %s น.", local)
}
