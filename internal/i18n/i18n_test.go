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
