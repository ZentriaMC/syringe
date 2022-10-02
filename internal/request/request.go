package request

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type CredentialRequest struct {
	Random     []byte
	Unit       string
	Credential string
}

func (cr CredentialRequest) Bytes() (b []byte, err error) {
	buf := bytes.Buffer{}

	if _, err = fmt.Fprintf(&buf, "@%s/unit/%s/%s", cr.Random, cr.Unit, cr.Credential); err != nil {
		return
	}

	b = buf.Bytes()
	return
}

func ParseCredentialRequest(raw []byte) (cr CredentialRequest, err error) {
	// @$RANDOM/unit/waldo.service/foo
	if len(raw) == 0 || len(raw) > 108 || raw[0] != '@' {
		err = fmt.Errorf("invalid magic")
		return
	}

	parts := bytes.SplitN(raw[1:], []byte{'/'}, 4)
	if len(parts) != 4 {
		err = fmt.Errorf("unexpected parts: %d (expected 4)", len(parts))
		return
	}

	if string(parts[1]) != "unit" {
		err = fmt.Errorf("unable to process message, part idx 1 was '%s' instead of 'unit'", parts[1])
		return
	}

	cr.Random = raw[1 : len(raw)-(len(parts[1])+len(parts[2])+len(parts[3])+3)]
	cr.Unit = string(parts[2])
	cr.Credential = string(parts[3])

	return
}

func NewCredentialRequest(unit, credential string) (cr CredentialRequest, err error) {
	randBytesRaw := make([]byte, 8)
	if _, err = rand.Read(randBytesRaw); err != nil {
		return
	}

	randBytes := []byte(hex.EncodeToString(randBytesRaw))
	return NewCredentialRequestWithRand(unit, credential, randBytes)
}

func NewCredentialRequestWithRand(unit, credential string, randBytes []byte) (cr CredentialRequest, err error) {
	if l := len(randBytes); l == 0 {
		err = fmt.Errorf("len(randBytes) == 0")
		return
	}

	// TODO: maybe validate unit & credential values?
	cr.Unit = unit
	cr.Credential = credential
	cr.Random = randBytes

	return
}
