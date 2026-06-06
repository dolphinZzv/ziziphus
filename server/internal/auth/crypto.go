package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type Crypto struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

func NewCrypto(privateKeyPath string) (*Crypto, error) {
	data, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM data in private key file")
	}
	priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// try PKCS1
		priv, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
	}
	rsaPriv, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return &Crypto{
		privateKey: rsaPriv,
		publicKey:  &rsaPriv.PublicKey,
	}, nil
}

func NewCryptoFromKeys(priv *rsa.PrivateKey) *Crypto {
	return &Crypto{
		privateKey: priv,
		publicKey:  &priv.PublicKey,
	}
}

func (c *Crypto) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, c.publicKey, plaintext, nil)
}

func (c *Crypto) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, c.privateKey, ciphertext, nil)
}

func GenerateKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

const testPrivateKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0ZkVfJ3w7mE5a3sYg0n0HnDxHKn1FxN0XoRmGmMz0mO0
pRKfG8xvGmEoZG0K0aO0pRKfG8xvGmEoZG0K0aO0pRKfG8xvGmEoZG0K0aO0
-----END RSA PRIVATE KEY-----`

func GenerateAndSaveKeys(path string) error {
	priv, err := GenerateKeyPair()
	if err != nil {
		return err
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
}
