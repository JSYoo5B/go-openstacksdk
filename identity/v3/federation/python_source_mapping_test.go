package federation_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func sourceMappingField(t *testing.T, row *sdk.SourceMappingRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s: %#v want %s", key, row, want)
	}
}
func sourceMappingBody(t *testing.T, raw string) map[string]json.RawMessage {
	t.Helper()
	var envelope map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope["mapping"]
}
func TestSourceMappingCreatePreservesRawRulesAndNullableDescriptorView(t *testing.T) {
	var calls []pyMappingRecordCall
	api := pyMappingRecordAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.Method != "PUT" || r.URL.EscapedPath() != pyMappingRecordBase+"/a%2Fb" {
			t.Fatal(r.Method, r.URL)
		}
		fields := sourceMappingBody(t, calls[len(calls)-1].body)
		if string(fields["id"]) != `"a/b"` || string(fields["rules"]) != `{"unknown_rule":{"keep":1}}` || string(fields["schema_version"]) != `false` || len(fields) != 3 {
			t.Fatal(fields)
		}
		return 203, `{"mapping":{"rules":{"actual":{"future":true}},"schema_version":false,"x_extension":1}}`
	})
	row, err := api.CreateSourceMappingRecord(context.Background(), sdk.WithSourceMappingAttribute("id", "a/b"), sdk.WithSourceMappingAttribute("rules", map[string]any{"unknown_rule": map[string]int{"keep": 1}}), sdk.WithSourceMappingAttribute("schema_version", false), sdk.WithSourceMappingAttribute("ignored", "unused"))
	if err != nil || row == nil || len(calls) != 1 || row.StatusCode != 203 {
		t.Fatal(row, err, calls)
	}
	sourceMappingField(t, row, "id", `"a/b"`)
	sourceMappingField(t, row, "name", `"a/b"`)
	sourceMappingField(t, row, "rules", `[{"actual":{"future":true}}]`)
	sourceMappingField(t, row, "schema_version", `"False"`)
	if row.Resource.Body["x_extension"] != nil || string(row.Wire.Body["x_extension"]) != "1" || !strings.Contains(string(row.Envelope), "x_extension") {
		t.Fatal(row)
	}
}
func TestSourceMappingCreateIgnoresNameDefaultsAndStatusPolicy(t *testing.T) {
	for _, code := range []int{200, 201, 202, 399} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			var calls []pyMappingRecordCall
			api := pyMappingRecordAPI(t, &calls, func(r *http.Request) (int, string) {
				if r.URL.Path != pyMappingRecordBase+"/first" {
					t.Fatal(r.URL)
				}
				fields := sourceMappingBody(t, calls[len(calls)-1].body)
				if len(fields) != 1 || string(fields["id"]) != `"first"` {
					t.Fatal(fields)
				}
				return code, `{}`
			})
			row, err := api.CreateSourceMappingRecord(context.Background(), sdk.WithSourceMappingAttribute("id", "first"), sdk.WithSourceMappingAttribute("name", "last"))
			if err != nil || row == nil || len(calls) != 1 {
				t.Fatal(row, err, calls)
			}
			sourceMappingField(t, row, "rules", `null`)
			sourceMappingField(t, row, "schema_version", `null`)
		})
	}
}
func TestSourceMappingUpdateDirtyNoopNullAndIgnoredName(t *testing.T) {
	for _, mode := range []string{"schema-only", "explicit-null", "alias-route", "noop", "unknown-only", "alias-only"} {
		t.Run(mode, func(t *testing.T) {
			var calls []pyMappingRecordCall
			api := pyMappingRecordAPI(t, &calls, func(r *http.Request) (int, string) {
				want := "selected"

				if r.Method != "PATCH" || r.URL.Path != pyMappingRecordBase+"/"+want {
					t.Fatal(r.Method, r.URL)
				}
				body := sourceMappingBody(t, calls[len(calls)-1].body)
				if body["id"] != nil || body["name"] != nil || len(body) != 1 {
					t.Fatal(body)
				}
				if mode == "explicit-null" && string(body["rules"]) != "null" {
					t.Fatal(body)
				}
				return 202, `{}`
			})
			var options []sdk.SourceMappingOption
			switch mode {
			case "schema-only":
				options = append(options, sdk.WithSourceMappingAttribute("schema_version", 2))
			case "explicit-null":
				options = append(options, sdk.WithSourceMappingAttribute("rules", nil))
			case "alias-route":
				options = append(options, sdk.WithSourceMappingAttribute("name", "other"), sdk.WithSourceMappingAttribute("rules", []any{}))
			case "unknown-only":
				options = append(options, sdk.WithSourceMappingAttribute("future", true))
			case "alias-only":
				options = append(options, sdk.WithSourceMappingAttribute("name", "other"))
			}
			row, err := api.UpdateSourceMappingRecord(context.Background(), "selected", options...)
			if err != nil || row == nil {
				t.Fatal(row, err, calls)
			}
			noop := mode == "noop" || mode == "unknown-only" || mode == "alias-only"
			if noop {
				if len(calls) != 0 || row.Wire != nil || row.Envelope != nil || row.Header != nil || row.StatusCode != 0 {
					t.Fatal(row, calls)
				}
			} else if len(calls) != 1 {
				t.Fatal(calls)
			}
			if mode == "schema-only" {
				sourceMappingField(t, row, "schema_version", `"2"`)
			}
			if mode == "alias-only" || mode == "alias-route" {
				sourceMappingField(t, row, "id", `"selected"`)
				sourceMappingField(t, row, "name", `"selected"`)
			}
		})
	}
}
func TestSourceMappingFindGETFirstFallbackPaginationAndDuplicate(t *testing.T) {
	for _, mode := range []string{"get", "fallback", "duplicate", "missing", "strict"} {
		t.Run(mode, func(t *testing.T) {
			var calls []pyMappingRecordCall
			api := pyMappingRecordAPI(t, &calls, func(r *http.Request) (int, string) {
				if strings.HasSuffix(r.URL.Path, "/needle") {
					if r.URL.RawQuery != "" {
						t.Fatal(r.URL)
					}
					if mode == "get" {
						return 203, `{"mapping":{"id":"actual","name":"ignored","rules":false,"schema_version":null}}`
					}
					return 403, `{}`
				}
				if r.URL.Query().Has("name") {
					t.Fatal("Mapping has no server name query", r.URL)
				}
				if mode == "missing" || mode == "strict" {
					return 200, `{"mappings":[]}`
				}
				if r.URL.Query().Get("marker") == "one" {
					return 200, `{"mappings":[{"id":"needle","schema_version":true}]}`
				}
				first := `{"id":"one"}`
				if mode == "duplicate" {
					first = `{"id":"needle"}`
				}
				return 200, `{"mappings":[` + first + `],"mappings_links":[{"rel":"next","href":"?marker=one"}]}`
			})
			var options []sdk.SourceMappingFindOption
			if mode == "strict" {
				options = append(options, sdk.WithSourceMappingFindIgnoreMissing(false))
			}
			row, err := api.FindSourceMappingRecord(context.Background(), "needle", options...)
			switch mode {
			case "get":
				if err != nil || len(calls) != 1 {
					t.Fatal(row, err, calls)
				}
				sourceMappingField(t, row, "id", `"actual"`)
				sourceMappingField(t, row, "rules", `[false]`)
			case "fallback":
				if err != nil || len(calls) != 3 {
					t.Fatal(row, err, calls)
				}
				sourceMappingField(t, row, "id", `"needle"`)
				sourceMappingField(t, row, "schema_version", `"True"`)
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || len(calls) != 3 {
					t.Fatal(row, err, calls)
				}
			case "missing":
				if row != nil || err != nil {
					t.Fatal(row, err)
				}
			case "strict":
				if row != nil || !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(row, err)
				}
			}
		})
	}
}
func TestSourceMappingOwnershipAndCallbackGuards(t *testing.T) {
	var calls []pyMappingRecordCall
	fields := map[string]json.RawMessage{"id": json.RawMessage(`"owned"`), "rules": json.RawMessage(`[{"future":1}]`)}
	headers := map[string]string{"X-Owned": "yes"}
	option := sdk.WithSourceMappingOpts(sdk.SourceMappingOpts{Attributes: fields, Headers: headers})
	fields["id"][1] = 'X'
	fields["rules"][11] = '2'
	headers["X-Owned"] = "changed"
	api := pyMappingRecordAPI(t, &calls, func(r *http.Request) (int, string) {
		if !strings.HasSuffix(r.URL.Path, "/owned") || r.Header.Get("X-Owned") != "yes" {
			t.Fatal(r.URL, r.Header)
		}
		return 201, `{"mapping":{"id":"owned","rules":[{"future":1}]}}`
	})
	row, err := api.CreateSourceMappingRecord(context.Background(), option)
	if err != nil || row == nil {
		t.Fatal(row, err)
	}
	cloned := row.Clone()
	cloned.Resource.Body["id"][1] = 'X'
	cloned.Wire.Body["id"][1] = 'X'
	cloned.Envelope[0] = '['
	sourceMappingField(t, row, "id", `"owned"`)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	before := len(calls)
	_, err = api.CreateSourceMappingRecord(ctx, func(c *sdk.SourceMappingOpts) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = api.FindSourceMappingRecord(context.Background(), "owned", func(c *sdk.SourceMappingFindOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}
func TestSourceMappingPreflightCollisionsAndInvalidBodies(t *testing.T) {
	for _, mode := range []string{"missing-create-id", "name-only-create", "update-id", "update-value", "update-mapping", "base-path", "invalid-json", "nil-option", "invalid-id"} {
		t.Run(mode, func(t *testing.T) {
			var calls []pyMappingRecordCall
			api := pyMappingRecordAPI(t, &calls, func(*http.Request) (int, string) { t.Fatal("unexpected HTTP"); return 500, "" })
			var err error
			switch mode {
			case "missing-create-id":
				_, err = api.CreateSourceMappingRecord(context.Background(), sdk.WithSourceMappingAttribute("rules", []any{}))
			case "name-only-create":
				_, err = api.CreateSourceMappingRecord(context.Background(), sdk.WithSourceMappingAttribute("name", "ignored"))
			case "update-id", "update-value", "update-mapping":
				_, err = api.UpdateSourceMappingRecord(context.Background(), "m", sdk.WithSourceMappingAttribute(strings.TrimPrefix(mode, "update-"), nil))
			case "base-path":
				_, err = api.CreateSourceMappingRecord(context.Background(), sdk.WithSourceMappingAttribute("base_path", "/other"))
			case "invalid-json":
				_, err = api.CreateSourceMappingRecord(context.Background(), sdk.WithSourceMappingOpts(sdk.SourceMappingOpts{Attributes: map[string]json.RawMessage{"id": json.RawMessage(`"m"`), "rules": json.RawMessage(`invalid`)}}))
			case "nil-option":
				_, err = api.CreateSourceMappingRecord(context.Background(), nil)
			case "invalid-id":
				_, err = api.CreateSourceMappingRecord(context.Background(), sdk.WithSourceMappingAttribute("id", ".."))
			}
			if err == nil || len(calls) != 0 {
				t.Fatal(err, calls)
			}
		})
	}
}

type sourceMappingFaultBody struct {
	io.Reader
	cause      error
	closeCause error
	close      func()
}

func (b *sourceMappingFaultBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if err == io.EOF && b.cause != nil {
		return n, b.cause
	}
	return n, err
}
func (b *sourceMappingFaultBody) Close() error {
	if b.close != nil {
		b.close()
	}
	return b.closeCause
}
func TestSourceMappingPhysicalFaultsPreserveProofAndStopFallback(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel", "accepted-shape"} {
		t.Run(mode, func(t *testing.T) {
			var calls []pyMappingRecordCall
			api := pyMappingRecordAPI(t, &calls, func(*http.Request) (int, string) { t.Fatal("base transport used"); return 500, "" })
			cause := errors.New("physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			count := 0
			api.RawClient().HTTPClient.Transport = pyMappingRecordTransport(func(r *http.Request) (*http.Response, error) {
				count++
				status := 404
				raw := `{"error":"physical"}`
				if mode == "accepted-shape" {
					status = 203
					raw = `[]`
				}
				body := &sourceMappingFaultBody{Reader: strings.NewReader(raw)}
				if mode == "read" {
					body.cause = cause
				}
				if mode == "close" {
					body.closeCause = cause
				}
				if mode == "cancel" {
					body.close = func() { cancel(cause) }
				}
				return &http.Response{StatusCode: status, Body: body, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"physical-proof"}}}, nil
			})
			row, err := api.FindSourceMappingRecord(ctx, "m")
			if err == nil || count != 1 {
				t.Fatal(row, err, count)
			}
			if mode != "accepted-shape" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "accepted-shape" {
				var physical *resource.ResponseError
				if !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != "[]" || physical.Header.Get("X-Request-Id") != "physical-proof" {
					t.Fatal(err)
				}
			}
		})
	}
}
