package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/rechargepackage"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// PaymentOrderRepository implements service.PaymentOrderRepository using Ent.
type PaymentOrderRepository struct{ client *dbent.Client }

// NewPaymentOrderRepository creates a new PaymentOrderRepository.
func NewPaymentOrderRepository(client *dbent.Client) service.PaymentOrderRepository {
	return &PaymentOrderRepository{client: client}
}

func (r *PaymentOrderRepository) Create(ctx context.Context, order *service.PaymentOrder) error {
	o, err := r.client.PaymentOrder.Create().
		SetOrderNo(order.OrderNo).
		SetUserID(order.UserID).
		SetAmountCny(order.AmountCNY).
		SetExchangeRate(order.ExchangeRate).
		SetAmount(order.AmountUSD).
		SetBonus(order.Bonus).
		SetTotalCredit(order.TotalCredit).
		SetChannel(order.Channel).
		SetStatus(order.Status).
		Save(ctx)
	if err != nil {
		return err
	}
	order.ID = int64(o.ID)
	return nil
}

func (r *PaymentOrderRepository) GetByOrderNo(ctx context.Context, orderNo string) (*service.PaymentOrder, error) {
	client := clientFromContext(ctx, r.client)
	o, err := client.PaymentOrder.Query().
		Where(paymentorder.OrderNoEQ(orderNo)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return entToPaymentOrder(o), nil
}

func (r *PaymentOrderRepository) GetByID(ctx context.Context, id int64) (*service.PaymentOrder, error) {
	client := clientFromContext(ctx, r.client)
	o, err := client.PaymentOrder.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return entToPaymentOrder(o), nil
}

func (r *PaymentOrderRepository) UpdateStatusAtomically(ctx context.Context, orderNo, fromStatus, toStatus, tradeNo string, paidAt *time.Time) (int, error) {
	client := clientFromContext(ctx, r.client)
	q := client.PaymentOrder.Update().
		Where(
			paymentorder.OrderNoEQ(orderNo),
			paymentorder.StatusEQ(fromStatus),
		).
		SetStatus(toStatus).
		SetTradeNo(tradeNo)
	if paidAt != nil {
		q = q.SetPaidAt(*paidAt)
	}
	return q.Save(ctx)
}

func (r *PaymentOrderRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]service.PaymentOrder, int, error) {
	total, err := r.client.PaymentOrder.Query().Where(paymentorder.UserIDEQ(userID)).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	orders, err := r.client.PaymentOrder.Query().
		Where(paymentorder.UserIDEQ(userID)).
		Order(dbent.Desc(paymentorder.FieldCreatedAt)).
		Limit(limit).Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]service.PaymentOrder, len(orders))
	for i, o := range orders {
		result[i] = *entToPaymentOrder(o)
	}
	return result, total, nil
}

func (r *PaymentOrderRepository) UpdateCommissionStatus(ctx context.Context, orderNo, status string) (int, error) {
	client := clientFromContext(ctx, r.client)
	return client.PaymentOrder.Update().
		Where(paymentorder.OrderNoEQ(orderNo)).
		SetCommissionStatus(status).
		Save(ctx)
}

func entToPaymentOrder(o *dbent.PaymentOrder) *service.PaymentOrder {
	return &service.PaymentOrder{
		ID:               int64(o.ID),
		OrderNo:          o.OrderNo,
		UserID:           o.UserID,
		AmountCNY:        o.AmountCny,
		ExchangeRate:     o.ExchangeRate,
		AmountUSD:        o.Amount,
		Bonus:            o.Bonus,
		TotalCredit:      o.TotalCredit,
		Channel:          o.Channel,
		Status:           o.Status,
		CommissionStatus: o.CommissionStatus,
		TradeNo:          o.TradeNo,
		PaidAt:           o.PaidAt,
		CreatedAt:        o.CreatedAt,
	}
}

// RechargePackageRepository implements service.RechargePackageRepository using Ent.
type RechargePackageRepository struct{ client *dbent.Client }

// NewRechargePackageRepository creates a new RechargePackageRepository.
func NewRechargePackageRepository(client *dbent.Client) service.RechargePackageRepository {
	return &RechargePackageRepository{client: client}
}

func (r *RechargePackageRepository) List(ctx context.Context, activeOnly bool) ([]service.RechargePackage, error) {
	q := r.client.RechargePackage.Query().Order(dbent.Asc(rechargepackage.FieldSortOrder))
	if activeOnly {
		q = q.Where(rechargepackage.IsActiveEQ(true))
	}
	pkgs, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]service.RechargePackage, len(pkgs))
	for i, p := range pkgs {
		result[i] = entToRechargePackage(p)
	}
	return result, nil
}

func (r *RechargePackageRepository) GetByID(ctx context.Context, id int64) (*service.RechargePackage, error) {
	p, err := r.client.RechargePackage.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	pkg := entToRechargePackage(p)
	return &pkg, nil
}

func (r *RechargePackageRepository) Create(ctx context.Context, pkg *service.RechargePackage) error {
	p, err := r.client.RechargePackage.Create().
		SetAmount(pkg.AmountCNY).
		SetBonusRate(pkg.BonusRate).
		SetBonusFixed(pkg.BonusFixed).
		SetNillableLabel(pkg.Label).
		SetIsActive(pkg.IsActive).
		SetSortOrder(pkg.SortOrder).
		Save(ctx)
	if err != nil {
		return err
	}
	pkg.ID = int64(p.ID)
	return nil
}

func (r *RechargePackageRepository) Update(ctx context.Context, pkg *service.RechargePackage) error {
	_, err := r.client.RechargePackage.UpdateOneID(pkg.ID).
		SetAmount(pkg.AmountCNY).
		SetBonusRate(pkg.BonusRate).
		SetBonusFixed(pkg.BonusFixed).
		SetNillableLabel(pkg.Label).
		SetIsActive(pkg.IsActive).
		SetSortOrder(pkg.SortOrder).
		Save(ctx)
	return err
}

func (r *RechargePackageRepository) Delete(ctx context.Context, id int64) error {
	return r.client.RechargePackage.DeleteOneID(id).Exec(ctx)
}

func entToRechargePackage(p *dbent.RechargePackage) service.RechargePackage {
	return service.RechargePackage{
		ID:         int64(p.ID),
		AmountCNY:  p.Amount,
		BonusRate:  p.BonusRate,
		BonusFixed: p.BonusFixed,
		Label:      p.Label,
		IsActive:   p.IsActive,
		SortOrder:  p.SortOrder,
	}
}
