package httpapi

import (
	"errors"
	"net/http"

	"github.com/institution-a93/flyvpn/services/account-api/internal/store"
)

// GET /me → {id, telegramUsername, phone, referralCode}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.GetMe(r.Context(), userID(r))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.writeErr(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		s.fail(w, "get me", err)
		return
	}
	s.writeJSON(w, http.StatusOK, toUserJSON(u))
}

// DELETE /me → 204 (отзывает креды + сессии, чистит PII)
func (s *Server) deleteMe(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteAccount(r.Context(), userID(r)); err != nil {
		s.fail(w, "delete account", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /subscription → {plan, active, expiresAt, autoRenew}
func (s *Server) subscription(w http.ResponseWriter, r *http.Request) {
	sub, err := s.store.ActiveSubscription(r.Context(), userID(r))
	if errors.Is(err, store.ErrNotFound) {
		// нет активной подписки — отдаём неактивное состояние, а не ошибку.
		s.writeJSON(w, http.StatusOK, map[string]any{
			"plan": nil, "active": false, "expiresAt": nil, "autoRenew": false,
		})
		return
	}
	if err != nil {
		s.fail(w, "subscription", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"plan":      sub.Kind, // "trial" | "paid"
		"active":    sub.Active,
		"expiresAt": sub.ExpiresAt,
		"autoRenew": false, // decision #5
	})
}

// GET /traffic → {usedBytes, limitBytes}
func (s *Server) traffic(w http.ResponseWriter, r *http.Request) {
	used, limit, err := s.store.Traffic(r.Context(), userID(r))
	if errors.Is(err, store.ErrNotFound) {
		s.writeJSON(w, http.StatusOK, map[string]int64{"usedBytes": 0, "limitBytes": 0})
		return
	}
	if err != nil {
		s.fail(w, "traffic", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]int64{"usedBytes": used, "limitBytes": limit})
}

// GET /referral → {link, invitedCount, rewardedCount}
func (s *Server) referral(w http.ResponseWriter, r *http.Request) {
	code, stats, err := s.store.Referral(r.Context(), userID(r))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.writeErr(w, http.StatusNotFound, "not_found", "user not found")
			return
		}
		s.fail(w, "referral", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"link":          s.cfg.PublicBaseURL + "/?ref=" + code,
		"invitedCount":  stats.InvitedCount,
		"rewardedCount": stats.RewardedCount,
	})
}
