//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaymentConfig_UsesIndependentPurchaseAvailability(t *testing.T) {
	t.Parallel()

	// Given
	service := &PaymentConfigService{}
	balancePurchaseEnabled := false
	subscriptionPurchaseEnabled := true
	repository := &paymentConfigSettingRepoStub{values: map[string]string{}}

	// When
	defaults := service.parsePaymentConfig(map[string]string{})
	service = &PaymentConfigService{settingRepo: repository}
	err := service.UpdatePaymentConfig(context.Background(), UpdatePaymentConfigRequest{
		BalancePurchaseEnabled:      &balancePurchaseEnabled,
		SubscriptionPurchaseEnabled: &subscriptionPurchaseEnabled,
	})
	persisted := service.parsePaymentConfig(repository.values)

	// Then
	require.True(t, defaults.BalancePurchaseEnabled)
	require.True(t, defaults.SubscriptionPurchaseEnabled)
	require.NoError(t, err)
	require.Equal(t, "false", repository.values[SettingBalancePurchaseEnabled])
	require.Equal(t, "true", repository.values[SettingSubscriptionPurchaseEnabled])
	require.False(t, persisted.BalancePurchaseEnabled)
	require.True(t, persisted.SubscriptionPurchaseEnabled)
}
