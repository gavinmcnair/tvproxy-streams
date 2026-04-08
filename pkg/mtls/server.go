package mtls

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	CACert     *x509.Certificate
	CAKey      *ecdsa.PrivateKey
	Store      *Store
	CertPath   string
	KeyPath    string
	ConfigDir  string
}

func (s *Server) TLSConfig() *tls.Config {
	caPool := x509.NewCertPool()
	caPool.AddCert(s.CACert)

	return &tls.Config{
		ClientCAs:  caPool,
		ClientAuth: tls.VerifyClientCertIfGiven,
	}
}

func (s *Server) RequireClientCert(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
			return
		}
		if !s.Store.IsAuthorized(r.TLS.PeerCertificates[0]) {
			http.Error(w, "client certificate not authorized", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) EnrollHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}

	email, ok := s.Store.ConsumeToken(req.Token)
	if !ok {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	certPEM, keyPEM, caPEM, err := IssueClientCert(s.CACert, s.CAKey, email)
	if err != nil {
		http.Error(w, "failed to issue certificate", http.StatusInternalServerError)
		return
	}

	fingerprint := PEMFingerprint(certPEM)
	s.Store.AddClient(email, fingerprint, string(certPEM))

	log.Printf("enrolled client: %s (%s)", email, fingerprint)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"cert":        string(certPEM),
		"key":         string(keyPEM),
		"ca":          string(caPEM),
		"email":       email,
		"fingerprint": fingerprint,
	})
}

func Setup(configDir string) (*Server, error) {
	caCert, caKey, err := LoadOrCreateCA(configDir)
	if err != nil {
		return nil, fmt.Errorf("CA setup: %w", err)
	}

	certPath, keyPath, err := LoadOrCreateServerCert(configDir, caCert, caKey)
	if err != nil {
		return nil, fmt.Errorf("server cert setup: %w", err)
	}

	store := NewStore(configDir)

	log.Printf("mTLS enabled: CA loaded, %d enrolled clients", len(store.Clients()))

	return &Server{
		CACert:    caCert,
		CAKey:     caKey,
		Store:     store,
		CertPath:  certPath,
		KeyPath:   keyPath,
		ConfigDir: configDir,
	}, nil
}
