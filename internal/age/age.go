package age

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"io"
	"os"
	"sync"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"
)

type Decryptor struct {
	mu         sync.RWMutex
	identities []age.Identity
}

func NewDecryptor() *Decryptor {
	return &Decryptor{}
}

func (d *Decryptor) LoadIdentities(paths []string) (err error) {
	var identities []age.Identity
	for _, path := range paths {
		var fileIdentities []age.Identity
		if fileIdentities, err = parseIdentityFile(path); err != nil {
			err = fmt.Errorf("failed to load age identity from '%s': %w", path, err)
			return
		}
		identities = append(identities, fileIdentities...)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.identities = identities
	return
}

func (d *Decryptor) Decrypt(r io.Reader) (plaintext []byte, err error) {
	d.mu.RLock()
	identities := d.identities
	d.mu.RUnlock()

	if len(identities) == 0 {
		err = fmt.Errorf("no age identities configured")
		return
	}

	var reader io.Reader
	if reader, err = age.Decrypt(r, identities...); err != nil {
		err = fmt.Errorf("age decryption failed: %w", err)
		return
	}

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, reader); err != nil {
		err = fmt.Errorf("failed to read decrypted content: %w", err)
		return
	}

	plaintext = buf.Bytes()
	return
}

func parseIdentityFile(path string) (identities []age.Identity, err error) {
	var data []byte
	if data, err = os.ReadFile(path); err != nil {
		return
	}

	// Try parsing as SSH private key first
	if id, sshErr := parseSSHIdentity(data); sshErr == nil {
		identities = append(identities, id)
		return
	}

	// Fall back to native age identity format
	var parsed []age.Identity
	if parsed, err = age.ParseIdentities(bytes.NewReader(data)); err != nil {
		err = fmt.Errorf("failed to parse as age or SSH identity: %w", err)
		return
	}

	identities = append(identities, parsed...)
	return
}

func parseSSHIdentity(data []byte) (id age.Identity, err error) {
	var key interface{}
	if key, err = ssh.ParseRawPrivateKey(data); err != nil {
		return
	}

	switch k := key.(type) {
	case *ed25519.PrivateKey:
		id, err = agessh.NewEd25519Identity(*k)
	case ed25519.PrivateKey:
		id, err = agessh.NewEd25519Identity(k)
	default:
		err = fmt.Errorf("unsupported SSH key type %T, only ed25519 is supported", key)
	}
	return
}
