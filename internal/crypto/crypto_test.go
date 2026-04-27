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
	"bytes"
	"testing"
)

var testKey = DeriveKey([]byte("test-secret-for-unit-tests"))

func TestDeriveKey_Length(t *testing.T) {
	key := DeriveKey([]byte("any secret"))
	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	a := DeriveKey([]byte("same secret"))
	b := DeriveKey([]byte("same secret"))
	if !bytes.Equal(a, b) {
		t.Error("DeriveKey is not deterministic for the same input")
	}
}

func TestDeriveKey_DifferentInputs(t *testing.T) {
	a := DeriveKey([]byte("secret-one"))
	b := DeriveKey([]byte("secret-two"))
	if bytes.Equal(a, b) {
		t.Error("different secrets produced the same key")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	plaintext := []byte("eyJrZXkiOiAidmFsdWUifQ==") // base64 config data

	encrypted, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_ProducesUniqueNonce(t *testing.T) {
	plaintext := []byte("same plaintext")

	a, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}

	if a == b {
		t.Error("two encryptions of the same plaintext produced identical ciphertext — nonce is not random")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	plaintext := []byte("secret config")

	encrypted, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}

	wrongKey := DeriveKey([]byte("completely-different-secret"))
	_, err = Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Error("expected error decrypting with wrong key, got nil")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	plaintext := []byte("sensitive config data")

	encrypted, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatal(err)
	}

	/* Flip a byte in the middle of the base64 string */
	tampered := []byte(encrypted)
	tampered[len(tampered)/2] ^= 0xFF
	_, err = Decrypt(string(tampered), testKey)
	if err == nil {
		t.Error("expected error decrypting tampered ciphertext, got nil")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	_, err := Decrypt("!!!not-valid-base64!!!", testKey)
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	import64 := "YQ==" // base64("a") — too short to contain a nonce
	_, err := Decrypt(import64, testKey)
	if err == nil {
		t.Error("expected error for ciphertext shorter than nonce, got nil")
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	plaintext := []byte("")

	encrypted, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatalf("Encrypt() error on empty plaintext: %v", err)
	}

	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("Decrypt() error on empty plaintext: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}
