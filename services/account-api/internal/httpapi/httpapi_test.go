package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	accountapi "github.com/institution-a93/flyvpn/services/account-api"
	"github.com/institution-a93/flyvpn/services/account-api/internal/config"
	"github.com/institution-a93/flyvpn/services/account-api/internal/store"
	"github.com/institution-a93/flyvpn/services/account-api/internal/token"

	"log/slog"
)

// e2e через httptest: OTP-prototype login → /me → /subscription → /traffic → POST /devices.
// Запускается только при ACCOUNTAPI_TEST_DSN.
func TestEndToEndHTTP(t *testing.T) {
	dsn := os.Getenv("ACCOUNTAPI_TEST_DSN")
	if dsn == "" {
		t.Skip("ACCOUNTAPI_TEST_DSN не задан")
	}
	ctx := context.Background()
	st, err := store.New(ctx, dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	cfg := config.Config{
		VPNRemote: "vpn.example.com", VPNRemoteID: "vpn.example.com",
		Organization: "FLY VPN", DisplayName: "FLY VPN",
		JWTSecret: "test-secret", AccessTTL: 15 * time.Minute, RefreshTTL: time.Hour,
		PublicBaseURL: "https://flynet.pro",
	}
	tk := token.New(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)
	srv := New(cfg, st, tk, nil, nil, accountapi.ProfileTemplate, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ts := httptest.NewServer(srv.Routes())
	defer ts.Close()

	phone := "+79" + time.Now().Format("050405000")

	// 1. OTP request (prototype: код 0000).
	mustPost(t, ts.URL+"/api/v1/auth/otp/request", "", map[string]string{"phone": phone}, http.StatusOK)

	// 2. OTP verify → токены.
	var sess sessionJSON
	doJSON(t, mustPost(t, ts.URL+"/api/v1/auth/otp/verify", "",
		map[string]string{"phone": phone, "code": "0000"}, http.StatusOK), &sess)
	if sess.AccessToken == "" || sess.User.ReferralCode == "" {
		t.Fatalf("bad session: %+v", sess)
	}
	bearer := sess.AccessToken

	// 3. /me требует токен.
	mustGet(t, ts.URL+"/api/v1/me", "", http.StatusUnauthorized)
	var me userJSON
	doJSON(t, mustGet(t, ts.URL+"/api/v1/me", bearer, http.StatusOK), &me)
	if me.Phone != phone {
		t.Fatalf("me.phone = %q, want %q", me.Phone, phone)
	}

	// 4. /subscription — триал активен.
	var sub map[string]any
	doJSON(t, mustGet(t, ts.URL+"/api/v1/subscription", bearer, http.StatusOK), &sub)
	if sub["plan"] != "trial" || sub["active"] != true {
		t.Fatalf("subscription = %+v", sub)
	}

	// 5. /traffic — лимит 300 МБ.
	var tr map[string]int64
	doJSON(t, mustGet(t, ts.URL+"/api/v1/traffic", bearer, http.StatusOK), &tr)
	if tr["limitBytes"] != int64(config.TrialLimitBytes) {
		t.Fatalf("traffic = %+v", tr)
	}

	// 6. POST /devices — выдаёт валидный .mobileconfig.
	var dev map[string]any
	doJSON(t, mustPost(t, ts.URL+"/api/v1/devices", bearer, nil, http.StatusOK), &dev)
	raw, err := base64.StdEncoding.DecodeString(dev["config"].(string))
	if err != nil || !strings.Contains(string(raw), "com.apple.vpn.managed") {
		t.Fatalf("bad mobileconfig: err=%v body=%.80q", err, string(raw))
	}
}

func mustPost(t *testing.T, url, bearer string, body any, want int) *http.Response {
	t.Helper()
	var rdr *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req, _ := http.NewRequest(http.MethodPost, url, rdr)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return do(t, req, want)
}

func mustGet(t *testing.T, url, bearer string, want int) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return do(t, req, want)
}

func do(t *testing.T, req *http.Request, want int) *http.Response {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status %d, want %d", req.Method, req.URL.Path, resp.StatusCode, want)
	}
	return resp
}

func doJSON(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
