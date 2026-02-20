//go:build unit

package service

import (
	"testing"
)

func TestPlatformProxyRules_GetRule_Anthropic(t *testing.T) {
	rules := NewPlatformProxyRules()
	rule, ok := rules.GetRule("anthropic")
	if !ok {
		t.Fatal("expected anthropic rule to exist")
	}
	if rule.Platform != "anthropic" {
		t.Errorf("expected platform anthropic, got %s", rule.Platform)
	}
	if rule.MaxAccountsPerIP != 5 {
		t.Errorf("expected MaxAccountsPerIP 5, got %d", rule.MaxAccountsPerIP)
	}
	if !rule.StickyIP {
		t.Error("expected StickyIP true for anthropic")
	}
	expectRegions := []string{"us-east", "us-central", "us-west"}
	if len(rule.PreferredRegions) != len(expectRegions) {
		t.Fatalf("expected %d regions, got %d", len(expectRegions), len(rule.PreferredRegions))
	}
	for i, r := range expectRegions {
		if rule.PreferredRegions[i] != r {
			t.Errorf("region[%d]: expected %s, got %s", i, r, rule.PreferredRegions[i])
		}
	}
}

func TestPlatformProxyRules_GetRule_Gemini(t *testing.T) {
	rules := NewPlatformProxyRules()
	rule, ok := rules.GetRule("gemini")
	if !ok {
		t.Fatal("expected gemini rule to exist")
	}
	if rule.Platform != "gemini" {
		t.Errorf("expected platform gemini, got %s", rule.Platform)
	}
	if rule.MaxAccountsPerIP != 10 {
		t.Errorf("expected MaxAccountsPerIP 10, got %d", rule.MaxAccountsPerIP)
	}
	if !rule.StickyIP {
		t.Error("expected StickyIP true for gemini")
	}
	expectRegions := []string{"us-east", "us-west", "eu-west"}
	if len(rule.PreferredRegions) != len(expectRegions) {
		t.Fatalf("expected %d regions, got %d", len(expectRegions), len(rule.PreferredRegions))
	}
	for i, r := range expectRegions {
		if rule.PreferredRegions[i] != r {
			t.Errorf("region[%d]: expected %s, got %s", i, r, rule.PreferredRegions[i])
		}
	}
}

func TestPlatformProxyRules_GetRule_OpenAI(t *testing.T) {
	rules := NewPlatformProxyRules()
	rule, ok := rules.GetRule("openai")
	if !ok {
		t.Fatal("expected openai rule to exist")
	}
	if rule.Platform != "openai" {
		t.Errorf("expected platform openai, got %s", rule.Platform)
	}
	if rule.MaxAccountsPerIP != 8 {
		t.Errorf("expected MaxAccountsPerIP 8, got %d", rule.MaxAccountsPerIP)
	}
	if !rule.StickyIP {
		t.Error("expected StickyIP true for openai")
	}
	expectRegions := []string{"us-east", "us-west"}
	if len(rule.PreferredRegions) != len(expectRegions) {
		t.Fatalf("expected %d regions, got %d", len(expectRegions), len(rule.PreferredRegions))
	}
	for i, r := range expectRegions {
		if rule.PreferredRegions[i] != r {
			t.Errorf("region[%d]: expected %s, got %s", i, r, rule.PreferredRegions[i])
		}
	}
}

func TestPlatformProxyRules_GetRule_Unknown(t *testing.T) {
	rules := NewPlatformProxyRules()
	_, ok := rules.GetRule("unknown-platform")
	if ok {
		t.Error("expected unknown platform to return false")
	}
}

func TestPlatformProxyRules_AllRules(t *testing.T) {
	rules := NewPlatformProxyRules()
	all := rules.AllRules()
	if len(all) != 3 {
		t.Errorf("expected 3 platform rules, got %d", len(all))
	}
	for _, name := range []string{"anthropic", "gemini", "openai"} {
		if _, ok := all[name]; !ok {
			t.Errorf("expected platform %s in AllRules", name)
		}
	}
}
