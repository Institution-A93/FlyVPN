package httpapi

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/institution-a93/flyvpn/services/account-api/internal/credentials"
	"github.com/institution-a93/flyvpn/services/account-api/internal/mobileconfig"
	"github.com/institution-a93/flyvpn/services/account-api/internal/store"
)

// POST /devices → выдаёт новый кред и .mobileconfig. Требует активную подписку (триал годится).
func (s *Server) issueDevice(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	sub, err := s.store.ActiveSubscription(r.Context(), uid)
	if errors.Is(err, store.ErrNotFound) {
		s.writeErr(w, http.StatusForbidden, "no_subscription", "no active subscription")
		return
	}
	if err != nil {
		s.fail(w, "device subscription", err)
		return
	}

	username, err := credentials.GenerateUsername()
	if err != nil {
		s.fail(w, "gen username", err)
		return
	}
	password, err := credentials.GeneratePassword()
	if err != nil {
		s.fail(w, "gen password", err)
		return
	}
	deviceID, _, err := s.store.IssueDevice(r.Context(), uid, username, credentials.NTHash(password))
	if err != nil {
		s.fail(w, "issue device", err)
		return
	}

	profile, err := mobileconfig.Render(s.template, mobileconfig.Fields{
		ProfileIdentifier: "pro.flynet.vpn." + uuid.NewString(),
		ProfileUUID:       uuid.NewString(),
		PayloadUUID:       uuid.NewString(),
		DisplayName:       s.cfg.DisplayName,
		OrgName:           s.cfg.Organization,
		VPNRemoteAddress:  s.cfg.VPNRemote,
		VPNRemoteID:       s.cfg.VPNRemoteID,
		EAPUsername:       username,
		EAPPassword:       password,
	})
	if err != nil {
		s.fail(w, "render profile", err)
		return
	}

	// config — base64 .mobileconfig (фронт сохраняет как файл). subscriptionInfo — текущее состояние.
	s.writeJSON(w, http.StatusOK, map[string]any{
		"deviceId": deviceID,
		"config":   base64.StdEncoding.EncodeToString(profile),
		"subscriptionInfo": map[string]any{
			"plan":      sub.Kind,
			"active":    sub.Active,
			"expiresAt": sub.ExpiresAt,
		},
	})
}

// GET /devices → список активных устройств (кредов).
func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	devs, err := s.store.ListDevices(r.Context(), userID(r))
	if err != nil {
		s.fail(w, "list devices", err)
		return
	}
	out := make([]map[string]any, 0, len(devs))
	for _, d := range devs {
		out = append(out, map[string]any{
			"id":         d.ID,
			"username":   d.Username,
			"framedIp":   d.FramedIP,
			"issuedAt":   d.IssuedAt,
			"lastUsedAt": d.LastUsedAt,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"devices": out})
}

// DELETE /devices/{id} → отзыв (revoked_at).
func (s *Server) revokeDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.RevokeDevice(r.Context(), userID(r), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.writeErr(w, http.StatusNotFound, "not_found", "device not found")
			return
		}
		s.fail(w, "revoke device", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
