package nowpayments_test

import (
	"encoding/json"
	"testing"

	"trip2g/internal/nowpayments"

	"github.com/stretchr/testify/require"
)

// TestServiceFeeFloatParsing tests that ServiceFee can be parsed as a float
// This test addresses the error: strconv.ParseInt: parsing "0.000173": invalid syntax.
func TestServiceFeeFloatParsing(t *testing.T) {
	// This is the actual JSON from the error log that was failing
	jsonData := `{
		"actually_paid": 0.13598286,
		"actually_paid_at_fiat": 0,
		"fee": {
			"currency": "usdtmatic",
			"depositFee": 0.003117,
			"serviceFee": 0.000173,
			"withdrawalFee": 0.006711
		},
		"invoice_id": 6301727088,
		"order_description": "Second brain course",
		"order_id": "N12MWGYZ",
		"outcome_amount": 0.017031,
		"outcome_currency": "usdtmatic",
		"parent_payment_id": null,
		"pay_address": "0x6229ea8A608d21Ca3801b4f5B04DebB83a308699",
		"pay_amount": 0.13598286,
		"pay_currency": "maticmainnet",
		"payin_extra_id": null,
		"payment_extra_ids": null,
		"payment_id": 6145518892,
		"payment_status": "finished",
		"price_amount": 0.03,
		"price_currency": "usd",
		"purchase_id": "4713616180",
		"updated_at": 1749037915966
	}`

	var req nowpayments.IPNRequest
	err := json.Unmarshal([]byte(jsonData), &req)

	// Before the fix, this would fail with:
	// "parse error: strconv.ParseInt: parsing \"0.000173\": invalid syntax"
	require.NoError(t, err, "Should be able to parse IPN request with float serviceFee")

	// Verify the fee values are correctly parsed
	require.Equal(t, "usdtmatic", req.Fee.Currency)
	require.Equal(t, 0.003117, req.Fee.DepositFee)
	require.Equal(t, 0.000173, req.Fee.ServiceFee) // This was the problematic field
	require.Equal(t, 0.006711, req.Fee.WithdrawalFee)

	// Verify other fields are also correctly parsed
	require.Equal(t, 0.13598286, req.ActuallyPaid)
	require.Equal(t, int64(6301727088), req.InvoiceID)
	require.Equal(t, "N12MWGYZ", req.OrderID)
	require.Equal(t, nowpayments.PaymentStatusFinished, req.PaymentStatus)
	require.Equal(t, "4713616180", req.PurchaseID)
}

// TestServiceFeeIntegerValue tests that ServiceFee still works with integer values.
func TestServiceFeeIntegerValue(t *testing.T) {
	jsonData := `{
		"actually_paid": 1.0,
		"actually_paid_at_fiat": 0,
		"fee": {
			"currency": "usd",
			"depositFee": 0.1,
			"serviceFee": 5,
			"withdrawalFee": 0.2
		},
		"invoice_id": 123,
		"order_description": "Test",
		"order_id": "TEST123",
		"outcome_amount": 1.0,
		"outcome_currency": "usd",
		"parent_payment_id": null,
		"pay_address": "test_address",
		"pay_amount": 1.0,
		"pay_currency": "usd",
		"payin_extra_id": null,
		"payment_extra_ids": null,
		"payment_id": 456,
		"payment_status": "confirmed",
		"price_amount": 1.0,
		"price_currency": "usd",
		"purchase_id": "PURCHASE123",
		"updated_at": 1640995200000
	}`

	var req nowpayments.IPNRequest
	err := json.Unmarshal([]byte(jsonData), &req)
	require.NoError(t, err, "Should be able to parse IPN request with integer serviceFee")

	// Verify integer value is correctly parsed as float
	require.Equal(t, 5.0, req.Fee.ServiceFee)
}
