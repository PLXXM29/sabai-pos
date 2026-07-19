package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueAndParseAccessToken(t *testing.T) {
	secret := "test-secret"
	uid, sid := uuid.New(), uuid.New()
	tok, err := IssueAccessToken(secret, uid, sid, "manager", time.Minute, time.Now())
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := ParseAccessToken(secret, tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.Subject != uid.String() {
		t.Errorf("subject: want %s got %s", uid, claims.Subject)
	}
	if claims.StoreID != sid.String() {
		t.Errorf("store_id: want %s got %s", sid, claims.StoreID)
	}
	if claims.Role != "manager" {
		t.Errorf("role: want manager got %s", claims.Role)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	secret := "test-secret"
	// issued in the past so it's already expired
	tok, _ := IssueAccessToken(secret, uuid.New(), uuid.New(), "cashier", time.Minute, time.Now().Add(-2*time.Minute))
	if _, err := ParseAccessToken(secret, tok); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	tok, _ := IssueAccessToken("secret-a", uuid.New(), uuid.New(), "cashier", time.Minute, time.Now())
	if _, err := ParseAccessToken("secret-b", tok); err == nil {
		t.Fatal("expected wrong-secret token to be rejected")
	}
}
