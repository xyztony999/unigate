package api

// Port of src/test/java/com/unigate/PasswordResetTest.java (HTTP level).

import (
	"context"
	"net/http"
	"testing"

	"unigate/backend/internal/errcode"
)

// TestIntegrationPasswordResetIssuesTemporaryPasswordAndForcesChange ports
// resetIssuesTemporaryPasswordAndForcesChange: the service issues a >=12 char
// temporary password, the temporary password logs in with mustChangePassword,
// the old password is rejected 401, and the forced change succeeds.
func TestIntegrationPasswordResetIssuesTemporaryPasswordAndForcesChange(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)
	ctx := context.Background()

	temporary, err := e.svc.ResetToTemporaryPassword(ctx, itAdminUsername)
	if err != nil {
		t.Fatalf("reset to temporary password: %v", err)
	}
	if len(temporary) < 12 {
		t.Fatalf("临时密码长度不足: %d", len(temporary))
	}

	reset := e.browser(t)
	reset.bootstrap()
	login := reset.write("Login", map[string]any{"username": itAdminUsername, "password": temporary}, http.StatusOK)
	if !itBool(t, login, "mustChangePassword") {
		t.Fatalf("temporary-password login must force a change: %+v", login)
	}

	old := e.browser(t)
	old.bootstrap()
	old.write("Login", map[string]any{"username": itAdminUsername, "password": itAdminPassword}, http.StatusUnauthorized)
	// Frozen wire mapping (Java ApiException): internal AUTHENTICATION_FAILED
	// is published as UNAUTHENTICATED.
	if old.lastCode != "UNAUTHENTICATED" {
		t.Fatalf("old password login code = %q, want UNAUTHENTICATED", old.lastCode)
	}

	reset.request("SessionMe", map[string]any{}, http.StatusOK)
	reset.write("PasswordChange", map[string]any{"oldPassword": temporary, "newPassword": "Changed-reset-123!"}, http.StatusOK)
}

// TestIntegrationPasswordResetRejectsUnknownAccount ports
// resetRejectsUnknownAccount: resetting a missing account is a 404 ApiException.
func TestIntegrationPasswordResetRejectsUnknownAccount(t *testing.T) {
	e := integrationEnv(t)
	e.reset(t)

	_, err := e.svc.ResetToTemporaryPassword(context.Background(), "no-such-user")
	if err == nil {
		t.Fatal("reset of unknown account must fail")
	}
	ae := errcode.AsError(err)
	if ae == nil {
		t.Fatalf("want ApiException, got %v", err)
	}
	if ae.Status != http.StatusNotFound || ae.Code != "RESOURCE_NOT_FOUND" {
		t.Fatalf("unknown account reset = %d %s, want 404 RESOURCE_NOT_FOUND", ae.Status, ae.Code)
	}
}
