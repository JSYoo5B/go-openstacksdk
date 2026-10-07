package secretstores

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// SecretStore retains root-object HTTP fields and exact extension JSON. A
// literal canonical id takes priority; otherwise ID is derived passively from
// secret_store_ref. It is never a target URL. Body retains omission and null.
type SecretStore struct {
	resource.Metadata
	ID                string  `json:"-"`
	Name              string  `json:"name"`
	Status            string  `json:"status"`
	SecretStoreRef    string  `json:"secret_store_ref"`
	GlobalDefault     *bool   `json:"global_default"`
	CryptoPlugin      *string `json:"crypto_plugin"`
	SecretStorePlugin *string `json:"secret_store_plugin"`
	CreatedAt         *string `json:"created"`
	UpdatedAt         *string `json:"updated"`
}

func (value *SecretStore) UnmarshalJSON(data []byte) error {
	type plain SecretStore
	var metadata resource.Metadata
	if err := resource.DecodeObject(data, &struct{}{}, &metadata); err != nil {
		return err
	}
	// Decode only declared, exact wire keys. encoding/json otherwise lets
	// unknown case variants overwrite canonical fields or fail typed decode.
	// The complete original object still belongs to Metadata.Body.
	fields := make(map[string]json.RawMessage)
	for _, key := range []string{"name", "status", "secret_store_ref", "global_default", "crypto_plugin", "secret_store_plugin", "created", "updated"} {
		if raw, exists := metadata.Body[key]; exists {
			fields[key] = raw
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	var decoded plain
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	metadata.Header, metadata.StatusCode = value.Header, value.StatusCode
	decoded.Metadata = metadata
	*value = SecretStore(decoded)
	if id, exists := value.Body["id"]; exists {
		// Resource.id gives a literal id priority over alternate identity,
		// including an explicit empty string or null (represented by "").
		return json.Unmarshal(id, &value.ID)
	}
	if value.SecretStoreRef == "" {
		return nil
	}
	// Escape '%' for parsing only, so Parse's Path is the original spelling
	// rather than its percent-decoded form. Literal Unicode/space also stay
	// literal. No reconstructed/escaped URL ever leaves this decoder.
	parsed, err := url.Parse(strings.ReplaceAll(value.SecretStoreRef, "%", "%25"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path == "" {
		return fmt.Errorf("invalid absolute secret_store_ref %q", value.SecretStoreRef)
	}
	// Python HREFToUUID uses the final raw path component. Keep escaped
	// characters literal; no unescaping, UUID guessing or network follow-up.
	path := parsed.Path
	value.ID = path[strings.LastIndexByte(path, '/')+1:]
	return nil
}
