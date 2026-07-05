package integrations

import (
	"strings"
	"testing"
)

func TestRegistryKeysUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range Registry {
		if s.Key == "" {
			t.Fatalf("integration with empty key: %+v", s)
		}
		if seen[s.Key] {
			t.Errorf("duplicate integration key: %s", s.Key)
		}
		seen[s.Key] = true
		if s.Key != strings.ToLower(s.Key) {
			t.Errorf("key %q must be lowercase", s.Key)
		}
	}
}

func TestTokenIntegrationsHavePaths(t *testing.T) {
	for _, s := range Registry {
		if s.Kind != KindToken {
			continue
		}
		if s.ConnectPath == "" {
			t.Errorf("%s: token integration missing ConnectPath", s.Key)
		}
		if !strings.HasPrefix(s.ConnectPath, "/api/integrations/") {
			t.Errorf("%s: ConnectPath %q must be under /api/integrations/", s.Key, s.ConnectPath)
		}
		if s.DisconnectPath == "" {
			t.Errorf("%s: token integration missing DisconnectPath", s.Key)
		}
		if len(s.Fields) == 0 {
			t.Errorf("%s: token integration has no credential fields", s.Key)
		}
		hasRequired := false
		for _, f := range s.Fields {
			if f.Required {
				hasRequired = true
			}
			if f.Flag == "" || f.JSON == "" {
				t.Errorf("%s: field with empty Flag or JSON: %+v", s.Key, f)
			}
		}
		if !hasRequired {
			t.Errorf("%s: token integration has no required fields", s.Key)
		}
	}
}

func TestKnowledgeIntegrationsHaveSourceType(t *testing.T) {
	for _, s := range Registry {
		if s.Kind != KindKnowledge {
			continue
		}
		if s.SourceType == "" {
			t.Errorf("%s: knowledge integration missing SourceType", s.Key)
		}
		if len(s.Fields) == 0 {
			t.Errorf("%s: knowledge integration has no credential fields", s.Key)
		}
	}
}

func TestFieldFlagsUniquePerIntegration(t *testing.T) {
	for _, s := range Registry {
		flags := map[string]bool{}
		jsons := map[string]bool{}
		for _, f := range s.Fields {
			if flags[f.Flag] {
				t.Errorf("%s: duplicate flag --%s", s.Key, f.Flag)
			}
			flags[f.Flag] = true
			if jsons[f.JSON] {
				t.Errorf("%s: duplicate JSON field %s", s.Key, f.JSON)
			}
			jsons[f.JSON] = true
		}
	}
}

func TestGet(t *testing.T) {
	if Get("cloudflare") == nil {
		t.Error("Get(cloudflare) returned nil")
	}
	if Get("does-not-exist") != nil {
		t.Error("Get(does-not-exist) should return nil")
	}
}
