package hpke

import (
	"crypto/rand"
	"testing"

	"github.com/cloudflare/circl/internal/test"
)

func TestAeadExporter(t *testing.T) {
	suite := Suite{KdfID: HkdfSha256, AeadID: AeadAes128Gcm}
	exporter := &encdecCtx{Suite: suite}
	maxLength := uint(255 * suite.KdfID.Hash().Size())

	err := test.CheckPanic(func() {
		exporter.Export([]byte("exporter"), maxLength+1)
	})
	test.CheckNoErr(t, err, "exporter max size")
}

func TestAeadSeqOverflows(t *testing.T) {
	suite := Suite{AeadID: AeadAes128Gcm}

	key := make([]byte, suite.AeadID.KeySize())
	_, _ = rand.Read(key)
	aead, err := suite.AeadID.New(key)
	test.CheckNoErr(t, err, "bad key")

	Nn := aead.NonceSize()
	nonce := make([]byte, Nn)
	_, _ = rand.Read(nonce)
	sealer := &sealCtx{&encdecCtx{suite, aead, nonce, make([]byte, Nn), nil}}
	opener := &openCtx{&encdecCtx{suite, aead, nonce, make([]byte, Nn), nil}}

	pt := []byte("plaintext")
	aad := []byte("aad")

	// Sets sequence number to 256 before its max value = 0xFF...FF.
	for i := 0; i < Nn; i++ {
		sealer.seq[i] = 0xFF
		opener.seq[i] = 0xFF
	}
	sealer.seq[Nn-1] = 0x00
	opener.seq[Nn-1] = 0x00

	numAttempts := 260
	wantCorrect := 2 * 255
	wantIncorrect := 2*numAttempts - wantCorrect
	gotCorrect := 0
	gotIncorrect := 0

	for i := 0; i < numAttempts; i++ {
		ct, err := sealer.Seal(pt, aad)
		switch true {
		case ct != nil && err == nil:
			gotCorrect++
		case ct == nil && err != nil:
			gotIncorrect++
		default:
			t.FailNow()
		}

		pt2, err := opener.Open(ct, aad)
		switch true {
		case pt2 != nil && err == nil:
			gotCorrect++
		case pt2 == nil && err != nil:
			gotIncorrect++
		default:
			t.FailNow()
		}
	}

	if gotCorrect != wantCorrect {
		test.ReportError(t, gotCorrect, wantCorrect)
	}
	if gotIncorrect != wantIncorrect {
		test.ReportError(t, gotIncorrect, wantIncorrect)
	}
}