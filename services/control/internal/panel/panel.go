// Package panel — тонкая HTML-панель для НЕтехнического суппорта поверх contract/fleet.
// Server-rendered (html/template, авто-экранирование), вход по operator-токену в cookie.
// Все записи идут через contract (единственный писатель). За приватной сетью (ADR-0021).
package panel

import (
	"crypto/subtle"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/institution-a93/flyvpn/services/control/internal/contract"
	"github.com/institution-a93/flyvpn/services/control/internal/fleet"
)

const cookieName = "op_token"

// Deps — что нужно панели.
type Panel struct {
	token string
	plans map[string]contract.Plan
	ct    *contract.Contract
	fl    *fleet.Fleet
	log   *slog.Logger
	tpl   *template.Template
}

func New(token string, plans map[string]contract.Plan, ct *contract.Contract, fl *fleet.Fleet, log *slog.Logger) *Panel {
	return &Panel{token: token, plans: plans, ct: ct, fl: fl, log: log, tpl: template.Must(template.New("").Funcs(funcs).Parse(tpls))}
}

// Register вешает ручки панели на общий mux (рядом с API).
func (p *Panel) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /panel/login", p.loginForm)
	mux.HandleFunc("POST /panel/login", p.login)
	mux.HandleFunc("GET /panel", p.auth(p.home))
	mux.HandleFunc("GET /panel/user", p.auth(p.user))
	mux.HandleFunc("POST /panel/user/provision", p.auth(p.doProvision))
	mux.HandleFunc("POST /panel/user/renew", p.auth(p.doRenew))
	mux.HandleFunc("POST /panel/user/revoke", p.auth(p.doRevoke))
	mux.HandleFunc("GET /panel/nodes", p.auth(p.nodes))
}

// --- auth (cookie с operator-токеном) ---

func (p *Panel) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil || p.token == "" || subtle.ConstantTimeCompare([]byte(c.Value), []byte(p.token)) != 1 {
			http.Redirect(w, r, "/panel/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (p *Panel) loginForm(w http.ResponseWriter, _ *http.Request) { p.render(w, "login", nil) }

func (p *Panel) login(w http.ResponseWriter, r *http.Request) {
	tok := r.FormValue("token")
	if p.token == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(p.token)) != 1 {
		p.render(w, "login", map[string]any{"Err": "неверный токен"})
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: tok, Path: "/panel", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/panel", http.StatusSeeOther)
}

// --- страницы ---

func (p *Panel) home(w http.ResponseWriter, _ *http.Request) { p.render(w, "home", nil) }

func (p *Panel) planCodes() []string {
	out := make([]string, 0, len(p.plans))
	for k := range p.plans {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (p *Panel) user(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Redirect(w, r, "/panel", http.StatusSeeOther)
		return
	}
	ctx := r.Context()
	cr, err := p.ct.Get(ctx, id)
	data := map[string]any{"ID": id, "Plans": p.planCodes()}
	if err == contract.ErrNotFound {
		data["NoCred"] = true
		p.render(w, "user", data)
		return
	}
	if err != nil {
		p.fail(w, "get", err)
		return
	}
	used, _ := p.ct.Usage(ctx, id)
	data["Cr"] = cr
	data["Used"] = used
	data["State"] = contract.State(cr, used)
	p.render(w, "user", data)
}

func (p *Panel) doProvision(w http.ResponseWriter, r *http.Request) { p.act(w, r, "provision") }
func (p *Panel) doRenew(w http.ResponseWriter, r *http.Request)     { p.act(w, r, "renew") }
func (p *Panel) doRevoke(w http.ResponseWriter, r *http.Request)    { p.act(w, r, "revoke") }

func (p *Panel) act(w http.ResponseWriter, r *http.Request, kind string) {
	id := r.FormValue("id")
	ctx := r.Context()
	var err error
	switch kind {
	case "provision":
		_, err = p.ct.Provision(ctx, id, p.plans[r.FormValue("plan")])
	case "renew":
		err = p.ct.Renew(ctx, id, p.plans[r.FormValue("plan")])
	case "revoke":
		err = p.ct.Revoke(ctx, id)
	}
	if err != nil && err != contract.ErrNotFound {
		p.fail(w, kind, err)
		return
	}
	http.Redirect(w, r, "/panel/user?id="+id, http.StatusSeeOther)
}

func (p *Panel) nodes(w http.ResponseWriter, r *http.Request) {
	ns, err := p.fl.List(r.Context())
	if err != nil {
		p.fail(w, "nodes", err)
		return
	}
	p.render(w, "nodes", map[string]any{"Nodes": ns})
}

// --- render/helpers ---

func (p *Panel) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := p.tpl.ExecuteTemplate(w, name, data); err != nil {
		p.log.Error("panel render", "tpl", name, "err", err)
	}
}

func (p *Panel) fail(w http.ResponseWriter, step string, err error) {
	p.log.Error("panel", "step", step, "err", err)
	http.Error(w, "internal error", 500)
}

var funcs = template.FuncMap{
	"gb":   func(b int64) string { return fmt.Sprintf("%.2f ГБ", float64(b)/(1<<30)) },
	"gbp":  func(b *int64) string { if b == nil { return "∞" }; return fmt.Sprintf("%.0f ГБ", float64(*b)/(1<<30)) },
	"date": func(t *time.Time) string { if t == nil { return "—" }; return t.Format("2006-01-02 15:04") },
}

const tpls = `
{{define "head"}}<!doctype html><meta charset=utf-8><title>FLY VPN · control</title>
<style>body{font:15px system-ui;margin:2rem;max-width:760px}a{color:#06c}input,select,button{font:inherit;padding:.4rem}
.s-active{color:#2e7d32;font-weight:600}.s-lapsed{color:#b8860b;font-weight:600}.s-revoked{color:#c0392b;font-weight:600}
table{border-collapse:collapse}td,th{border:1px solid #ddd;padding:.4rem .7rem;text-align:left}form{display:inline}</style>
<p><a href=/panel>поиск</a> · <a href=/panel/nodes>узлы</a></p>{{end}}

{{define "login"}}{{template "head"}}<h2>Вход</h2>
{{if .Err}}<p style=color:#c00>{{.Err}}</p>{{end}}
<form method=post action=/panel/login><input name=token type=password placeholder="operator token" autofocus> <button>Войти</button></form>{{end}}

{{define "home"}}{{template "head"}}<h2>Найти клиента</h2>
<form method=get action=/panel/user><input name=id placeholder="user id (uuid)" size=40 autofocus> <button>Открыть</button></form>{{end}}

{{define "user"}}{{template "head"}}<h2>Клиент</h2><p>id: <code>{{.ID}}</code></p>
{{if .NoCred}}<p>Креда нет. Выдать:</p>
  <form method=post action=/panel/user/provision><input type=hidden name=id value="{{.ID}}">
  <select name=plan>{{range .Plans}}<option>{{.}}</option>{{end}}</select> <button>Provision</button></form>
{{else}}{{with .Cr}}
  <table>
   <tr><th>статус</th><td class="s-{{$.State}}">{{$.State}}</td></tr>
   <tr><th>username</th><td><code>{{.Username}}</code></td></tr>
   <tr><th>framed_ip</th><td>{{.FramedIP}}</td></tr>
   <tr><th>срок до</th><td>{{date .ExpiresAt}}</td></tr>
   <tr><th>трафик</th><td>{{gb $.Used}} / {{gbp .TrafficCap}}</td></tr>
  </table>
  <h3>Действия</h3>
  <form method=post action=/panel/user/renew><input type=hidden name=id value="{{.UserID}}">
   <select name=plan>{{range $.Plans}}<option>{{.}}</option>{{end}}</select> <button>Продлить</button></form>
  <form method=post action=/panel/user/revoke onsubmit="return confirm('Отозвать доступ?')"><input type=hidden name=id value="{{.UserID}}"> <button>Отозвать</button></form>
{{end}}{{end}}{{end}}

{{define "nodes"}}{{template "head"}}<h2>Узлы</h2><table><tr><th>регион</th><th>роль</th><th>IP</th><th>статус</th><th>heartbeat</th></tr>
{{range .Nodes}}<tr><td>{{.Region}}</td><td>{{.Role}}</td><td>{{.PublicIP}}</td><td class="s-{{if eq .Status "up"}}active{{else}}revoked{{end}}">{{.Status}}</td><td>{{date .LastHeartbeat}}</td></tr>{{end}}
</table>{{end}}
`
