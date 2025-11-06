package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/tink/go/kwp/subtle"
	"go.uber.org/zap"
)

var (
	operatorPriv = flag.String("operator-priv", "", "Operator private key (hex string)")
	wrappingPEM  = flag.String("wrapping-pem", "", "Wrapping PEM (string)")
)

func main() {

	flag.Parse()

	operatorPublicKey, ciphertext, err := wrapOperatorPrivateKey()
	if err != nil {
		log.Fatal("failed to wrap operator private key", zap.Error(err))
	}

	fmt.Println("\nOperator Public Key:", operatorPublicKey.String())

	fmt.Println("\nCiphertext:", ciphertext)
}

func wrapOperatorPrivateKey() (common.Address, string, error) {

	if *operatorPriv == "" {
		return common.Address{}, "", errors.New("operator private key is required")
	}

	priv, err := crypto.HexToECDSA(*operatorPriv)
	if err != nil {
		return common.Address{}, "", err
	}

	evmPublicKey := crypto.PubkeyToAddress(priv.PublicKey)

	pkcs8Bytes, err := marshalSecp256k1PrivateKeyToPKCS8(priv)
	if err != nil {
		return common.Address{}, "", err
	}

	if *wrappingPEM == "" {
		return common.Address{}, "", errors.New("wrapping PEM is required")
	}

	fixedWrappingPEM := strings.ReplaceAll(*wrappingPEM, `\n`, "\n")

	block, _ := pem.Decode([]byte(fixedWrappingPEM))
	if block == nil {
		return common.Address{}, "", errors.New("failed to decode PEM block")
	}

	pubKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return common.Address{}, "", err
	}

	rsaPub, ok := pubKeyInterface.(*rsa.PublicKey)
	if !ok {
		return common.Address{}, "", errors.New("not an RSA public key")
	}

	// Generate AES key
	aesKey := make([]byte, 32)
	_, err = rand.Read(aesKey)
	if err != nil {
		return common.Address{}, "", err
	}

	// KWP encryption of the private key
	wrapper, err := subtle.NewKWP(aesKey)
	if err != nil {
		return common.Address{}, "", err
	}

	wrappedKey, err := wrapper.Wrap(pkcs8Bytes)
	if err != nil {
		return common.Address{}, "", err
	}

	// Encrypt AES key with RSA
	encryptedAESKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, aesKey, nil)
	if err != nil {
		return common.Address{}, "", err
	}

	final := append(encryptedAESKey, wrappedKey...)
	ciphertext := base64.StdEncoding.EncodeToString(final)

	return evmPublicKey, ciphertext, nil
}

func marshalSecp256k1PrivateKeyToPKCS8(key *ecdsa.PrivateKey) ([]byte, error) {
	secp256k1OID := asn1.ObjectIdentifier{1, 3, 132, 0, 10}

	privateKeyBytes := key.D.Bytes()
	if len(privateKeyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(privateKeyBytes):], privateKeyBytes)
		privateKeyBytes = padded
	}

	sec1Key := struct {
		Version       int
		PrivateKey    []byte
		NamedCurveOID asn1.ObjectIdentifier `asn1:"optional,explicit,tag:0"`
		PublicKey     asn1.BitString        `asn1:"optional,explicit,tag:1"`
	}{
		Version:       1,
		PrivateKey:    privateKeyBytes,
		NamedCurveOID: secp256k1OID,
	}

	sec1Der, err := asn1.Marshal(sec1Key)
	if err != nil {
		return nil, err
	}

	pkcs8 := struct {
		Version int
		Algo    struct {
			Algorithm  asn1.ObjectIdentifier
			Parameters asn1.ObjectIdentifier
		}
		PrivateKey []byte
	}{
		Version: 0,
		Algo: struct {
			Algorithm  asn1.ObjectIdentifier
			Parameters asn1.ObjectIdentifier
		}{
			Algorithm:  asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}, // ecPublicKey
			Parameters: secp256k1OID,
		},
		PrivateKey: sec1Der,
	}

	return asn1.Marshal(pkcs8)
}
