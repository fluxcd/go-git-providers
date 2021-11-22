/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testutils

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

// KeyPair holds the public and private key PEM block bytes.
type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

// KeyPairGenerator generates a new key pair.
type KeyPairGenerator interface {
	Generate() (*KeyPair, error)
}

// RSAGenerator generates RSA key pairs.
type RSAGenerator struct {
	bits int
}

// NewRSAGenerator returns a new RSA key pair generator.
func NewRSAGenerator(bits int) KeyPairGenerator {
	return &RSAGenerator{bits}
}

// Generate generates a new key pair.
func (g *RSAGenerator) Generate() (*KeyPair, error) {
	pk, err := rsa.GenerateKey(rand.Reader, g.bits)
	if err != nil {
		return nil, err
	}
	err = pk.Validate()
	if err != nil {
		return nil, err
	}
	pub, err := generatePublicKey(&pk.PublicKey)
	if err != nil {
		return nil, err
	}
	priv, err := encodePrivateKeyToPEM(pk)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// ECDSAGenerator generates ECDSA key pairs.
type ECDSAGenerator struct {
	c elliptic.Curve
}

// NewECDSAGenerator returns a new ECDSA key pair generator.
func NewECDSAGenerator(c elliptic.Curve) KeyPairGenerator {
	return &ECDSAGenerator{c}
}

// Generate generates a new key pair.
func (g *ECDSAGenerator) Generate() (*KeyPair, error) {
	pk, err := ecdsa.GenerateKey(g.c, rand.Reader)
	if err != nil {
		return nil, err
	}
	pub, err := generatePublicKey(&pk.PublicKey)
	if err != nil {
		return nil, err
	}
	priv, err := encodePrivateKeyToPEM(pk)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// Ed25519Generator generates Ed25519 key pairs.
type Ed25519Generator struct{}

// NewEd25519Generator returns a new Ed25519 key pair generator.
func NewEd25519Generator() KeyPairGenerator {
	return &Ed25519Generator{}
}

// Generate generates a new key pair.
func (g *Ed25519Generator) Generate() (*KeyPair, error) {
	pk, pv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	pub, err := generatePublicKey(pk)
	if err != nil {
		return nil, err
	}
	priv, err := encodePrivateKeyToPEM(pv)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

func generatePublicKey(pk interface{}) ([]byte, error) {
	b, err := ssh.NewPublicKey(pk)
	if err != nil {
		return nil, err
	}
	k := ssh.MarshalAuthorizedKey(b)
	return k, nil
}

// encodePrivateKeyToPEM encodes the given private key to a PEM block.
// The encoded format is PKCS#8 for universal support of the most
// common key types (rsa, ecdsa, ed25519).
func encodePrivateKeyToPEM(pk interface{}) ([]byte, error) {
	b, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		return nil, err
	}
	block := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: b,
	}
	return pem.EncodeToMemory(&block), nil
}
