package store

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/institution-a93/flyvpn/services/account-api/internal/config"
)

// Интеграционный тест: запускается только если задан ACCOUNTAPI_TEST_DSN
// (схема 0001..0004 применена). Без DSN — пропуск (default-прогон зелёный без БД).
func testStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	dsn := os.Getenv("ACCOUNTAPI_TEST_DSN")
	if dsn == "" {
		t.Skip("ACCOUNTAPI_TEST_DSN не задан — пропуск интеграционного теста")
	}
	ctx := context.Background()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(st.Close)
	return st, ctx
}

var phoneSeq atomic.Int64

// uniq — монотонно растущий уникальный суффикс (БД переживает прогоны; берём младшие
// разряды наносекунд + счётчик, чтобы значения реально отличались внутри прогона).
func uniq() string {
	return strconv.FormatInt(time.Now().UnixNano()+phoneSeq.Add(1), 10)
}

func uniqPhone() string {
	n := (time.Now().UnixNano() + phoneSeq.Add(1)) % 1_000_000_000
	return "+79" + strconv.FormatInt(n, 10)
}

func TestSignupTrialAndDevice(t *testing.T) {
	st, ctx := testStore(t)
	phone := uniqPhone()

	uid, isNew, err := st.UpsertByPhone(ctx, phone, "")
	if err != nil || !isNew {
		t.Fatalf("signup: uid=%s isNew=%v err=%v", uid, isNew, err)
	}
	// Повторный логин — тот же user, не новый.
	uid2, isNew2, err := st.UpsertByPhone(ctx, phone, "")
	if err != nil || uid2 != uid || isNew2 {
		t.Fatalf("re-login: uid=%s isNew=%v err=%v", uid2, isNew2, err)
	}

	// Триал создан: активная подписка kind=trial, лимит 300 МБ.
	sub, err := st.ActiveSubscription(ctx, uid)
	if err != nil {
		t.Fatalf("active sub: %v", err)
	}
	if sub.Kind != "trial" || sub.TrafficBytesLimit != int64(config.TrialLimitBytes) {
		t.Fatalf("trial sub = %+v", sub)
	}

	used, limit, err := st.Traffic(ctx, uid)
	if err != nil || used != 0 || limit != int64(config.TrialLimitBytes) {
		t.Fatalf("traffic used=%d limit=%d err=%v", used, limit, err)
	}

	// Устройство выдаётся, попадает в список, отзывается.
	devID, ip, err := st.IssueDevice(ctx, uid, "user_"+phone[4:], "8846f7eaee8fb117ad06bdd830b7586c")
	if err != nil || devID == "" || ip == "" {
		t.Fatalf("issue device: id=%s ip=%s err=%v", devID, ip, err)
	}
	devs, err := st.ListDevices(ctx, uid)
	if err != nil || len(devs) != 1 {
		t.Fatalf("list devices: %d err=%v", len(devs), err)
	}
	if err := st.RevokeDevice(ctx, uid, devID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	devs, _ = st.ListDevices(ctx, uid)
	if len(devs) != 0 {
		t.Fatalf("после отзыва осталось %d устройств", len(devs))
	}
}

func TestPurchaseAndReferral(t *testing.T) {
	st, ctx := testStore(t)

	// Инвайтер.
	inviter, _, err := st.UpsertByPhone(ctx, uniqPhone(), "")
	if err != nil {
		t.Fatal(err)
	}
	code, _, err := st.Referral(ctx, inviter)
	if err != nil || code == "" {
		t.Fatalf("referral code: %q err=%v", code, err)
	}

	// Приглашённый по ?ref=code.
	invitee, _, err := st.UpsertByPhone(ctx, uniqPhone(), code)
	if err != nil {
		t.Fatal(err)
	}

	// Платёж приглашённого.
	pay, isNew, err := st.CreatePayment(ctx, invitee, "vpn-10gb-month", 300, "sbp", "idem-"+invitee)
	if err != nil || !isNew {
		t.Fatalf("create payment: isNew=%v err=%v", isNew, err)
	}
	// Идемпотентность создания.
	pay2, isNew2, err := st.CreatePayment(ctx, invitee, "vpn-10gb-month", 300, "sbp", "idem-"+invitee)
	if err != nil || isNew2 || pay2.ID != pay.ID {
		t.Fatalf("idempotent create: id1=%s id2=%s isNew=%v err=%v", pay.ID, pay2.ID, isNew2, err)
	}

	// Подтверждение вебхуком → платная подписка + реф-бонус инвайтеру.
	tx := "tx-" + uniq()
	if err := st.ConfirmPayment(ctx, pay.ID, tx, []byte(`{"status":"CONFIRMED"}`)); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	// Идемпотентность подтверждения.
	if err := st.ConfirmPayment(ctx, pay.ID, tx, []byte(`{"status":"CONFIRMED"}`)); err != nil {
		t.Fatalf("confirm idempotent: %v", err)
	}

	sub, err := st.ActiveSubscription(ctx, invitee)
	if err != nil || sub.Kind != "paid" || sub.TrafficBytesLimit != int64(config.PaidLimitBytes) {
		t.Fatalf("paid sub = %+v err=%v", sub, err)
	}

	// Инвайтер получил +1 ГБ (видно в limitBytes через bonus, у инвайтера триал).
	_, limit, err := st.Traffic(ctx, inviter)
	if err != nil {
		t.Fatal(err)
	}
	wantLimit := int64(config.TrialLimitBytes) + int64(config.ReferralBonus)
	if limit != wantLimit {
		t.Fatalf("inviter limit = %d, want %d (trial+bonus)", limit, wantLimit)
	}

	_, stats, err := st.Referral(ctx, inviter)
	if err != nil || stats.InvitedCount != 1 || stats.RewardedCount != 1 {
		t.Fatalf("referral stats = %+v err=%v", stats, err)
	}
}

func TestSessionRotation(t *testing.T) {
	st, ctx := testStore(t)
	uid, _, err := st.UpsertByPhone(ctx, uniqPhone(), "")
	if err != nil {
		t.Fatal(err)
	}
	hA, hB, hC := "hash-A-"+uniq(), "hash-B-"+uniq(), "hash-C-"+uniq()
	if _, err := st.CreateSession(ctx, uid, hA, "ua", "", time.Hour); err != nil {
		t.Fatal(err)
	}
	got, _, err := st.RotateSession(ctx, hA, hB, "ua", "", time.Hour)
	if err != nil || got != uid {
		t.Fatalf("rotate: uid=%s err=%v", got, err)
	}
	// Старый hash больше не валиден (защита от повторного использования).
	if _, _, err := st.RotateSession(ctx, hA, hC, "ua", "", time.Hour); err != ErrNotFound {
		t.Fatalf("повторное использование refresh должно падать ErrNotFound, got %v", err)
	}
}
