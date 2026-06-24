package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/institution-a93/flyvpn/services/account-api/internal/auth"
	"github.com/institution-a93/flyvpn/services/account-api/internal/store"
	"github.com/institution-a93/flyvpn/services/account-api/internal/token"
)

// otpTTL — срок жизни одноразового кода (§5.2 ~5 мин).
const otpTTL = 5 * time.Minute

// readBody читает тело запроса целиком (≤1 МБ). Для telegramLogin нужен повторный разбор.
func readBody(r *http.Request) []byte {
	b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	return b
}

// userJSON — представление пользователя в ответах (поля под фронт src/lib/types.ts).
type userJSON struct {
	ID               string `json:"id"`
	TelegramUsername string `json:"telegramUsername"`
	Phone            string `json:"phone"`
	ReferralCode     string `json:"referralCode"`
}

type sessionJSON struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	User         userJSON `json:"user"`
}

// issueSession создаёт сессию (refresh) + access-JWT и возвращает их вместе с user.
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, uid string) {
	plaintext, hash, err := token.NewRefresh()
	if err != nil {
		s.fail(w, "gen refresh", err)
		return
	}
	if _, err := s.store.CreateSession(r.Context(), uid, hash, r.UserAgent(), clientIP(r), s.tokens.RefreshTTL()); err != nil {
		s.fail(w, "create session", err)
		return
	}
	access, err := s.tokens.Access(uid)
	if err != nil {
		s.fail(w, "gen access", err)
		return
	}
	me, err := s.store.GetMe(r.Context(), uid)
	if err != nil {
		s.fail(w, "get me", err)
		return
	}
	s.writeJSON(w, http.StatusOK, sessionJSON{
		AccessToken:  access,
		RefreshToken: plaintext,
		User:         toUserJSON(me),
	})
}

func toUserJSON(u store.User) userJSON {
	return userJSON{ID: u.ID, TelegramUsername: u.TelegramUsername, Phone: u.Phone, ReferralCode: u.ReferralCode}
}

// POST /auth/otp/request {phone}
func (s *Server) otpRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.Phone) == "" {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "phone required")
		return
	}
	phone := strings.TrimSpace(body.Phone)

	code, err := auth.GenerateCode(s.twilio != nil)
	if err != nil {
		s.fail(w, "gen otp", err)
		return
	}
	if err := s.store.StoreOTP(r.Context(), phone, auth.HashCode(phone, code), clientIP(r), otpTTL); err != nil {
		if errors.Is(err, store.ErrRateLimited) {
			s.writeErr(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		s.fail(w, "store otp", err)
		return
	}
	if s.twilio != nil {
		if err := s.twilio.Send(r.Context(), phone, code); err != nil {
			s.fail(w, "send sms", err)
			return
		}
	} else {
		s.log.Info("otp prototype mode", "phone", phone) // код фиксирован 0000
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// POST /auth/otp/verify {phone, code, ref?}
func (s *Server) otpVerify(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
		Ref   string `json:"ref"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Phone == "" || body.Code == "" {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "phone and code required")
		return
	}
	phone := strings.TrimSpace(body.Phone)
	if err := s.store.VerifyOTP(r.Context(), phone, auth.HashCode(phone, body.Code)); err != nil {
		if errors.Is(err, store.ErrOTPInvalid) {
			s.writeErr(w, http.StatusUnauthorized, "otp_invalid", "invalid or expired code")
			return
		}
		s.fail(w, "verify otp", err)
		return
	}
	uid, _, err := s.store.UpsertByPhone(r.Context(), phone, strings.TrimSpace(body.Ref))
	if err != nil {
		s.fail(w, "upsert phone", err)
		return
	}
	s.issueSession(w, r, uid)
}

// POST /auth/telegram — payload Telegram login-widget (+ опц. ref).
func (s *Server) telegramLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.TelegramEnabled() {
		s.writeErr(w, http.StatusServiceUnavailable, "unavailable", "telegram login not configured")
		return
	}
	// Числа сохраняем как json.Number (строковое представление для data-check-string).
	dec := json.NewDecoder(bytes.NewReader(readBody(r)))
	dec.UseNumber()
	raw := map[string]any{}
	if err := dec.Decode(&raw); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid payload")
		return
	}

	ref, _ := raw["ref"].(string)
	delete(raw, "ref")

	hash, _ := raw["hash"].(string)
	data := map[string]string{}
	for k, v := range raw {
		if k == "hash" {
			continue
		}
		data[k] = toStr(v)
	}
	if err := auth.VerifyTelegram(s.cfg.TelegramBotToken, data, hash); err != nil {
		s.writeErr(w, http.StatusUnauthorized, "telegram_auth", "signature check failed")
		return
	}

	tgID, err := strconv.ParseInt(data["id"], 10, 64)
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid telegram id")
		return
	}
	uid, _, err := s.store.UpsertByTelegram(r.Context(), store.TelegramProfile{
		TelegramID: tgID,
		Username:   data["username"],
		FirstName:  data["first_name"],
		LastName:   data["last_name"],
		PhotoURL:   data["photo_url"],
	}, ref)
	if err != nil {
		s.fail(w, "upsert telegram", err)
		return
	}
	s.issueSession(w, r, uid)
}

// POST /auth/refresh {refreshToken}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := decodeJSON(r, &body); err != nil || body.RefreshToken == "" {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "refreshToken required")
		return
	}
	oldHash := token.HashRefresh(body.RefreshToken)
	newPlain, newHash, err := token.NewRefresh()
	if err != nil {
		s.fail(w, "gen refresh", err)
		return
	}
	uid, _, err := s.store.RotateSession(r.Context(), oldHash, newHash, r.UserAgent(), clientIP(r), s.tokens.RefreshTTL())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid refresh token")
			return
		}
		s.fail(w, "rotate session", err)
		return
	}
	access, err := s.tokens.Access(uid)
	if err != nil {
		s.fail(w, "gen access", err)
		return
	}
	me, err := s.store.GetMe(r.Context(), uid)
	if err != nil {
		s.fail(w, "get me", err)
		return
	}
	s.writeJSON(w, http.StatusOK, sessionJSON{AccessToken: access, RefreshToken: newPlain, User: toUserJSON(me)})
}

// POST /auth/logout {refreshToken}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	_ = decodeJSON(r, &body)
	if body.RefreshToken != "" {
		if err := s.store.RevokeSessionByHash(r.Context(), token.HashRefresh(body.RefreshToken)); err != nil {
			s.fail(w, "revoke session", err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func toStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		return ""
	}
}
