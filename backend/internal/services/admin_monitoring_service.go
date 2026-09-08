package services

import (
	"context"
	"strings"

	"stock_linebot/backend/internal/models"
)

type AdminMonitoringStore interface {
	LineStats(context.Context) (models.AdminLineStats, error)
	FailedLineMessages(context.Context, int, int) ([]models.AdminLineMessage, int64, error)
	ApplicationLogs(context.Context, models.ApplicationLogFilter) ([]models.ApplicationLog, int64, error)
	AuditLogs(context.Context, int, int) ([]models.AdminAuditLog, int64, error)
}
type AdminMonitoringService struct{ store AdminMonitoringStore }

func NewAdminMonitoringService(store AdminMonitoringStore) *AdminMonitoringService {
	return &AdminMonitoringService{store: store}
}

func (s *AdminMonitoringService) LineStats(ctx context.Context) (models.AdminLineStats, error) {
	stats, err := s.store.LineStats(ctx)
	if err != nil {
		return models.AdminLineStats{}, newError("INTERNAL_SERVER_ERROR", "Could not load LINE statistics.", err)
	}
	return stats, nil
}
func (s *AdminMonitoringService) FailedLineMessages(ctx context.Context, pageValue, limitValue string) ([]models.AdminLineMessage, int, int, int64, error) {
	page, limit, err := monitoringPagination(pageValue, limitValue)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	items, total, storeErr := s.store.FailedLineMessages(ctx, limit, (page-1)*limit)
	if storeErr != nil {
		return nil, 0, 0, 0, newError("INTERNAL_SERVER_ERROR", "Could not load failed LINE messages.", storeErr)
	}
	return items, page, limit, total, nil
}
func (s *AdminMonitoringService) ApplicationLogs(ctx context.Context, level, service, pageValue, limitValue string) (models.ApplicationLogPage, error) {
	page, limit, err := monitoringPagination(pageValue, limitValue)
	if err != nil {
		return models.ApplicationLogPage{}, err
	}
	level = strings.ToUpper(strings.TrimSpace(level))
	service = strings.TrimSpace(service)
	if level != "" && level != "DEBUG" && level != "INFO" && level != "WARN" && level != "ERROR" {
		return models.ApplicationLogPage{}, newError("INVALID_FILTER", "Level must be DEBUG, INFO, WARN, or ERROR.", nil)
	}
	if len(service) > 100 {
		return models.ApplicationLogPage{}, newError("INVALID_FILTER", "Service must not exceed 100 characters.", nil)
	}
	logs, total, storeErr := s.store.ApplicationLogs(ctx, models.ApplicationLogFilter{Level: level, Service: service, Limit: limit, Offset: (page - 1) * limit})
	if storeErr != nil {
		return models.ApplicationLogPage{}, newError("INTERNAL_SERVER_ERROR", "Could not load application logs.", storeErr)
	}
	return models.ApplicationLogPage{Logs: logs, Page: page, Limit: limit, Total: total}, nil
}
func (s *AdminMonitoringService) AuditLogs(ctx context.Context, pageValue, limitValue string) (models.AdminAuditLogPage, error) {
	page, limit, err := monitoringPagination(pageValue, limitValue)
	if err != nil {
		return models.AdminAuditLogPage{}, err
	}
	logs, total, storeErr := s.store.AuditLogs(ctx, limit, (page-1)*limit)
	if storeErr != nil {
		return models.AdminAuditLogPage{}, newError("INTERNAL_SERVER_ERROR", "Could not load audit logs.", storeErr)
	}
	return models.AdminAuditLogPage{Logs: logs, Page: page, Limit: limit, Total: total}, nil
}
func monitoringPagination(pageValue, limitValue string) (int, int, error) {
	page, err := positiveInt(pageValue, 1, 1_000_000)
	if err != nil {
		return 0, 0, newError("INVALID_FILTER", "Page must be a positive integer.", err)
	}
	limit, err := positiveInt(limitValue, 50, 100)
	if err != nil {
		return 0, 0, newError("INVALID_FILTER", "Limit must be between 1 and 100.", err)
	}
	return page, limit, nil
}
