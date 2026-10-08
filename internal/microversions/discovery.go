package microversions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type Profile struct {
	Major                string
	MatchEndpointVersion bool
	RequireSelfLink      bool
	LegacyVersionHeader  string
	HeaderService        string
	OkCodes              []int
}

var NovaProfile = Profile{Major: "2", HeaderService: "compute", MatchEndpointVersion: true, RequireSelfLink: true, LegacyVersionHeader: "X-OpenStack-Nova-API-Version", OkCodes: novaCodes()}
var CinderProfile = Profile{Major: "3", HeaderService: "volume", LegacyVersionHeader: "X-OpenStack-Volume-API-Version", OkCodes: []int{200, 300}}

func novaCodes() []int {
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = index + 200
	}
	return codes
}

// Discovery retains actual accepted responses separately from the last clean
// unavailable rejection. A found row may omit either advertised bound.
type Discovery struct {
	Maximum, Minimum string
	Found            bool
	Responses        []*rest.Response
	LastRejection    error
}

// Read owns a finite version-endpoint/root discovery. Only fault-free native
// 404/405 or a valid document without a matching row permits the second GET.
// Advertised links never change the source route or authentication authority.
func Read(ctx context.Context, source *cloudread.Source, profile Profile) (Discovery, error) {
	var result Discovery
	if err := cloudread.Context(ctx); err != nil {
		return result, err
	}
	if source == nil {
		return result, invalid("discovery source is required")
	}
	profile.OkCodes = slices.Clone(profile.OkCodes)
	if profile.Major == "" || len(profile.OkCodes) == 0 {
		return result, invalid("discovery profile needs a major and explicit success codes")
	}
	guard := func(ctx context.Context) error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
	if err := guard(ctx); err != nil {
		return result, err
	}
	targets, endpointVersion, err := discoveryURLs(source.Client.Endpoint, profile.MatchEndpointVersion)
	if err != nil {
		return result, err
	}
	for _, target := range targets {
		wire, err := discoveryGet(ctx, source, target, profile)
		if wire != nil {
			result.Responses = append(result.Responses, wire)
		}
		if err != nil {
			if rejection, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && (rejection.Actual == 404 || rejection.Actual == 405) {
				result.LastRejection = err
				continue
			}
			return result, err
		}
		maximum, minimum, found, decodeErr := bounds(wire.Body, profile, endpointVersion)
		if err := errors.Join(decodeErr, guard(ctx)); err != nil {
			return result, wire.Fail(cloudread.ContextError(ctx, err))
		}
		if !found {
			continue
		}
		result.Maximum, result.Minimum, result.Found = maximum, minimum, true
		return result, nil
	}
	if err := guard(ctx); err != nil {
		if len(result.Responses) != 0 {
			return result, result.Responses[len(result.Responses)-1].Fail(err)
		}
		return result, err
	}
	return result, nil
}

var pathVersion = regexp.MustCompile(`^v[0-9]+(?:\.[0-9]+)*$`)

// Strip only a trailing catalog project and version, preserving the encoded
// reverse-proxy prefix. Endpoint matching is opt-in; Cinder keeps its v3 rule.
func discoveryURLs(endpoint string, matchEndpoint bool) ([]string, Version, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, err
	}
	parts := strings.Split(strings.TrimRight(u.EscapedPath(), "/"), "/")
	index := len(parts) - 1
	if index > 0 && !pathVersion.MatchString(parts[index]) && pathVersion.MatchString(parts[index-1]) {
		index--
	}
	if !pathVersion.MatchString(parts[index]) {
		return []string{gophercloud.NormalizeURL(endpoint)}, nil, nil
	}
	var endpointVersion Version
	if matchEndpoint {
		endpointVersion, err = ParseVersion(parts[index])
		if err != nil {
			return nil, nil, err
		}
	}
	build := func(length int) string {
		copy := *u
		copy.RawPath = strings.Join(parts[:length], "/") + "/"
		copy.Path, _ = url.PathUnescape(copy.RawPath)
		return copy.String()
	}
	return []string{build(index + 1), build(index)}, endpointVersion, nil
}

type advertisement struct {
	Links   json.RawMessage `json:"links"`
	ID      json.RawMessage `json:"id"`
	Status  string          `json:"status"`
	Maximum string          `json:"max_version"`
	Version string          `json:"version"`
	Minimum string          `json:"min_version"`
}

// Flat, version, versions and versions.values documents share one decoder.
// Return the first eligible major/API row without inventing range consistency
// or requiring optional bounds. Pure support/selection functions consume them.
func bounds(body json.RawMessage, profile Profile, endpointVersion Version) (string, string, bool, error) {
	if !utf8.Valid(body) {
		return "", "", false, fmt.Errorf("version discovery must be UTF-8 JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", "", false, err
	}
	if envelope == nil {
		return "", "", false, fmt.Errorf("version discovery must be a nonnull object")
	}
	var rows []advertisement
	if _, present := envelope["id"]; present {
		var row advertisement
		if err := json.Unmarshal(body, &row); err != nil {
			return "", "", false, err
		}
		rows = append(rows, row)
	} else if raw, present := envelope["version"]; present {
		var row *advertisement
		if err := json.Unmarshal(raw, &row); err != nil {
			return "", "", false, err
		}
		if row == nil {
			return "", "", false, fmt.Errorf("discovery version must be an object")
		}
		rows = append(rows, *row)
	} else if raw, present := envelope["versions"]; present {
		var nested map[string]json.RawMessage
		if json.Unmarshal(raw, &nested) == nil && nested != nil {
			var present bool
			raw, present = nested["values"]
			if !present {
				return "", "", false, fmt.Errorf("discovery versions lacks values")
			}
		}
		if err := json.Unmarshal(raw, &rows); err != nil {
			return "", "", false, err
		}
		if rows == nil {
			return "", "", false, fmt.Errorf("discovery versions must be an array")
		}
	}
	for _, row := range rows {
		if profile.RequireSelfLink {
			eligible, err := advertisedEndpoint(row)
			if err != nil {
				return "", "", false, err
			}
			if !eligible {
				continue
			}
		}
		var idText string
		if err := json.Unmarshal(row.ID, &idText); err != nil {
			return "", "", false, err
		}
		id, err := ParseVersion(idText)
		if err != nil {
			return "", "", false, err
		}
		if id[0] == nil || id[0].String() != profile.Major {
			continue
		}
		if profile.MatchEndpointVersion && endpointVersion != nil && id.Compare(endpointVersion) != 0 {
			continue
		}
		switch strings.ToUpper(row.Status) {
		case "", "CURRENT", "SUPPORTED", "STABLE", "DEPRECATED":
		default:
			continue
		}
		maximum := row.Maximum
		if maximum == "" {
			maximum = row.Version
		}
		return maximum, row.Minimum, true, nil
	}
	return "", "", false, nil
}

// Keystoneauth skips Nova advertisements without a permitted status or a
// self link. The link proves row eligibility but cannot override our route.
func advertisedEndpoint(row advertisement) (bool, error) {
	switch strings.ToLower(row.Status) {
	case "stable", "current", "supported", "deprecated":
	default:
		return false, nil
	}
	if len(row.ID) == 0 || len(row.Links) == 0 {
		return false, nil
	}
	var links []map[string]json.RawMessage
	if err := json.Unmarshal(row.Links, &links); err != nil {
		return false, err
	}
	if links == nil {
		return false, fmt.Errorf("discovery links must be an array")
	}
	for _, link := range links {
		var rel, href string
		if json.Unmarshal(link["rel"], &rel) != nil || strings.ToLower(rel) != "self" {
			continue
		}
		raw, present := link["href"]
		if !present || len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &href) != nil {
			continue
		}
		if _, err := url.Parse(href); err != nil {
			continue
		}
		return true, nil
	}
	return false, nil
}

func discoveryGet(ctx context.Context, source *cloudread.Source, target string, profile Profile) (*rest.Response, error) {
	var faults faultsState
	guard := func(ctx context.Context) error {
		return errors.Join(source.Guard(ctx), faults.error(), rest.CheckOperationGuard(ctx))
	}
	client, err := fixedrequest.NewGuarded(&source.Client, http.MethodGet, target, guard)
	if err != nil {
		return nil, err
	}
	// Discovery is versionless even when the action has a selected version.
	// Change only this private client, retaining the captured original source.
	client.Microversion = ""
	for key := range client.MoreHeaders {
		if versionHeader(key, profile) {
			delete(client.MoreHeaders, key)
		}
	}
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		if (req.Body != nil && req.Body != http.NoBody) || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
			faults.add(invalid("microversion discovery request must remain bodyless"))
			if req.Body != nil {
				faults.add(req.Body.Close())
			}
		}
		for key, values := range req.Header {
			if strings.EqualFold(key, "Transfer-Encoding") || (strings.EqualFold(key, "Content-Length") && (len(values) != 1 || values[0] != "0")) {
				faults.add(invalid("microversion discovery changes physical body framing"))
			}
			if versionHeader(key, profile) {
				faults.add(invalid("microversion discovery changes physical microversion headers"))
			}
		}
		if err := guard(req.Context()); err != nil {
			return nil, err
		}
		response, err := parent.RoundTrip(req)
		if response != nil && response.Body != nil && (response.StatusCode == 404 || response.StatusCode == 405) {
			response.Body = &observedBody{ReadCloser: response.Body, faults: &faults}
		}
		return response, err
	})
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, target, nil, nil, profile.OkCodes...)
}

func versionHeader(key string, profile Profile) bool {
	return strings.EqualFold(key, "OpenStack-API-Version") || profile.LegacyVersionHeader != "" && strings.EqualFold(key, profile.LegacyVersionHeader)
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type faultsState struct {
	mu    sync.Mutex
	cause error
}

func (f *faultsState) add(err error) {
	if err == nil {
		return
	}
	f.mu.Lock()
	f.cause = errors.Join(f.cause, err)
	f.mu.Unlock()
}
func (f *faultsState) error() error { f.mu.Lock(); defer f.mu.Unlock(); return f.cause }

type observedBody struct {
	io.ReadCloser
	faults *faultsState
}

func (b *observedBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	if err != io.EOF {
		b.faults.add(err)
	}
	return n, err
}
func (b *observedBody) Close() error { err := b.ReadCloser.Close(); b.faults.add(err); return err }

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
