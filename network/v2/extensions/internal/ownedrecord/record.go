// Package ownedrecord owns the Neutron record workflow shared by SDK-owned
// packages whose pinned openstacksdk Resource class inherits Resource
// directly (no NetworkResource revision, If-Match or tag defaults). It drives
// internal/neutronfind for create, fetch, commit, delete, list and find, so
// every package reproduces the same Source projection, local Body filters,
// *_links and marker paging and ignore-missing defaults.
package ownedrecord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Spec describes one pinned Python Resource class. Profile.Path is the fixed
// collection path; a parented class leaves it empty and sets Parent (the URI
// attribute name), ParentArg (the proxy argument name), ParentLabel and
// Route, which receives the escaped parent. UpdateReserved lists the bound
// update_* argument names that **attrs cannot repeat.
type Spec struct {
	Kind           string
	Profile        neutronfind.Profile
	Parent         string
	ParentArg      string
	ParentLabel    string
	Route          func(escapedParent string) string
	UpdateReserved []string
}

// Binding is one package's API view: Client reads the API's current client so
// a replacement during an operation fails instead of mixing endpoints.
type Binding struct {
	Spec   *Spec
	Client func() *gophercloud.ServiceClient
}

// Record keeps the Python Resource view and the actual HTTP evidence. Resource
// holds every declared Body key (JSON null when unknown) and kept unknown Body
// keys; Wire is the actual unwrapped object, nil without one. Parent is the
// bound URI attribute for parented classes.
type Record struct {
	Resource, Wire *resource.RawResource
	Parent         string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// Field decodes one Resource Body value. JSON null, absence and a JSON type
// mismatch return nil; the raw value stays available in Resource.Body.
func Field[T any](r *Record, key string) *T {
	if r == nil || r.Resource == nil {
		return nil
	}
	raw, ok := r.Resource.Body[key]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	value := new(T)
	if err := json.Unmarshal(raw, value); err != nil {
		return nil
	}
	return value
}

// Text returns a Resource Body string, or "" when it is not a JSON string.
func Text(r *Record, key string) string {
	if value := Field[string](r, key); value != nil {
		return *value
	}
	return ""
}

func Invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// ValidateSegment accepts one nonblank UTF-8 segment without control
// characters. The segment is URL-escaped exactly once inside a route.
func ValidateSegment(label, value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return Invalid("%s must be a nonblank UTF-8 segment", label)
	}
	for _, c := range value {
		if unicode.IsControl(c) {
			return Invalid("%s has control characters", label)
		}
	}
	return nil
}

// Wrap adds operation context and any context cancellation cause.
func (b Binding) Wrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, b.Spec.Kind, cloudread.ContextError(ctx, err))
}

// engine captures the selected network client once per operation. A parented
// class binds the escaped parent into the collection and versioned paths.
func (b Binding) engine(ctx context.Context, parent string) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if b.Client == nil {
		return nil, Invalid("configured %s API is required", b.Spec.Kind)
	}
	profile := b.Spec.Profile
	if b.Spec.Parent != "" {
		if err := ValidateSegment(b.Spec.ParentLabel, parent); err != nil {
			return nil, err
		}
		profile.Path = b.Spec.Route(url.PathEscape(parent))
	}
	profile.VersionedPath = "/v2.0/" + profile.Path
	selected := b.Client()
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if b.Client() != selected {
			return Invalid("selected %s client changed", b.Spec.Kind)
		}
		return nil
	})
}

// record converts an engine record. URI attributes are consumed by Resource
// constructors and excluded from to_dict; a member response that introduces
// the same key is read back through the URI descriptor as the bound parent.
func (b Binding) record(value *neutronfind.Record, parent string) *Record {
	if value == nil {
		return nil
	}
	if key := b.Spec.Parent; key != "" && value.Resource != nil {
		delete(value.Resource.Body, key)
		if !value.FromList() && value.Wire != nil {
			if _, supplied := value.Wire.Body[key]; supplied {
				value.Resource.Body[key], _ = json.Marshal(parent)
			}
		}
	}
	return &Record{Resource: value.Resource, Wire: value.Wire, Parent: parent,
		Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
}
