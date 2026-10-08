package secrets

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FetchedSecret separates the actual metadata response from the payload.
// SecretID is the fixed request identity, not a fabricated response field.
// Body keeps original canonical and extension fields, including null/omission.
type FetchedSecret struct {
	resource.Metadata
	SecretID               string          `json:"-"`
	Name                   string          `json:"name"`
	Status                 string          `json:"status"`
	SecretRef              string          `json:"secret_ref"`
	Algorithm              string          `json:"algorithm"`
	Mode                   string          `json:"mode"`
	SecretType             string          `json:"secret_type"`
	BitLength              json.RawMessage `json:"bit_length"`
	ContentTypes           json.RawMessage `json:"content_types"`
	ExpiresAt              *string         `json:"expiration"`
	CreatedAt              *string         `json:"created"`
	UpdatedAt              *string         `json:"updated"`
	PayloadContentType     *string         `json:"payload_content_type"`
	PayloadContentEncoding *string         `json:"payload_content_encoding"`
	Payload                *FetchedPayload `json:"-"`
}

// FetchedPayload owns the exact payload response bytes and HTTP evidence.
// Text is populated only for an explicitly selected, valid UTF-8 text/plain
// response. Accept records the choice; response Content-Type cannot change it.
type FetchedPayload struct {
	Body       []byte
	Text       *string
	Accept     string
	Header     http.Header
	StatusCode int
}

func (value *FetchedSecret) UnmarshalJSON(data []byte) error {
	var raw resource.Metadata
	if err := resource.DecodeObject(data, &struct{}{}, &raw); err != nil {
		return err
	}
	// Project exact wire keys before typed decoding. Case variants remain raw
	// extensions and cannot shadow a canonical property through encoding/json.
	projection := make(map[string]json.RawMessage)
	for _, key := range []string{"name", "status", "secret_ref", "algorithm", "mode", "secret_type", "bit_length", "content_types", "expiration", "created", "updated", "payload_content_type", "payload_content_encoding"} {
		if field, exists := raw.Body[key]; exists {
			projection[key] = field
		}
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return err
	}
	type plain FetchedSecret
	var decoded plain
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	decoded.Metadata = raw
	*value = FetchedSecret(decoded)
	return nil
}
