package plati

import "testing"

func TestVerify(t *testing.T) {
	secret := "s3cr3t"
	payload := []byte("orderid=42&buyer=alice")
	sig := Sign(secret, payload)

	if !Verify(secret, payload, sig) {
		t.Error("валидная подпись отклонена")
	}
	if Verify(secret, payload, sig+"00") {
		t.Error("подпись неверной длины принята")
	}
	if Verify("wrong", payload, sig) {
		t.Error("подпись с чужим секретом принята")
	}
	if Verify(secret, []byte("tampered"), sig) {
		t.Error("подпись изменённого payload принята")
	}
}
