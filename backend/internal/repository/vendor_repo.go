package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/vendor"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"entgo.io/ent/dialect/sql"
)

type vendorRepository struct {
	client *dbent.Client
}

func NewVendorRepository(client *dbent.Client) service.VendorRepository {
	return &vendorRepository{client: client}
}

func (r *vendorRepository) Create(ctx context.Context, v *service.Vendor) error {
	builder := r.client.Vendor.Create().
		SetName(v.Name).
		SetAPIFormat(v.APIFormat).
		SetBaseURL(v.BaseURL).
		SetAuthType(v.AuthType).
		SetBillingType(v.BillingType).
		SetUsedQuotaUsd(v.UsedQuotaUSD).
		SetStatus(v.Status).
		SetHealthCheckEnabled(v.HealthCheckEnabled).
		SetHealthCheckInterval(v.HealthCheckInterval).
		SetHealthCheckModel(v.HealthCheckModel).
		SetConsecutiveFailures(v.ConsecutiveFailures).
		SetAutoPurchaseEnabled(v.AutoPurchaseEnabled).
		SetBalanceAlertEnabled(v.BalanceAlertEnabled)

	if v.Description != nil {
		builder.SetDescription(*v.Description)
	}
	if v.APIPathOverride != nil {
		builder.SetAPIPathOverride(*v.APIPathOverride)
	}
	if v.ExtraHeaders != nil {
		builder.SetExtraHeaders(v.ExtraHeaders)
	}
	if v.CostPer1kInput != nil {
		builder.SetCostPer1kInput(*v.CostPer1kInput)
	}
	if v.CostPer1kOutput != nil {
		builder.SetCostPer1kOutput(*v.CostPer1kOutput)
	}
	if v.TotalQuotaUSD != nil {
		builder.SetTotalQuotaUsd(*v.TotalQuotaUSD)
	}
	if v.BalanceUSD != nil {
		builder.SetBalanceUsd(*v.BalanceUSD)
	}
	if v.ExpiresAt != nil {
		builder.SetExpiresAt(*v.ExpiresAt)
	}
	if v.AutoPurchaseConfig != nil {
		builder.SetAutoPurchaseConfig(v.AutoPurchaseConfig)
	}
	if v.BalanceAlertThreshold != nil {
		builder.SetBalanceAlertThreshold(*v.BalanceAlertThreshold)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	applyVendorEntityToService(v, created)
	return nil
}

func (r *vendorRepository) GetByID(ctx context.Context, id int64) (*service.Vendor, error) {
	e, err := r.client.Vendor.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrVendorNotFound
		}
		return nil, err
	}
	return vendorEntityToService(e), nil
}

func (r *vendorRepository) Update(ctx context.Context, v *service.Vendor) error {
	builder := r.client.Vendor.UpdateOneID(v.ID).
		SetName(v.Name).
		SetAPIFormat(v.APIFormat).
		SetBaseURL(v.BaseURL).
		SetAuthType(v.AuthType).
		SetBillingType(v.BillingType).
		SetUsedQuotaUsd(v.UsedQuotaUSD).
		SetStatus(v.Status).
		SetHealthCheckEnabled(v.HealthCheckEnabled).SetHealthCheckInterval(v.HealthCheckInterval).
		SetHealthCheckModel(v.HealthCheckModel).
		SetConsecutiveFailures(v.ConsecutiveFailures).
		SetAutoPurchaseEnabled(v.AutoPurchaseEnabled).
		SetBalanceAlertEnabled(v.BalanceAlertEnabled)

	if v.Description != nil {
		builder.SetDescription(*v.Description)
	} else {
		builder.ClearDescription()
	}
	if v.APIPathOverride != nil {
		builder.SetAPIPathOverride(*v.APIPathOverride)
	} else {
		builder.ClearAPIPathOverride()
	}
	if v.ExtraHeaders != nil {
		builder.SetExtraHeaders(v.ExtraHeaders)
	}
	if v.CostPer1kInput != nil {
		builder.SetCostPer1kInput(*v.CostPer1kInput)
	} else {
		builder.ClearCostPer1kInput()
	}
	if v.CostPer1kOutput != nil {
		builder.SetCostPer1kOutput(*v.CostPer1kOutput)
	} else {
		builder.ClearCostPer1kOutput()
	}
	if v.TotalQuotaUSD != nil {
		builder.SetTotalQuotaUsd(*v.TotalQuotaUSD)
	} else {
		builder.ClearTotalQuotaUsd()
	}
	if v.BalanceUSD != nil {
		builder.SetBalanceUsd(*v.BalanceUSD)
	} else {
		builder.ClearBalanceUsd()
	}
	if v.ExpiresAt != nil {
		builder.SetExpiresAt(*v.ExpiresAt)
	} else {
		builder.ClearExpiresAt()
	}
	if v.AutoPurchaseConfig != nil {
		builder.SetAutoPurchaseConfig(v.AutoPurchaseConfig)
	}
	if v.BalanceAlertThreshold != nil {
		builder.SetBalanceAlertThreshold(*v.BalanceAlertThreshold)
	} else {
		builder.ClearBalanceAlertThreshold()
	}
	if v.LastHealthCheckAt != nil {
		builder.SetLastHealthCheckAt(*v.LastHealthCheckAt)
	} else {
		builder.ClearLastHealthCheckAt()
	}
	if v.LastHealthStatus != nil {
		builder.SetLastHealthStatus(*v.LastHealthStatus)
	} else {
		builder.ClearLastHealthStatus()
	}
	if v.LastHealthLatency != nil {
		builder.SetLastHealthLatency(*v.LastHealthLatency)
	} else {
		builder.ClearLastHealthLatency()
	}
	if v.ErrorMessage != nil {
		builder.SetErrorMessage(*v.ErrorMessage)
	} else {
		builder.ClearErrorMessage()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrVendorNotFound
		}
		return err
	}
	applyVendorEntityToService(v, updated)
	return nil
}

func (r *vendorRepository) CountAccountsByVendorID(ctx context.Context, vendorID int64) (int, error) {
	count, err := r.client.Account.Query().
		Where(func(s *sql.Selector) {
			s.Where(sql.EQ("vendor_id", vendorID))
			s.Where(sql.IsNull("deleted_at"))
		}).
		Count(ctx)
	return count, err
}

func (r *vendorRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.client.Vendor.Delete().Where(vendor.IDEQ(id)).Exec(ctx)
	return err
}

func (r *vendorRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.Vendor, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, "", "", "", "")
}

func (r *vendorRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, status, apiFormat, billingType, search string) ([]service.Vendor, *pagination.PaginationResult, error) {
	q := r.client.Vendor.Query()
	if status != "" {
		q = q.Where(vendor.StatusEQ(status))
	}
	if apiFormat != "" {
		q = q.Where(vendor.APIFormatEQ(apiFormat))
	}
	if billingType != "" {
		q = q.Where(vendor.BillingTypeEQ(billingType))
	}
	if search != "" {
		q = q.Where(vendor.Or(
			vendor.NameContainsFold(search),
			vendor.DescriptionContainsFold(search),
		))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	entities, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(vendor.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	return vendorEntitiesToService(entities), paginationResultFromTotal(int64(total), params), nil
}

func (r *vendorRepository) ListActive(ctx context.Context) ([]service.Vendor, error) {
	entities, err := r.client.Vendor.Query().
		Where(vendor.StatusEQ(service.VendorStatusActive)).
		Order(dbent.Desc(vendor.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return vendorEntitiesToService(entities), nil
}

func (r *vendorRepository) ListByStatus(ctx context.Context, status string) ([]service.Vendor, error) {
	entities, err := r.client.Vendor.Query().
		Where(vendor.StatusEQ(status)).
		Order(dbent.Desc(vendor.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return vendorEntitiesToService(entities), nil
}

func (r *vendorRepository) ListHealthCheckDue(ctx context.Context) ([]service.Vendor, error) {
	entities, err := r.client.Vendor.Query().
		Where(
			vendor.HealthCheckEnabledEQ(true),
			vendor.StatusEQ(service.VendorStatusActive),
			vendor.Or(
				vendor.LastHealthCheckAtIsNil(),
				func(s *sql.Selector) {
					s.Where(sql.ExprP(
						"last_health_check_at + (health_check_interval || ' seconds')::interval <= $1",
						time.Now(),
					))
				},
			),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return vendorEntitiesToService(entities), nil
}

func (r *vendorRepository) ListBalanceAlertDue(ctx context.Context) ([]service.Vendor, error) {
	entities, err := r.client.Vendor.Query().
		Where(
			vendor.BalanceAlertEnabledEQ(true),
			vendor.BalanceUsdNotNil(),
			vendor.BalanceAlertThresholdNotNil(),
			func(s *sql.Selector) {
				s.Where(sql.ExprP("balance_usd <= balance_alert_threshold"))
			},
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return vendorEntitiesToService(entities), nil
}

func (r *vendorRepository) UpdateHealthStatus(ctx context.Context, id int64, status string, latency *int, errMsg *string, consecutiveFailures int) error {
	now := time.Now()
	builder := r.client.Vendor.UpdateOneID(id).
		SetLastHealthCheckAt(now).
		SetLastHealthStatus(status).
		SetConsecutiveFailures(consecutiveFailures)

	if latency != nil {
		builder.SetLastHealthLatency(*latency)
	} else {
		builder.ClearLastHealthLatency()
	}
	if errMsg != nil {
		builder.SetErrorMessage(*errMsg)
	} else {
		builder.ClearErrorMessage()
	}

	_, err := builder.Save(ctx)
	if err != nil && dbent.IsNotFound(err) {
		return service.ErrVendorNotFound
	}
	return err
}

func (r *vendorRepository) UpdateBalance(ctx context.Context, id int64, balanceUSD *float64, usedQuotaUSD float64) error {
	builder := r.client.Vendor.UpdateOneID(id).
		SetUsedQuotaUsd(usedQuotaUSD)

	if balanceUSD != nil {
		builder.SetBalanceUsd(*balanceUSD)
	} else {
		builder.ClearBalanceUsd()
	}

	_, err := builder.Save(ctx)
	if err != nil && dbent.IsNotFound(err) {
		return service.ErrVendorNotFound
	}
	return err
}

func (r *vendorRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	_, err := r.client.Vendor.UpdateOneID(id).
		SetStatus(status).
		Save(ctx)
	if err != nil && dbent.IsNotFound(err) {
		return service.ErrVendorNotFound
	}
	return err
}

func vendorEntityToService(e *dbent.Vendor) *service.Vendor {
	if e == nil {
		return nil
	}
	return &service.Vendor{
		ID:                    e.ID,
		Name:                  e.Name,
		Description:           e.Description,
		APIFormat:             e.APIFormat,
		BaseURL:               e.BaseURL,
		AuthType:              e.AuthType,
		APIPathOverride:       e.APIPathOverride,
		ExtraHeaders:          e.ExtraHeaders,
		BillingType:           e.BillingType,
		CostPer1kInput:        e.CostPer1kInput,
		CostPer1kOutput:       e.CostPer1kOutput,
		TotalQuotaUSD:         e.TotalQuotaUsd,
		UsedQuotaUSD:          e.UsedQuotaUsd,
		BalanceUSD:            e.BalanceUsd,
		ExpiresAt:             e.ExpiresAt,
		Status:                e.Status,
		HealthCheckEnabled:    e.HealthCheckEnabled,
		HealthCheckInterval:   e.HealthCheckInterval,
		HealthCheckModel:      e.HealthCheckModel,
		LastHealthCheckAt:     e.LastHealthCheckAt,
		LastHealthStatus:      e.LastHealthStatus,
		LastHealthLatency:     e.LastHealthLatency,
		ErrorMessage:          e.ErrorMessage,
		ConsecutiveFailures:   e.ConsecutiveFailures,
		AutoPurchaseEnabled:   e.AutoPurchaseEnabled,
		AutoPurchaseConfig:    e.AutoPurchaseConfig,
		BalanceAlertEnabled:   e.BalanceAlertEnabled,
		BalanceAlertThreshold: e.BalanceAlertThreshold,
		CreatedAt:             e.CreatedAt,
		UpdatedAt:             e.UpdatedAt,
	}
}

func applyVendorEntityToService(dst *service.Vendor, src *dbent.Vendor) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

func vendorEntitiesToService(entities []*dbent.Vendor) []service.Vendor {
	out := make([]service.Vendor, 0, len(entities))
	for _, e := range entities {
		out = append(out, *vendorEntityToService(e))
	}
	return out
}
