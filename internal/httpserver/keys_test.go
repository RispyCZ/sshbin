package httpserver

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/rispycz/sshbin/internal/sharing"
	"github.com/rispycz/sshbin/internal/sshkeys"
)

func genAuthorizedKey(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	return sshkeys.Canonical(sshPub) + " test@host"
}

func addKeyReq(t *testing.T, h *handler, cookie *http.Cookie, title, key string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"title": title, "key": key})
	req := httptest.NewRequest("POST", "/api/keys", strings.NewReader(string(body)))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.requireSessionAPI(h.apiKeysAdd)(rec, req)
	return rec
}

func TestAPIKeys_AddListDelete(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	cookie := login(t, h, sender, "owner@example.com")
	key := genAuthorizedKey(t)

	rec := addKeyReq(t, h, cookie, "laptop", key)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add status = %d, want 201 (%s)", rec.Code, rec.Body)
	}
	var added keyDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &added); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if added.Title != "laptop" || !strings.HasPrefix(added.Fingerprint, "SHA256:") {
		t.Errorf("unexpected key DTO: %+v", added)
	}

	// List returns it.
	listReq := httptest.NewRequest("GET", "/api/keys", nil)
	listReq.AddCookie(cookie)
	listRec := httptest.NewRecorder()
	h.requireSessionAPI(h.apiKeysList)(listRec, listReq)
	var list []keyDTO
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].ID != added.ID {
		t.Fatalf("list = %+v, want the added key", list)
	}

	// Delete it.
	delReq := httptest.NewRequest("DELETE", "/api/keys/"+added.ID, nil)
	delReq.SetPathValue("id", added.ID)
	delReq.AddCookie(cookie)
	delRec := httptest.NewRecorder()
	h.requireSessionAPI(h.apiKeysDelete)(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", delRec.Code)
	}
}

func TestAPIKeys_AddInvalid(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	cookie := login(t, h, sender, "owner@example.com")

	rec := addKeyReq(t, h, cookie, "bad", "this is not a key")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAPIKeys_AddDuplicateConflict(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	cookie := login(t, h, sender, "owner@example.com")
	key := genAuthorizedKey(t)

	if rec := addKeyReq(t, h, cookie, "first", key); rec.Code != http.StatusCreated {
		t.Fatalf("first add status = %d, want 201", rec.Code)
	}
	if rec := addKeyReq(t, h, cookie, "again", key); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate add status = %d, want 409", rec.Code)
	}
}

func TestAPIKeys_DeleteNotOwned(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	ownerCookie := login(t, h, sender, "owner@example.com")
	key := genAuthorizedKey(t)
	rec := addKeyReq(t, h, ownerCookie, "laptop", key)
	var added keyDTO
	json.Unmarshal(rec.Body.Bytes(), &added)

	// A different user cannot delete it.
	intruder := login(t, h, sender, "intruder@example.com")
	delReq := httptest.NewRequest("DELETE", "/api/keys/"+added.ID, nil)
	delReq.SetPathValue("id", added.ID)
	delReq.AddCookie(intruder)
	delRec := httptest.NewRecorder()
	h.requireSessionAPI(h.apiKeysDelete)(delRec, delReq)
	if delRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", delRec.Code)
	}
}

func TestAPIKeys_DeleteAllClearsKeys(t *testing.T) {
	h, sender := newTestHandler(t, sharing.NewMemoryRepository())
	cookie := login(t, h, sender, "owner@example.com")
	if rec := addKeyReq(t, h, cookie, "laptop", genAuthorizedKey(t)); rec.Code != http.StatusCreated {
		t.Fatalf("add status = %d", rec.Code)
	}

	req := httptest.NewRequest("DELETE", "/api/profile", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.requireSessionAPI(h.apiProfileDeleteAll)(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete-all status = %d, want 204", rec.Code)
	}

	remaining, err := h.keys.ListByEmail(context.Background(), "owner@example.com")
	if err != nil {
		t.Fatalf("ListByEmail: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("keys remaining after delete-all: %d, want 0", len(remaining))
	}
}
