package crypto

import (
	"strings"
	"testing"
)

const validTestKey = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func FuzzDecrypt(f *testing.F) {
	svc, err := NewCryptoService(validTestKey)
	if err != nil {
		f.Fatalf("setup: %v", err)
	}

	f.Add("")
	f.Add("AA==")
	f.Add("AAAAAAAAAAAAAAAA==")
	f.Add("not-base64!!!")
	f.Add(strings.Repeat("A", 256))

	if enc, encErr := svc.Encrypt("CTF{test_flag_1}"); encErr == nil {
		f.Add(enc)
	}

	f.Fuzz(func(_ *testing.T, ciphertext string) {
		_, _ = svc.Decrypt(ciphertext)
	})
}

func FuzzEncryptDecryptRoundtrip(f *testing.F) {
	svc, err := NewCryptoService(validTestKey)
	if err != nil {
		f.Fatalf("setup: %v", err)
	}

	f.Add("")
	f.Add("CTF{hello_world}")
	f.Add("flag{unicode_日本語_тест}")
	f.Add(strings.Repeat("x", 1024))

	f.Fuzz(func(t *testing.T, plaintext string) {
		enc, err := svc.Encrypt(plaintext)
		if err != nil {
			return
		}

		got, err := svc.Decrypt(enc)
		if err != nil {
			t.Errorf("Decrypt(Encrypt(%q)) returned error: %v", plaintext, err)

			return
		}

		if got != plaintext {
			t.Errorf("round-trip mismatch: got %q, want %q", got, plaintext)
		}
	})
}
