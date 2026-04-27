/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// DeriveKey returns a 32-byte AES-256 key derived from the given secret
// using SHA-256. Any length secret is accepted.
func DeriveKey(secret []byte) []byte {
	key := sha256.Sum256(secret)
	return key[:]
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// The returned string is base64(nonce || ciphertext || tag), safe for
// storage in a text/mediumtext column.
func Encrypt(plaintext []byte, key []byte) (string, error) {

	block, err := aes.NewCipher(key)

	if err != nil {
		return "", fmt.Errorf("aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	/* Seal appends ciphertext+tag to nonce, giving us nonce||ciphertext||tag */

	sealed := gcm.Seal(nonce, nonce, plaintext, nil)

	return base64.StdEncoding.EncodeToString(sealed), nil

}

// Decrypt decodes and decrypts a value produced by Encrypt.
// Returns an error if the key is wrong or the ciphertext has been tampered with.
func Decrypt(encoded string, key []byte) ([]byte, error) {

	data, err := base64.StdEncoding.DecodeString(encoded)

	if err != nil {
		return nil, fmt.Errorf("base64: %w", err)
	}

	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()

	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)

	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil

}
