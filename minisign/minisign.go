// https://github.com/jedisct1/go-minisign
//
// Copyright (c) 2018-2019 Frank Denis
//
// This code is an adapted version of go-minisign.
//
// Use of this source code is governed by the license that can be found at:
//   https://github.com/jedisct1/go-minisign/blob/master/LICENSE

package minisign

import (
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/ed25519"
)

// Signature lines count
const sigLines = 4

type PublicKey struct {
	SignatureAlgorithm [2]byte
	KeyId              [8]byte
	PublicKey          [32]byte
}

type Signature struct {
	UntrustedComment   string
	SignatureAlgorithm [2]byte
	KeyId              [8]byte
	Signature          [64]byte
	TrustedComment     string
	GlobalSignature    [64]byte
}

func NewPublicKey(publicKeyStr string) (PublicKey, error) {
	var publicKey PublicKey

	bin, err := base64.StdEncoding.DecodeString(publicKeyStr)

	if err != nil || len(bin) != 42 {
		return publicKey, errors.New("Invalid encoded public key")
	}

	copy(publicKey.SignatureAlgorithm[:], bin[0:2])
	copy(publicKey.KeyId[:], bin[2:10])
	copy(publicKey.PublicKey[:], bin[10:42])

	return publicKey, nil
}

func DecodeSignature(in string) (Signature, int, error) {
	var signature Signature

	lines := strings.SplitN(in, "\n", sigLines+1)

	if len(lines) < 2 {
		return signature, 0, errors.New("Incomplete encoded signature")
	}

	signature.UntrustedComment = lines[0]

	bin1, err := base64.StdEncoding.DecodeString(lines[1])

	if err != nil || len(bin1) != 74 {
		return signature, 0, errors.New("Invalid encoded signature")
	}

	copy(signature.SignatureAlgorithm[:], bin1[0:2])
	copy(signature.KeyId[:], bin1[2:10])
	copy(signature.Signature[:], bin1[10:74])

	signature.TrustedComment = lines[2]

	bin2, err := base64.StdEncoding.DecodeString(lines[3])

	if err != nil || len(bin2) != 64 {
		return signature, 0, errors.New("Invalid encoded signature")
	}

	copy(signature.GlobalSignature[:], bin2)

	n := sigLines

	for i := 0; i < sigLines; i++ {
		n += len(lines[i])
	}

	return signature, n, nil
}

func (publicKey *PublicKey) Verify(bin []byte, signature Signature) (bool, error) {
	if publicKey.SignatureAlgorithm != signature.SignatureAlgorithm {
		return false, errors.New("Incompatible signature algorithm")
	}

	if signature.SignatureAlgorithm[0] != 0x45 || signature.SignatureAlgorithm[1] != 0x64 {
		return false, errors.New("Unsupported signature algorithm")
	}

	if publicKey.KeyId != signature.KeyId {
		return false, errors.New("Incompatible key identifiers")
	}

	if !ed25519.Verify(ed25519.PublicKey(publicKey.PublicKey[:]), bin, signature.Signature[:]) {
		return false, errors.New("Invalid signature")
	}

	if len(signature.TrustedComment) != 0 {
		if !strings.HasPrefix(signature.TrustedComment, "trusted comment: ") {
			return false, errors.New("Unexpected format for the trusted comment")
		}

		if !ed25519.Verify(ed25519.PublicKey(publicKey.PublicKey[:]), append(signature.Signature[:], []byte(signature.TrustedComment)[17:]...), signature.GlobalSignature[:]) {
			return false, errors.New("Invalid global signature")
		}
	}

	return true, nil
}
