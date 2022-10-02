package request_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/ZentriaMC/syringe/internal/request"
)

func TestRequestParsing(t *testing.T) {
	allTestData := [][]string{
		{
			"@$RANDOM/unit/waldo.service/foo",
			"$RANDOM",
			"waldo.service",
			"foo",
		},
		{
			"@adf9d86b6eda275e/unit/foobar.service/credx",
			"adf9d86b6eda275e",
			"foobar.service",
			"credx",
		},
	}

	for _, testData := range allTestData {
		requestPayload := []byte(testData[0])
		randomValue := []byte(testData[1])
		unit := testData[2]
		credential := testData[3]

		req, err := request.ParseCredentialRequest(requestPayload)
		if err != nil {
			t.Errorf("failed to parse '%s': %s", requestPayload, err)
			continue
		}

		if !bytes.Equal(req.Random, randomValue) {
			t.Errorf("Random: '%v' != '%v'", req.Random, randomValue)
			continue
		}

		if req.Unit != unit {
			t.Errorf("Unit: '%v' != '%v'", req.Unit, unit)
			continue
		}

		if req.Credential != credential {
			t.Errorf("Credential: '%v' != '%v'", req.Credential, credential)
			continue
		}

		spew.Dump(req)
	}
}

func TestRequestSerializing(t *testing.T) {
	var randU64 uint64 = 1765
	randBytes := bytes.Buffer{}
	fmt.Fprintf(&randBytes, "%d", randU64)

	must := func(cr request.CredentialRequest, err error) request.CredentialRequest {
		if err != nil {
			panic(err)
		}
		return cr
	}

	requests := []request.CredentialRequest{
		must(request.NewCredentialRequestWithRand("waldo.service", "foo", randBytes.Bytes())),
		must(request.NewCredentialRequestWithRand("foobar.service", "credx", randBytes.Bytes())),
	}

	for idx, credRequest := range requests {
		serialized, err := credRequest.Bytes()
		if err != nil {
			t.Errorf("failed to serialize request %d: %s", idx, err)
			continue
		}

		parsed, err := request.ParseCredentialRequest(serialized)
		if err != nil {
			t.Errorf("failed to parse request %d: %s", idx, err)
			continue
		}

		if credRequest.Unit != parsed.Unit {
			t.Errorf("Unit: '%v' != '%v'", credRequest.Unit, parsed.Unit)
			continue
		}

		if credRequest.Credential != parsed.Credential {
			t.Errorf("Credential: '%v' != '%v'", credRequest.Credential, parsed.Credential)
			continue
		}

		randStr := string(credRequest.Random)
		v, err := strconv.ParseUint(randStr, 10, 64)
		if err != nil {
			t.Errorf("failed to parse random number '%s': %s", randStr, err)
		}

		if randU64 != v {
			t.Errorf("Random: '%v' != '%v'", randU64, v)
		}
	}
}
