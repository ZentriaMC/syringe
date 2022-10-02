package secret

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	socktmpl "github.com/hashicorp/go-sockaddr/template"
	vaultapi "github.com/hashicorp/vault/api"

	cctx "github.com/ZentriaMC/syringe/internal/ctx"
)

type templateFn struct {
	ctx          context.Context
	cacheLock    sync.RWMutex
	cachedValues map[string]any
	sandboxPath  string
}

func (t *templateFn) inSandbox(path string) (err error) {
	if t.sandboxPath == "" {
		return
	}

	if path, err = filepath.EvalSymlinks(path); err != nil {
		return
	}

	if path, err = filepath.Rel(t.sandboxPath, path); err != nil {
		return
	}

	if strings.HasPrefix(path, "..") {
		err = fmt.Errorf("'%s' is not inside the sandbox", path)
		return
	}

	return
}

func (t *templateFn) cache(key string, compute func(key string) (any, error)) (v any, err error) {
	t.cacheLock.Lock()
	defer t.cacheLock.Unlock()

	var ok bool
	if v, ok = t.cachedValues[key]; ok {
		return
	}

	if v, err = compute(key); err != nil {
		return
	}

	t.cachedValues[key] = v
	return
}

func (t *templateFn) unitName() (name string, err error) {
	name, _ = cctx.CredentialRequest(t.ctx)
	return
}

func (t *templateFn) credentialName() (name string, err error) {
	_, name = cctx.CredentialRequest(t.ctx)
	return
}

func b64encode(value string) (encoded string, err error) {
	encoded = base64.StdEncoding.EncodeToString([]byte(value))
	return
}

func b64decode(value string) (decoded string, err error) {
	var decodedBytes []byte
	if decodedBytes, err = base64.StdEncoding.DecodeString(value); err != nil {
		err = fmt.Errorf("unable to decode base64: %w", err)
		return
	}

	decoded = string(decodedBytes)
	return
}

func sha256sum(item string) (string, error) {
	sum := sha256.Sum256([]byte(item))
	return hex.EncodeToString(sum[:]), nil
}

func sha1sum(item string) (string, error) {
	sum := sha1.Sum([]byte(item))
	return hex.EncodeToString(sum[:]), nil
}

func md5sum(item string) (string, error) {
	sum := md5.Sum([]byte(item))
	return hex.EncodeToString(sum[:]), nil
}

func timef(args ...string) (v string, err error) {
	t := time.Now()
	if len(args) == 0 {
		v = fmt.Sprintf("%d", t.Unix())
		return
	}

	format := args[0]
	if len(args) > 1 {
		// TODO: for loop over remaining args
		modifier := args[1]
		switch strings.ToLower(modifier) {
		case "utc":
			t = t.UTC()
		default:
			err = fmt.Errorf("unknown modifier '%s'", modifier)
			return
		}
	}

	switch strings.ToLower(format) {
	case "unix":
		v = fmt.Sprintf("%d", t.Unix())
		return
	case "unixmilli":
		v = fmt.Sprintf("%d", t.UnixMilli())
		return
	case "unixnano":
		v = fmt.Sprintf("%d", t.UnixNano())
		return
	case "rfc3339":
		v = t.Format(time.RFC3339)
		return
	default:
		v = t.Format(format)
		return
	}
}

func (t *templateFn) readFile(path string) (contents string, err error) {
	if err = t.inSandbox(path); err != nil {
		return
	}

	var ccontents any
	key := fmt.Sprintf("file:%s", path)
	ccontents, err = t.cache(key, func(key string) (v any, err error) {
		var readBytes []byte
		if readBytes, err = ioutil.ReadFile(path); err != nil {
			err = fmt.Errorf("unable to read '%s': %w", path, err)
			return
		}

		v = string(readBytes)
		return
	})
	if err != nil {
		return
	}

	contents = ccontents.(string)
	return
}

func (t *templateFn) readVault(path string, args ...string) (v any, err error) {
	var vault *vaultapi.Client
	if vault, err = cctx.VaultClient(t.ctx).Clone(); err != nil {
		return
	}

	wrapTTL := 0
	noCache := false
	readFn := func(key string) (v any, err error) {
		vault.SetWrappingLookupFunc(func(operation, opath string) string {
			if operation != http.MethodGet || opath != path {
				return ""
			}

			if wrapTTL == 0 {
				return ""
			}

			return fmt.Sprintf("%ds", wrapTTL)
		})

		var secret *vaultapi.Secret
		if secret, err = vault.Logical().ReadWithContext(t.ctx, path); err != nil {
			return
		}

		v = secret
		return
	}

	// TODO: use args

	var ccontents any
	key := fmt.Sprintf("vault:GET:%s", path)
	if noCache || wrapTTL > 0 {
		ccontents, err = readFn(key)
	} else {
		ccontents, err = t.cache(key, readFn)
	}

	if err != nil {
		return
	}

	v = ccontents
	return
}

func spewDump(args ...any) (v string, err error) {
	v = spew.Sdump(args...)
	return
}

func sockaddr(args ...string) (v string, err error) {
	t := fmt.Sprintf("{{ %s }}", strings.Join(args, " "))
	v, err = socktmpl.Parse(t)
	return
}
