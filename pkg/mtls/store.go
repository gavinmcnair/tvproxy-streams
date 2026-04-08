package mtls

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Client struct {
	Email       string `json:"email"`
	Fingerprint string `json:"fingerprint"`
	CertPEM     string `json:"cert_pem"`
	EnrolledAt  string `json:"enrolled_at"`
}

type EnrollmentToken struct {
	Token     string
	Email     string
	ExpiresAt time.Time
}

type Store struct {
	mu      sync.RWMutex
	clients []Client
	tokens  []EnrollmentToken
	path    string
}

func NewStore(configDir string) *Store {
	s := &Store{path: filepath.Join(configDir, "clients.json")}
	s.load()
	return s
}

func (s *Store) Clients() []Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Client, len(s.clients))
	copy(out, s.clients)
	return out
}

func (s *Store) AddClient(email, fingerprint, certPEM string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.clients {
		if c.Email == email {
			s.clients[i].Fingerprint = fingerprint
			s.clients[i].CertPEM = certPEM
			s.clients[i].EnrolledAt = time.Now().Format(time.RFC3339)
			s.save()
			return
		}
	}
	s.clients = append(s.clients, Client{
		Email:       email,
		Fingerprint: fingerprint,
		CertPEM:     certPEM,
		EnrolledAt:  time.Now().Format(time.RFC3339),
	})
	s.save()
}

func (s *Store) Revoke(email string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	var remaining []Client
	removed := 0
	for _, c := range s.clients {
		if c.Email == email {
			removed++
		} else {
			remaining = append(remaining, c)
		}
	}
	if removed > 0 {
		s.clients = remaining
		s.save()
	}
	return removed
}

func (s *Store) IsAuthorized(cert *x509.Certificate) bool {
	fp := CertFingerprint(cert)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		if c.Fingerprint == fp {
			return true
		}
	}
	return false
}

func (s *Store) GenerateToken(email string, ttl time.Duration) string {
	b := make([]byte, 16)
	rand.Read(b)
	token := "TVP-ENROLL-" + hex.EncodeToString(b)

	s.mu.Lock()
	s.tokens = append(s.tokens, EnrollmentToken{
		Token:     token,
		Email:     email,
		ExpiresAt: time.Now().Add(ttl),
	})
	s.mu.Unlock()
	return token
}

func (s *Store) ConsumeToken(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i, t := range s.tokens {
		if t.Token == token && now.Before(t.ExpiresAt) {
			email := t.Email
			s.tokens = append(s.tokens[:i], s.tokens[i+1:]...)
			return email, true
		}
	}
	return "", false
}

func CertFingerprint(cert *x509.Certificate) string {
	h := sha256.Sum256(cert.Raw)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h[:]))
}

func PEMFingerprint(certPEM []byte) string {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return ""
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return ""
	}
	return CertFingerprint(cert)
}

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.clients)
}

func (s *Store) save() {
	data, _ := json.MarshalIndent(s.clients, "", "  ")
	tmp := s.path + ".tmp"
	os.WriteFile(tmp, data, 0600)
	os.Rename(tmp, s.path)
}
