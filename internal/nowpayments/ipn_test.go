package nowpayments

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

func TestIPN(t *testing.T) {
	ipnKey := os.Getenv("NOWPAYMENTS_IPN_KEY")
	if ipnKey == "" {
		t.Skip("IPN key not set in environment variables")
	}

	body := `{"actually_paid":1.70073759,"actually_paid_at_fiat":0,"fee":{"currency":"usdtmatic","depositFee":0.004169,"serviceFee":0,"withdrawalFee":0.005413},"invoice_id":6158118973,"order_description":"Second brain course","order_id":"LEX3SZMT","outcome_amount":0.378719,"outcome_currency":"usdtmatic","parent_payment_id":null,"pay_address":"0xABB808cE003E6120bf20bF536834bDEF4e8A6259","pay_amount":1.70073759,"pay_currency":"maticmainnet","payin_extra_id":null,"payment_extra_ids":null,"payment_id":4562642554,"payment_status":"confirming","price_amount":0.4,"price_currency":"usd","purchase_id":"5844495295"}`
	sig := "38f4eed2aa642678a3ba65da73cd214da951e6557760cd13c9c4193eb2cc624b1bd2280faebab793f6264b69c8d362f78f3324452f1eaedd1be7e485371618e4"

	ok, err := CheckIPNSignature(ipnKey, sig, []byte(body))
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("IPN signature verification failed")
	}
}

func TestCheckIPNSignature_RefusesWithoutAKey(t *testing.T) {
	body := []byte(`{"order_id":"X","payment_status":"finished"}`)
	mac := hmac.New(sha512.New, []byte(""))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	ok, err := CheckIPNSignature("", sig, body)

	if ok {
		t.Fatal("a signature made with an empty key was accepted")
	}
	if !errors.Is(err, ErrNoIPNKey) {
		t.Fatalf("err = %v, want ErrNoIPNKey", err)
	}
}
