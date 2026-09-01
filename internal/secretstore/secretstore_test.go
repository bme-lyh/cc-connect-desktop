package secretstore

import "testing"

func TestSecretStore_ReferenceRoundTrip(t *testing.T) {
	key := "bot-id/telegram/token"
	ref := Reference(key)
	got, ok := ParseReference(ref)
	if !ok || got != key {
		t.Fatalf("ParseReference(%q) = %q, %v", ref, got, ok)
	}
	if _, ok := ParseReference("plain-secret"); ok {
		t.Fatal("plain value must not parse as a reference")
	}
}

func TestSecretStore_MemoryDoesNotExposeMissingValues(t *testing.T) {
	store := NewMemory()
	if err := store.Set("bot/platform/token", "sensitive-value"); err != nil {
		t.Fatal(err)
	}
	value, err := store.Get("bot/platform/token")
	if err != nil || value != "sensitive-value" {
		t.Fatalf("Get() = %q, %v", value, err)
	}
	if err := store.Delete("bot/platform/token"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("bot/platform/token"); err == nil {
		t.Fatal("deleted value should be unavailable")
	}
}
