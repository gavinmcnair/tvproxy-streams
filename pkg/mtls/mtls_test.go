package mtls

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

func TestLoadOrCreateCA(t *testing.T) {
	dir := t.TempDir()
	cert, key, err := LoadOrCreateCA(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !cert.IsCA {
		t.Fatal("expected CA cert")
	}
	if key == nil {
		t.Fatal("expected key")
	}

	cert2, key2, err := LoadOrCreateCA(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cert2.SerialNumber.Cmp(cert.SerialNumber) != 0 {
		t.Fatal("expected same CA on reload")
	}
	_ = key2
}

func TestIssueClientCert(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey, _ := LoadOrCreateCA(dir)

	certPEM, keyPEM, caPEM, err := IssueClientCert(caCert, caKey, "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 || len(caPEM) == 0 {
		t.Fatal("expected non-empty PEM output")
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("bad cert PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if cert.Subject.CommonName != "test@example.com" {
		t.Errorf("CN = %q, want test@example.com", cert.Subject.CommonName)
	}
}

func TestStoreEnrollmentFlow(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	token := store.GenerateToken("alice@example.com", 10*time.Minute)
	if token == "" {
		t.Fatal("empty token")
	}

	email, ok := store.ConsumeToken(token)
	if !ok || email != "alice@example.com" {
		t.Fatalf("consume failed: ok=%v email=%q", ok, email)
	}

	_, ok = store.ConsumeToken(token)
	if ok {
		t.Fatal("token should be consumed")
	}
}

func TestStoreExpiredToken(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	token := store.GenerateToken("bob@example.com", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := store.ConsumeToken(token)
	if ok {
		t.Fatal("expired token should not be consumable")
	}
}

func TestStoreClientCRUD(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	store.AddClient("alice@example.com", "sha256:aaa", "CERT_PEM")
	store.AddClient("bob@example.com", "sha256:bbb", "CERT_PEM2")

	clients := store.Clients()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}

	n := store.Revoke("alice@example.com")
	if n != 1 {
		t.Fatalf("expected 1 revoked, got %d", n)
	}
	if len(store.Clients()) != 1 {
		t.Fatal("expected 1 client after revoke")
	}

	store2 := NewStore(dir)
	if len(store2.Clients()) != 1 {
		t.Fatal("expected persistence after reload")
	}
}

func TestStoreIsAuthorized(t *testing.T) {
	dir := t.TempDir()
	caCert, caKey, _ := LoadOrCreateCA(dir)
	store := NewStore(dir)

	certPEM, _, _, _ := IssueClientCert(caCert, caKey, "auth@example.com")
	fp := PEMFingerprint(certPEM)
	store.AddClient("auth@example.com", fp, string(certPEM))

	block, _ := pem.Decode(certPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	if !store.IsAuthorized(cert) {
		t.Fatal("expected authorized")
	}

	store.Revoke("auth@example.com")
	if store.IsAuthorized(cert) {
		t.Fatal("expected unauthorized after revoke")
	}
}
