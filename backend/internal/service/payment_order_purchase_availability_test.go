//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestCreateOrder_RejectsDisabledPurchaseType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		request    CreateOrderRequest
		config     *PaymentConfig
		wantReason string
	}{
		{
			name:    "rejects balance when balance purchases are closed",
			request: CreateOrderRequest{OrderType: payment.OrderTypeBalance, Amount: 10},
			config: &PaymentConfig{
				BalancePurchaseEnabled:      false,
				SubscriptionPurchaseEnabled: true,
			},
			wantReason: "BALANCE_PURCHASE_DISABLED",
		},
		{
			name:    "rejects subscriptions when subscription purchases are closed",
			request: CreateOrderRequest{OrderType: payment.OrderTypeSubscription, PlanID: 1},
			config: &PaymentConfig{
				BalancePurchaseEnabled:      true,
				SubscriptionPurchaseEnabled: false,
			},
			wantReason: "SUBSCRIPTION_PURCHASE_DISABLED",
		},
		{
			name:    "rejects unknown order types before they can fall through to balance",
			request: CreateOrderRequest{OrderType: "unknown", Amount: 10},
			config: &PaymentConfig{
				BalancePurchaseEnabled:      false,
				SubscriptionPurchaseEnabled: true,
			},
			wantReason: "INVALID_ORDER_TYPE",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			service := &PaymentService{}

			// When
			_, err := service.validateOrderInput(context.Background(), test.request, test.config)

			// Then
			require.Error(t, err)
			require.Equal(t, test.wantReason, infraerrors.FromError(err).Reason)
		})
	}
}
