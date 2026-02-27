package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/referral"
	"github.com/Wei-Shaw/sub2api/ent/referralcommission"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ReferralRepository implements service.ReferralRepository using Ent.
type ReferralRepository struct{ client *dbent.Client }

// NewReferralRepository creates a new ReferralRepository.
func NewReferralRepository(client *dbent.Client) service.ReferralRepository {
	return &ReferralRepository{client: client}
}

func (r *ReferralRepository) CreateReferral(ctx context.Context, inviterID, inviteeID int64) error {
	_, err := r.client.Referral.Create().
		SetInviterID(inviterID).
		SetInviteeID(inviteeID).
		Save(ctx)
	return err
}

func (r *ReferralRepository) GetInviterByInviteeID(ctx context.Context, inviteeID int64) (int64, error) {
	ref, err := r.client.Referral.Query().
		Where(referral.InviteeIDEQ(inviteeID)).
		Only(ctx)
	if err != nil {
		return 0, err
	}
	return ref.InviterID, nil
}

func (r *ReferralRepository) GetUserByInviteCode(ctx context.Context, code string) (int64, error) {
	u, err := r.client.User.Query().
		Where(user.InviteCodeEQ(code)).
		Only(ctx)
	if err != nil {
		return 0, err
	}
	return int64(u.ID), nil
}

func (r *ReferralRepository) GetInviteCode(ctx context.Context, userID int64) (string, error) {
	u, err := r.client.User.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	if u.InviteCode == nil {
		return "", nil
	}
	return *u.InviteCode, nil
}

func (r *ReferralRepository) SetInviteCode(ctx context.Context, userID int64, code string) error {
	_, err := r.client.User.UpdateOneID(userID).SetInviteCode(code).Save(ctx)
	return err
}

func (r *ReferralRepository) CreateCommission(ctx context.Context, c *service.ReferralCommission) error {
	client := clientFromContext(ctx, r.client)
	builder := client.ReferralCommission.Create().
		SetInviterID(c.InviterID).
		SetInviteeID(c.InviteeID).
		SetOrderID(c.OrderID).
		SetOrderAmountUsd(c.OrderAmountUSD).
		SetCommissionRate(c.CommissionRate).
		SetCommissionAmount(c.CommissionAmount).
		SetStatus(c.Status)
	if c.SettledAt != nil {
		builder = builder.SetSettledAt(*c.SettledAt)
	}
	o, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	c.ID = int64(o.ID)
	return nil
}

func (r *ReferralRepository) ListCommissions(ctx context.Context, inviterID int64, limit, offset int) ([]service.ReferralCommission, int, error) {
	total, err := r.client.ReferralCommission.Query().
		Where(referralcommission.InviterIDEQ(inviterID)).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := r.client.ReferralCommission.Query().
		Where(referralcommission.InviterIDEQ(inviterID)).
		Order(dbent.Desc(referralcommission.FieldCreatedAt)).
		Limit(limit).Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]service.ReferralCommission, len(items))
	for i, item := range items {
		result[i] = service.ReferralCommission{
			ID:               int64(item.ID),
			InviterID:        item.InviterID,
			InviteeID:        item.InviteeID,
			OrderID:          item.OrderID,
			OrderAmountUSD:   item.OrderAmountUsd,
			CommissionRate:   item.CommissionRate,
			CommissionAmount: item.CommissionAmount,
			Status:           item.Status,
			SettledAt:        item.SettledAt,
			CreatedAt:        item.CreatedAt,
		}
	}
	return result, total, nil
}

func (r *ReferralRepository) SumCommissions(ctx context.Context, inviterID int64) (float64, error) {
	var result []struct {
		Sum *float64 `json:"sum"`
	}
	err := r.client.ReferralCommission.Query().
		Where(referralcommission.InviterIDEQ(inviterID)).
		Aggregate(dbent.Sum(referralcommission.FieldCommissionAmount)).
		Scan(ctx, &result)
	if err != nil {
		return 0, err
	}
	if len(result) == 0 || result[0].Sum == nil {
		return 0, nil
	}
	return *result[0].Sum, nil
}

func (r *ReferralRepository) CountInvitees(ctx context.Context, inviterID int64) (int, error) {
	return r.client.Referral.Query().
		Where(referral.InviterIDEQ(inviterID)).
		Count(ctx)
}
