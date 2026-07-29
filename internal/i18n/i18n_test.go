package i18n

import "testing"

func TestNormalizeAndTranslate(t *testing.T) {
	if Normalize("en_US") != "en-US" {
		t.Fatal("normalize")
	}
	if T("es", "settings") != "Ajustes" {
		t.Fatal(T("es", "settings"))
	}
	if T("unknown", "quit") != "sair" {
		t.Fatal("fallback")
	}
}

func TestEveryModuleStringHasAllLanguages(t *testing.T) {
	for key := range modules["pt-BR"] {
		for _, language := range []string{"en-US", "es"} {
			if modules[language][key] == "" {
				t.Errorf("chave %q ausente em %s", key, language)
			}
		}
	}
}

func TestLocalizeError(t *testing.T) {
	got := LocalizeError("en-US", "Docker indisponível: permission denied")
	if got != "Docker unavailable: permission denied" {
		t.Fatalf("erro não traduzido: %q", got)
	}
}

func TestSecurityTextUsesSelectedLanguage(t *testing.T) {
	if got := SecurityText("pt-BR", "Container runs as root"); got != "Container executa como root" {
		t.Fatalf("texto de segurança não traduzido: %q", got)
	}
	if got := SecurityText("es", "Enable no-new-privileges"); got != "Habilitar no-new-privileges" {
		t.Fatalf("texto de seguridad no traducido: %q", got)
	}
	if got := SecurityText("en-US", "Container runs as root"); got != "Container runs as root" {
		t.Fatalf("English security text changed: %q", got)
	}
}

func TestLocalizeHardeningError(t *testing.T) {
	got := LocalizeError("en-US", "container mudou desde a inspeção; execute o hardening novamente")
	if got != "container changed since inspection; run hardening again" {
		t.Fatalf("hardening error not localized: %q", got)
	}
}
