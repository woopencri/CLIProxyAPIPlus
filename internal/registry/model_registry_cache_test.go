package registry

import "testing"

type testOAuthAliasEntry struct {
	Name  string
	Alias string
	Fork  bool
}

type testOAuthAliasConfig struct {
	OAuthModelAlias map[string][]testOAuthAliasEntry
}

func TestGetAvailableModelsReturnsClonedSnapshots(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "OpenAI", []*ModelInfo{{ID: "m1", OwnedBy: "team-a", DisplayName: "Model One"}})

	first := r.GetAvailableModels("openai")
	if len(first) != 1 {
		t.Fatalf("expected 1 model, got %d", len(first))
	}
	first[0]["id"] = "mutated"
	first[0]["display_name"] = "Mutated"

	second := r.GetAvailableModels("openai")
	if got := second[0]["id"]; got != "m1" {
		t.Fatalf("expected cached snapshot to stay isolated, got id %v", got)
	}
	if got := second[0]["display_name"]; got != "Model One" {
		t.Fatalf("expected cached snapshot to stay isolated, got display_name %v", got)
	}
}

func TestGetAvailableModelsInvalidatesCacheOnRegistryChanges(t *testing.T) {
	r := newTestModelRegistry()
	r.RegisterClient("client-1", "OpenAI", []*ModelInfo{{ID: "m1", OwnedBy: "team-a", DisplayName: "Model One"}})

	models := r.GetAvailableModels("openai")
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if got := models[0]["display_name"]; got != "Model One" {
		t.Fatalf("expected initial display_name Model One, got %v", got)
	}

	r.RegisterClient("client-1", "OpenAI", []*ModelInfo{{ID: "m1", OwnedBy: "team-a", DisplayName: "Model One Updated"}})
	models = r.GetAvailableModels("openai")
	if got := models[0]["display_name"]; got != "Model One Updated" {
		t.Fatalf("expected updated display_name after cache invalidation, got %v", got)
	}

	r.SuspendClientModel("client-1", "m1", "manual")
	models = r.GetAvailableModels("openai")
	if len(models) != 0 {
		t.Fatalf("expected no available models after suspension, got %d", len(models))
	}

	r.ResumeClientModel("client-1", "m1")
	models = r.GetAvailableModels("openai")
	if len(models) != 1 {
		t.Fatalf("expected model to reappear after resume, got %d", len(models))
	}
}

func TestRegisterClient_InjectsOAuthAliasModels(t *testing.T) {
	r := newTestModelRegistry()
	cfg := &testOAuthAliasConfig{OAuthModelAlias: map[string][]testOAuthAliasEntry{
		"kiro": {
			{Name: "kiro-claude-sonnet-4-6", Alias: "claude-sonnet-4-6", Fork: true},
			{Name: "kiro-claude-sonnet-4-6", Alias: "claude-sonnet-4-6-latest", Fork: false},
		},
	}}

	r.RegisterClient("client-kiro", "kiro", []*ModelInfo{{
		ID:          "kiro-claude-sonnet-4-6",
		OwnedBy:     "amazon",
		Type:        "claude",
		DisplayName: "Kiro Sonnet",
	}}, cfg)

	models := r.GetModelsForClient("client-kiro")
	if len(models) != 2 {
		t.Fatalf("expected 2 client models after alias injection, got %d", len(models))
	}
	if models[0].ID != "kiro-claude-sonnet-4-6" {
		t.Fatalf("unexpected source model order: %+v", models[0])
	}
	if models[1].ID != "claude-sonnet-4-6" {
		t.Fatalf("expected fork alias model to be injected, got %+v", models[1])
	}
	if got := r.GetModelProviders("claude-sonnet-4-6"); len(got) != 1 || got[0] != "kiro" {
		t.Fatalf("expected alias provider to resolve to kiro, got %v", got)
	}
	available := r.GetAvailableModels("openai")
	if len(available) != 2 {
		t.Fatalf("expected alias in available models, got %d entries", len(available))
	}
}
