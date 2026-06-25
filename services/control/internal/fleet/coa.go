package fleet

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// Disconnector шлёт RADIUS Disconnect-Request на DAE-листенер узла (strongSwan
// eap-radius.dae). Интерфейс — чтобы мокать в тестах и не зависеть от radclient в коде.
type Disconnector interface {
	Disconnect(ctx context.Context, nasIP, username string) error
}

// liveSession — открытая сессия юзера (acctstoptime IS NULL) с узлом, где она живёт.
type liveSession struct {
	NASIP    string
	Username string
}

// openSessions возвращает живые сессии всех кредов юзера (radacct ↔ auth_credentials).
func (f *Fleet) openSessions(ctx context.Context, userID string) ([]liveSession, error) {
	rows, err := f.pool.Query(ctx, `
		SELECT DISTINCT host(a.nasipaddress), a.username
		  FROM radacct a JOIN auth_credentials c ON c.username = a.username
		 WHERE c.user_id = $1 AND a.acctstoptime IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []liveSession
	for rows.Next() {
		var s liveSession
		if err := rows.Scan(&s.NASIP, &s.Username); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DisconnectUser рвёт все живые сессии юзера через DAE (анти-локаут: при превышении
// капа/срока посреди сессии или при revoke — клиент реконнектится и попадает в
// restricted-пул либо, при revoke, не пускается вовсе). Идемпотентно: нет сессий — no-op.
func (f *Fleet) DisconnectUser(ctx context.Context, userID string, dis Disconnector) error {
	sessions, err := f.openSessions(ctx, userID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, s := range sessions {
		if err := dis.Disconnect(ctx, s.NASIP, s.Username); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("disconnect %s@%s: %w", s.Username, s.NASIP, err)
		}
	}
	return firstErr
}

// RadclientDisconnector — реализация поверх freeradius `radclient` (есть на control-узле,
// т.к. FreeRADIUS там же). Шлёт Disconnect-Request с User-Name на nasIP:port.
type RadclientDisconnector struct {
	Port   int    // DAE-порт узла (ingress_dae_port, по умолчанию 3799)
	Secret string // общий секрет DAE (ingress_dae_secret)
	Bin    string // путь к radclient (по умолчанию "radclient")
}

func (r RadclientDisconnector) Disconnect(ctx context.Context, nasIP, username string) error {
	bin := r.Bin
	if bin == "" {
		bin = "radclient"
	}
	target := net.JoinHostPort(nasIP, strconv.Itoa(r.Port))
	cmd := exec.CommandContext(ctx, bin, "-x", target, "disconnect", r.Secret)
	cmd.Stdin = strings.NewReader("User-Name=" + username + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("radclient disconnect %s: %w (%s)", target, err, strings.TrimSpace(string(out)))
	}
	return nil
}
