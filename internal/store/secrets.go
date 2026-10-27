package store

import (
	"encoding/json"
	"sync"
)

// Secrets is a credential store (the Keychain, Android Keystore, ...).
type Secrets interface {
	Set(name string, value []byte) error
	Get(name string) ([]byte, error)
}

// secretsName is the one entry holding every API key, as JSON.
const secretsName = "api-keys"

// secretVault keeps API keys out of settings.json once the platform's
// credential store has proven to work.
type secretVault struct {
	store Secrets
	mu    sync.Mutex
	// last is the blob most recently written, to skip identical writes.
	last string
}

// UseSecrets moves API keys into sec. It probes sec first and leaves
// keys in settings.json when the store doesn't work (a Linux desktop
// without a keyring), reporting whether it's in use. Call it before
// LoadSettings; it blocks.
func (s *Store) UseSecrets(sec Secrets) bool {
	if _, err := sec.Get(secretsName); err != nil {
		if err := sec.Set(secretsName+".probe", []byte("1")); err != nil {
			return false
		}
		if b, err := sec.Get(secretsName + ".probe"); err != nil || string(b) != "1" {
			return false
		}
	}
	s.vault = &secretVault{store: sec}
	return true
}

// keysOf collects st's API keys.
func keysOf(st *Settings) map[string]string {
	keys := map[string]string{}
	for _, p := range st.Providers {
		if p.APIKey != "" {
			keys["provider/"+p.ID] = p.APIKey
		}
	}
	if st.Search.APIKey != "" {
		keys["search"] = st.Search.APIKey
	}
	return keys
}

// fillKeys sets the keys found in the vault on st; keys still in the
// file (from before the vault, or a failed write) win.
func (v *secretVault) fillKeys(st *Settings) {
	b, err := v.store.Get(secretsName)
	if err != nil {
		return
	}
	var keys map[string]string
	if json.Unmarshal(b, &keys) != nil {
		return
	}
	v.last = string(b)
	for _, p := range st.Providers {
		if p.APIKey == "" {
			p.APIKey = keys["provider/"+p.ID]
		}
	}
	if st.Search.APIKey == "" {
		st.Search.APIKey = keys["search"]
	}
}

// write stores keys, reporting success.
func (v *secretVault) write(keys map[string]string) bool {
	b, _ := json.Marshal(keys)
	v.mu.Lock()
	defer v.mu.Unlock()
	if string(b) == v.last {
		return true
	}
	if v.store.Set(secretsName, b) != nil {
		return false
	}
	v.last = string(b)
	return true
}

// withoutKeys returns a copy of st with the API keys blanked.
func withoutKeys(st *Settings) *Settings {
	c := *st
	c.Providers = make([]*ProviderConfig, len(st.Providers))
	for i, p := range st.Providers {
		pc := *p
		pc.APIKey = ""
		c.Providers[i] = &pc
	}
	c.Search.APIKey = ""
	return &c
}
