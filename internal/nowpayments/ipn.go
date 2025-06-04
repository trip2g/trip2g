package nowpayments

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

//go:generate go tool github.com/mailru/easyjson/easyjson -snake_case -all -no_std_marshalers ./ipn.go

type IPNFee struct {
	Currency      string  `json:"currency"`
	DepositFee    float64 `json:"depositFee"`
	ServiceFee    float64 `json:"serviceFee"`
	WithdrawalFee float64 `json:"withdrawalFee"`
}

type IPNRequest struct {
	ActuallyPaid       float64     `json:"actually_paid"`
	ActuallyPaidAtFiat int         `json:"actually_paid_at_fiat"`
	Fee                IPNFee      `json:"fee"`
	InvoiceID          int64       `json:"invoice_id"`
	OrderDescription   string      `json:"order_description"`
	OrderID            string      `json:"order_id"`
	OutcomeAmount      float64     `json:"outcome_amount"`
	OutcomeCurrency    string      `json:"outcome_currency"`
	ParentPaymentID    interface{} `json:"parent_payment_id"`
	PayAddress         string      `json:"pay_address"`
	PayAmount          float64     `json:"pay_amount"`
	PayCurrency        string      `json:"pay_currency"`
	PayinExtraID       interface{} `json:"payin_extra_id"`
	PaymentExtraIds    interface{} `json:"payment_extra_ids"`
	PaymentID          int64       `json:"payment_id"`
	PriceAmount        float64     `json:"price_amount"`
	PriceCurrency      string      `json:"price_currency"`
	PurchaseID         string      `json:"purchase_id"`
	UpdatedAt          int64       `json:"updated_at"`

	PaymentStatus PaymentStatus `json:"payment_status"`
}

func CheckIPNSignature(secretKey string, sig string, body []byte) (bool, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal(body, &msg); err != nil {
		return false, fmt.Errorf("invalid JSON: %w", err)
	}

	// Sort keys
	keys := make([]string, 0, len(msg))
	for k := range msg {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Reconstruct sorted JSON with separators: ',' and ':'
	buffer := bytes.NewBufferString("{")
	for i, k := range keys {
		val, err := json.Marshal(msg[k])
		if err != nil {
			return false, fmt.Errorf("failed to marshal value for key %q: %w", k, err)
		}
		buffer.WriteString(fmt.Sprintf("%q:%s", k, val))
		if i != len(keys)-1 {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("}")

	// HMAC SHA-512
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(buffer.Bytes())
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(expectedSig), []byte(sig)), nil
}
