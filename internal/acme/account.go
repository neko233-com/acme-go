package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/registration"
)

type User struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration,omitempty"`
	key          crypto.PrivateKey
	keyPath      string
	statePath    string
}

func LoadOrCreateUser(email string, keyPath string) (*User, error) {
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		return nil, fmt.Errorf("create account dir: %w", err)
	}

	key, err := loadOrCreateKey(keyPath)
	if err != nil {
		return nil, err
	}

	user := &User{
		Email:     email,
		key:       key,
		keyPath:   keyPath,
		statePath: filepath.Join(filepath.Dir(keyPath), "account.json"),
	}
	if err := user.loadState(); err != nil {
		return nil, err
	}
	if user.Email == "" {
		user.Email = email
	}
	return user, nil
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func (u *User) Save() error {
	if err := savePrivateKey(u.keyPath, u.key); err != nil {
		return err
	}

	data, err := json.MarshalIndent(struct {
		Email        string                 `json:"email"`
		Registration *registration.Resource `json:"registration,omitempty"`
	}{
		Email:        u.Email,
		Registration: u.Registration,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal account state: %w", err)
	}
	if err := os.WriteFile(u.statePath, data, 0o600); err != nil {
		return fmt.Errorf("write account state: %w", err)
	}
	return nil
}

func (u *User) loadState() error {
	data, err := os.ReadFile(u.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read account state: %w", err)
	}

	state := struct {
		Email        string                 `json:"email"`
		Registration *registration.Resource `json:"registration,omitempty"`
	}{}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse account state: %w", err)
	}
	if u.Email == "" {
		u.Email = state.Email
	}
	u.Registration = state.Registration
	return nil
}

func loadOrCreateKey(path string) (crypto.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		key, parseErr := parsePrivateKeyPEM(data)
		if parseErr != nil {
			return nil, parseErr
		}
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	if err := savePrivateKey(path, key); err != nil {
		return nil, err
	}
	return key, nil
}

func savePrivateKey(path string, key crypto.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	return nil
}

func parsePrivateKeyPEM(data []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM private key")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		switch typed := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return typed, nil
		}
	}
	return nil, fmt.Errorf("unsupported private key format")
}
