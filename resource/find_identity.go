package resource

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
)

// FindIdentity automatically resolves one controller identity or exact name.
// Safe unescaped segments use GET first. Names that cannot safely use an ID
// route are searched only through the list query; FindFallbackNever rejects
// those names before HTTP. Identity strings are never trimmed or URL-decoded.
// Audited SDK bindings opt in explicitly; existing ID/Name lookup is unchanged.
func (c *Collection[T]) FindIdentity(ctx context.Context, identity string, options ...IdentityFindOption) (*T, error) {
	kind := ""
	if c != nil {
		kind = c.binding.Kind
	}
	fail := func(err error) (*T, error) {
		return nil, &OperationError{Operation: "find_identity", Resource: kind, Cause: err}
	}
	config, err := parseIdentityFindOptions(options)
	if err == nil {
		err = validateIdentityFindInput(identity)
	}
	if err != nil {
		return fail(err)
	}
	if c == nil || !c.binding.IdentityFind || c.binding.Get == nil || c.binding.ID == nil || c.binding.Name == nil {
		return fail(ErrUnsupported)
	}
	if ctx == nil {
		return fail(invalid("identity find context is required"))
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	ignoreMissing := config.IgnoreMissing == nil || *config.IgnoreMissing
	safeRoute := safeIdentityFindRoute(identity) && c.validateID(identity) == nil
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if safeRoute {
		value, err := c.Get(ctx, identity)
		if canceled := ctx.Err(); canceled != nil {
			return fail(canceled)
		}
		if err == nil {
			if _, err := c.identityFindID(value); err != nil {
				return fail(err)
			}
			if err := ctx.Err(); err != nil {
				return fail(err)
			}
			return value, nil
		}
		if !identityFindCanFallback(err, config.Fallback) {
			if !identityFindTerminalError(err) && gophercloud.ResponseCodeIs(err, http.StatusNotFound) && ignoreMissing {
				return nil, nil
			}
			return fail(err)
		}
	} else if config.Fallback == FindFallbackNever {
		return fail(invalid("identity cannot safely use a direct ID route"))
	}
	// A successful list observation is required before treating absence as
	// ignorable. List HTTP failures are never suppressed by IgnoreMissing.
	if c.binding.IterateControlled == nil && c.binding.Iterate == nil && (c.binding.List == nil || c.binding.Extract == nil) {
		return fail(ErrUnsupported)
	}
	if c.binding.NameQuery != nil {
		key := c.binding.NameQueryKey
		if key == "" {
			key = "name"
		}
		if err := validateIdentityFindQueryKey(key); err != nil {
			return fail(err)
		}
		if !config.Query.Has(key) {
			config.Query.Set(key, c.binding.NameQuery(identity))
		}
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	// WithName would filter out rows matching only their ID. Copy the validated
	// wire query directly: repeated WithQuery calls would collapse multiple
	// values and drop explicitly present keys with empty slices.
	query := cloneIdentityFindOptions(config).Query
	listQuery := func(options *listOptions) error {
		options.query = cloneIdentityFindOptions(IdentityFindOpts{Query: query}).Query
		return nil
	}
	var found *T
	var foundID string
	for value, err := range c.List(ctx, listQuery) {
		if err != nil {
			return fail(err)
		}
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		id, err := c.identityFindID(value)
		if err != nil {
			return fail(err)
		}
		name := c.binding.Name(value)
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		if id != identity && name != identity {
			continue
		}
		if found != nil {
			return fail(&AmbiguousError{Resource: kind, Name: identity, IDs: []string{foundID, id}})
		}
		found, foundID = value, id
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if found == nil && !ignoreMissing {
		// Do not attach a suppressed direct 400/403 to logical list absence.
		return fail(&NotFoundError{Resource: kind, Reference: identity})
	}
	return found, nil
}

func validateIdentityFindInput(identity string) error {
	if strings.TrimSpace(identity) == "" || !utf8.ValidString(identity) {
		return invalid("identity must be nonempty UTF-8")
	}
	for _, char := range identity {
		if unicode.IsControl(char) {
			return invalid("identity must not contain control characters")
		}
	}
	return nil
}

func safeIdentityFindRoute(identity string) bool {
	if ID(identity).Validate() != nil {
		return false
	}
	for _, char := range identity {
		if unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

func (c *Collection[T]) identityFindID(value *T) (string, error) {
	if value == nil {
		return "", invalid("identity response must contain a resource")
	}
	id := c.binding.ID(value)
	if err := validateIdentityFindInput(id); err != nil {
		return "", err
	}
	if err := c.validateID(id); err != nil {
		return "", err
	}
	return id, nil
}

func identityFindTerminalError(err error) bool {
	var accepted *ResponseError
	var transport *url.Error
	var syntax *json.SyntaxError
	var typed *json.UnmarshalTypeError
	var target *json.InvalidUnmarshalError
	return errors.As(err, &accepted) || errors.As(err, &transport) || errors.As(err, &syntax) || errors.As(err, &typed) || errors.As(err, &target) ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, ErrInvalidOption) || errors.Is(err, ErrUnsupported)
}

func identityFindCanFallback(err error, policy FindFallbackPolicy) bool {
	if policy == FindFallbackNever || identityFindTerminalError(err) {
		return false
	}
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return true
	}
	return policy == FindFallbackCompatible && (gophercloud.ResponseCodeIs(err, http.StatusBadRequest) || gophercloud.ResponseCodeIs(err, http.StatusForbidden))
}
