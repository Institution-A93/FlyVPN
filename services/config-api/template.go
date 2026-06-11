// Package configapi exposes embedded assets shared by the service.
package configapi

import _ "embed"

// ProfileTemplate — шаблон Apple Configuration Profile (IKEv2 + EAP-MSCHAPv2).
// Токены {{NAME}} подставляются генератором (internal/mobileconfig).
//
//go:embed profile.mobileconfig.tmpl
var ProfileTemplate string
