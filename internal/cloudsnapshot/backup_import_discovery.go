package cloudsnapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
)

var backupImportPathVersion = regexp.MustCompile(`^v[0-9]+(?:\.[0-9]+)*$`)

// Discovery is finite and operation-owned. Strip only a trailing catalog
// project and version, preserving an encoded reverse-proxy path prefix.
func backupImportDiscoveryURLs(endpoint string) ([]string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimRight(u.EscapedPath(), "/"), "/")
	index := len(parts) - 1
	if index > 0 && !backupImportPathVersion.MatchString(parts[index]) && backupImportPathVersion.MatchString(parts[index-1]) {
		index--
	}
	if !backupImportPathVersion.MatchString(parts[index]) {
		return []string{gophercloud.NormalizeURL(endpoint)}, nil
	}
	build := func(length int) string {
		copy := *u
		copy.RawPath = strings.Join(parts[:length], "/") + "/"
		copy.Path, _ = url.PathUnescape(copy.RawPath)
		return copy.String()
	}
	return []string{build(index + 1), build(index)}, nil
}

func (p *reader) backupImportMicroversion(ctx context.Context) (string, []*MutationPage, error) {
	if p.source.Client.Microversion != "" {
		return p.source.Client.Microversion, nil, p.source.Guard(ctx)
	}
	targets, err := backupImportDiscoveryURLs(p.source.Client.Endpoint)
	if err != nil {
		return "", nil, err
	}
	var pages []*MutationPage
	for _, target := range targets {
		wire, err := p.backupImportDiscoveryGet(ctx, target)
		if wire != nil {
			pages = append(pages, mutationProof(wire))
		}
		if err != nil {
			// Only a direct fault-free native rejection authorizes fallback.
			// A joined IO/callback/source error or expanded OkCodes never does.
			if rejection, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && (rejection.Actual == 404 || rejection.Actual == 405) {
				continue
			}
			return "", pages, err
		}
		maximum, minimum, found, err := backupImportBounds(wire.Body)
		if err != nil {
			return "", pages, wire.Fail(err)
		}
		if !found {
			continue
		}
		version, err := selectBackupImportVersion(maximum, minimum)
		if err != nil {
			return "", pages, wire.Fail(err)
		}
		if err := p.source.Guard(ctx); err != nil {
			return "", pages, wire.Fail(err)
		}
		return version, pages, nil
	}
	return "", pages, p.source.Guard(ctx)
}

type backupImportAdvertisement struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Maximum string `json:"max_version"`
	Version string `json:"version"`
	Minimum string `json:"min_version"`
}

// Supported Cinder discovery forms: flat, version, versions, versions.values.
// Select the first usable v3 advertisement; no global range consistency rule
// is added to the Source's maximum/optional-minimum selection utility.
func backupImportBounds(body json.RawMessage) (string, string, bool, error) {
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
	var rows []backupImportAdvertisement
	if _, present := envelope["id"]; present {
		var row backupImportAdvertisement
		if err := json.Unmarshal(body, &row); err != nil {
			return "", "", false, err
		}
		rows = append(rows, row)
	} else if raw, present := envelope["version"]; present {
		var row *backupImportAdvertisement
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
		id, err := parseBackupImportVersion(row.ID)
		if err != nil {
			return "", "", false, err
		}
		if id[0] == nil || id[0].String() != "3" {
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

func (p *reader) backupImportDiscoveryGet(ctx context.Context, target string) (*rest.Response, error) {
	var faults rejectedPollFaults
	guard := func(ctx context.Context) error { return errors.Join(p.source.Guard(ctx), faults.error()) }
	client, err := fixedrequest.NewGuarded(&p.source.Client, http.MethodGet, target, guard)
	if err != nil {
		return nil, err
	}
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = rejectedPollTransport(func(req *http.Request) (*http.Response, error) {
		if (req.Body != nil && req.Body != http.NoBody) || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
			faults.add(invalid("backup discovery request must remain bodyless"))
			if req.Body != nil {
				faults.add(req.Body.Close())
			}
		}
		for key, values := range req.Header {
			if strings.EqualFold(key, "Transfer-Encoding") || (strings.EqualFold(key, "Content-Length") && (len(values) != 1 || values[0] != "0")) {
				faults.add(invalid("backup discovery changes physical body framing"))
			}
			if _, owned := backupActionVersion(key); owned {
				faults.add(invalid("backup discovery changes physical microversion headers"))
			}
		}
		if err := guard(req.Context()); err != nil {
			return nil, err
		}
		response, err := parent.RoundTrip(req)
		if response != nil && response.Body != nil && (response.StatusCode == 404 || response.StatusCode == 405) {
			response.Body = &rejectedPollBody{ReadCloser: response.Body, faults: &faults}
		}
		return response, err
	})
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, target, nil, nil, 200, 300)
}
