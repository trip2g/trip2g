package nowpayments

// https://documenter.getpostman.com/view/7907941/2s93JusNJt#62a6d281-478d-4927-8cd0-f96d677b8de6
// waiting - waiting for the customer to send the payment. The initial status of each payment;
// confirming - the transaction is being processed on the blockchain. Appears when NOWPayments detect the funds from the user on the blockchain;
// confirmed - the process is confirmed by the blockchain. Customer’s funds have accumulated enough confirmations;
// sending - the funds are being sent to your personal wallet. We are in the process of sending the funds to you;
// partially_paid - it shows that the customer sent the less than the actual price. Appears when the funds have arrived in your wallet;
// finished - the funds have reached your personal address and the payment is finished;
// failed - the payment wasn't completed due to the error of some kind;
// refunded - the funds were refunded back to the user;
// expired - the user didn't send the funds to the specified address in the 7 days time window;

type PaymentStatus string

const (
	PaymentStatusWaiting       PaymentStatus = "waiting"
	PaymentStatusConfirming    PaymentStatus = "confirming"
	PaymentStatusConfirmed     PaymentStatus = "confirmed"
	PaymentStatusSending       PaymentStatus = "sending"
	PaymentStatusPartiallyPaid PaymentStatus = "partially_paid"
	PaymentStatusFinished      PaymentStatus = "finished"
	PaymentStatusFailed        PaymentStatus = "failed"
	PaymentStatusRefunded      PaymentStatus = "refunded"
	PaymentStatusExpired       PaymentStatus = "expired"
)

func (s PaymentStatus) Valid() bool {
	switch s {
	case PaymentStatusWaiting,
		PaymentStatusConfirming,
		PaymentStatusConfirmed,
		PaymentStatusSending,
		PaymentStatusPartiallyPaid,
		PaymentStatusFinished,
		PaymentStatusFailed,
		PaymentStatusRefunded,
		PaymentStatusExpired:
		return true
	default:
		return false
	}
}

func (s PaymentStatus) IsSuccessful() bool {
	switch s {
	case PaymentStatusConfirmed,
		PaymentStatusSending,
		PaymentStatusFinished:
		return true
	default:
		return false
	}
}
