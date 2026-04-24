package http

import (
	"testing"
)

func TestHashAPIKey_Deterministic(t *testing.T) {
	h1 := hashAPIKey("moak_testkey123")
	h2 := hashAPIKey("moak_testkey123")
	if h1 != h2 {
		t.Errorf("hashAPIKey not deterministic: %q != %q", h1, h2)
	}
}

func TestHashAPIKey_Length(t *testing.T) {
	h := hashAPIKey("moak_testkey123")
	if len(h) != 64 {
		t.Errorf("expected SHA-256 hex length 64, got %d", len(h))
	}
}

func TestHashAPIKey_DifferentKeys(t *testing.T) {
	h1 := hashAPIKey("moak_key1")
	h2 := hashAPIKey("moak_key2")
	if h1 == h2 {
		t.Error("different keys must produce different hashes")
	}
}

func TestRateBucket_AllowsUnderLimit(t *testing.T) {
	b := &rateBucket{}
	for i := 0; i < burstRateLimit; i++ {
		if !b.allow() {
			t.Fatalf("request %d should be allowed (under limit)", i+1)
		}
	}
}

func TestRateBucket_BlocksAtLimit(t *testing.T) {
	b := &rateBucket{}
	for i := 0; i < burstRateLimit; i++ {
		b.allow()
	}
	if b.allow() {
		t.Error("request exceeding burst limit should be blocked")
	}
}

func TestGenerateAPIKeyString_Format(t *testing.T) {
	key, err := generateAPIKeyString()
	if err != nil {
		t.Fatalf("generateAPIKeyString returned error: %v", err)
	}
	if len(key) != 5+32 {
		t.Errorf("expected key length %d, got %d", 5+32, len(key))
	}
	if key[:5] != "moak_" {
		t.Errorf("expected key prefix 'moak_', got %q", key[:5])
	}
}

func TestGenerateAPIKeyString_Unique(t *testing.T) {
	k1, _ := generateAPIKeyString()
	k2, _ := generateAPIKeyString()
	if k1 == k2 {
		t.Error("two generated keys should not be identical")
	}
}
