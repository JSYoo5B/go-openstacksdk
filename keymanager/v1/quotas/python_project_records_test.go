package quotas_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonProjectQuotaRecordRawUpdateNullAndNoop(t *testing.T) {
	scope, calls := pythonRejudgeQuotaScope(t, pythonRejudgeQuotaReply{204, ""}, pythonRejudgeQuotaReply{203, `{"project_quotas":{"id":null,"secrets":false,"extension":9007199254740993}}`})
	noop, err := scope.UpdateRecord(context.Background(), quotas.WithRecordAttribute("unknown", func() {}))
	if err != nil || noop.Sent || noop.StatusCode != 0 || noop.Wire != nil || noop.Envelope != nil || len(*calls) != 0 || string(noop.Resource.Body["id"]) != `"project-1"` {
		t.Fatal(noop, err, *calls)
	}
	got, err := scope.UpdateRecord(context.Background(), quotas.WithRecordAttribute("secrets", nil), quotas.WithRecordAttribute("orders", "untyped"), quotas.WithRecordAttribute("consumers", false))
	if err != nil || !got.Sent || got.StatusCode != 204 || got.Wire != nil || string(got.Resource.Body["orders"]) != `"untyped"` || string(got.Resource.Body["secrets"]) != "null" {
		t.Fatal(got, err)
	}
	if len(*calls) != 1 || (*calls)[0] != `PUT /key-manager/v1/project-quotas/project-1 {"project_quotas":{"consumers":false,"orders":"untyped","secrets":null}}` {
		t.Fatal(*calls)
	}
	fetched, err := scope.GetRecord(context.Background())
	if err != nil || fetched.RequestID != "project-1" || fetched.StatusCode != 203 || string(fetched.Resource.Body["id"]) != "null" || string(fetched.Resource.Body["secrets"]) != "false" || fetched.Resource.Body["extension"] != nil || string(fetched.Wire.Body["extension"]) != "9007199254740993" {
		t.Fatal(fetched, err)
	}
}
func TestPythonProjectQuotaRecordOwnOptionsAndMerge(t *testing.T) {
	scope, calls := pythonRejudgeQuotaScope(t, pythonRejudgeQuotaReply{202, `{"name":"server","orders":null}`})
	attrs := map[string]json.RawMessage{"name": json.RawMessage(`"seed"`), "orders": json.RawMessage(`123`), "secrets": json.RawMessage(`900719925474099312345`), "unknown": json.RawMessage(`invalid`)}
	option := quotas.WithRecordOpts(quotas.RecordOpts{Attributes: attrs})
	attrs["secrets"][0] = '1'
	got, err := scope.UpdateRecord(context.Background(), option)
	if err != nil || string(got.Resource.Body["secrets"]) != "900719925474099312345" || string(got.Resource.Body["orders"]) != "null" || string(got.Resource.Body["name"]) != `"server"` || got.Wire.Body["secrets"] != nil {
		t.Fatal(got, err, *calls)
	}
	clone := got.Clone()
	clone.Resource.Body["secrets"][0] = '0'
	clone.Envelope[0] = '!'
	if string(got.Resource.Body["secrets"]) != "900719925474099312345" || string(got.Envelope) != `{"name":"server","orders":null}` {
		t.Fatal(got)
	}
}
func TestPythonProjectQuotaRecordInvalidInputSourceAndCancellation(t *testing.T) {
	scope, calls := pythonRejudgeQuotaScope(t)
	for _, option := range []quotas.RecordOption{nil, quotas.WithRecordAttribute("id", "other"), quotas.WithRecordAttribute("connection", nil), quotas.WithRecordOpts(quotas.RecordOpts{Attributes: map[string]json.RawMessage{"secrets": json.RawMessage(`bad`)}})} {
		if _, err := scope.UpdateRecord(context.Background(), option); err == nil {
			t.Fatal("invalid accepted")
		}
	}
	if _, err := scope.GetRecord(context.Background(), quotas.WithRecordAttribute("name", "not supported")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	mutate := func(*quotas.RecordOpts) error { scope.RawClient().Endpoint = "https://foreign.invalid/"; return nil }
	if _, err := scope.UpdateRecord(context.Background(), mutate); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop")
	cancel(cause)
	if _, err := scope.GetRecord(ctx); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
}

type projectRecordTransport func(*http.Request) (*http.Response, error)

func (f projectRecordTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type projectRecordBody struct {
	io.Reader
	cause error
}

func (b projectRecordBody) Close() error { return b.cause }
func TestPythonProjectQuotaRecordFaultsKeepReceiptAndNeverReplay(t *testing.T) {
	for _, status := range []int{202, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("key-manager", "/barbican/v1")
			scope, err := quotas.New(client).InProject(context.Background(), resource.ID("project"))
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("close")
			attempts := 0
			client.ProviderClient.HTTPClient.Transport = projectRecordTransport(func(r *http.Request) (*http.Response, error) {
				attempts++
				return &http.Response{Request: r, StatusCode: status, Header: http.Header{"X-Proof": {"project"}}, Body: projectRecordBody{strings.NewReader(`{"project_quotas":{"name":"READY"}}`), cause}}, nil
			})
			_, err = scope.UpdateRecord(context.Background(), quotas.WithRecordAttribute("name", "BUILD"))
			if attempts != 1 || !errors.Is(err, cause) {
				t.Fatal(attempts, err)
			}
			if status == 202 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 202 || proof.Header.Get("X-Proof") != "project" {
					t.Fatal(err, proof)
				}
			} else {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 404 || proof.ResponseHeader.Get("X-Proof") != "project" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}
