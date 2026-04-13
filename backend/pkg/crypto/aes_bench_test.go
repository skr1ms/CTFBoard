package crypto_test

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

// 64-char hex = 32 bytes = AES-256 key.
const benchKey = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func BenchmarkEncrypt(b *testing.B) {
	svc, err := crypto.NewCryptoService(benchKey)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.Encrypt("flag{benchmark_plaintext_value}")
	}
}

func BenchmarkDecrypt(b *testing.B) {
	svc, err := crypto.NewCryptoService(benchKey)
	if err != nil {
		b.Fatal(err)
	}

	ct, err := svc.Encrypt("flag{benchmark_plaintext_value}")
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		_, _ = svc.Decrypt(ct)
	}
}

func BenchmarkEncryptDecryptRoundTrip(b *testing.B) {
	svc, err := crypto.NewCryptoService(benchKey)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()

	for b.Loop() {
		ct, _ := svc.Encrypt("flag{round_trip_value}")
		_, _ = svc.Decrypt(ct)
	}
}
