// Package accountapi exposes embedded assets shared by the service.
package accountapi

import _ "embed"

// ProfileTemplate — шаблон Apple Configuration Profile (IKEv2 + EAP-MSCHAPv2).
// Идентичен config-api/profile.mobileconfig.tmpl: устройство получает тот же профиль,
// неважно, выдан он по legacy Digiseller-коду (config-api) или из кабинета (account-api).
// Токены {{NAME}} подставляет internal/mobileconfig.
//
//go:embed profile.mobileconfig.tmpl
var ProfileTemplate string
