package filter

import "testing"

func TestRejectH8PaymentVertical(t *testing.T) {
	t.Parallel()
	if drop, _ := RejectH8PaymentVertical("оплата на ФОП, без трекера", ""); !drop {
		t.Fatal("expected fop drop")
	}
	if drop, _ := RejectH8PaymentVertical("Ищем media buyer Киев, USDT TRC20, voluum postback fail", ""); drop {
		t.Fatal("expected ua usdt buyer voice pass")
	}
	if drop, _ := RejectH8PaymentVertical("инфобиз курсы по арбитражу с нуля", ""); !drop {
		t.Fatal("expected infobiz drop")
	}
}

func TestRejectCryptoPayoutOnly(t *testing.T) {
	t.Parallel()
	if drop, _ := RejectCryptoPayoutOnly("join vip usdt pump channel"); !drop {
		t.Fatal("expected crypto-only drop")
	}
	if drop, _ := RejectCryptoPayoutOnly("weekly usdt payout but keitaro postback failing"); drop {
		t.Fatal("expected tracker pain pass")
	}
}
