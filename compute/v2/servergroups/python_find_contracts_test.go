package servergroups_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonGroupFindRecordUsesGETThenExactIDOrNameAndAllProjects(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			api, calls := pythonServerGroupAPI(t, func(req *http.Request) *http.Response {
				if req.URL.Path != pythonServerGroupBase {
					return pythonServerGroupWire(status, `{"error":"not an ID"}`)
				}
				return pythonServerGroupWire(200, `{"server_groups":[{"id":"unrelated","name":"other"},{"id":"web","name":"different","members":["s"],"member_ids":["last"],"rules":"not a mapping","policy":false,"opaque":9999999999999999999999999}]}`)
			})
			value, err := api.FindRecord(context.Background(), "web", servergroups.WithGroupFindAllProjects(true))
			if err != nil || value == nil || string(value.Resource.Body["id"]) != `"web"` || string(value.Resource.Body["name"]) != `"different"` || string(value.Resource.Body["member_ids"]) != `["last"]` || string(value.Resource.Body["rules"]) != `{}` || string(value.Resource.Body["policy"]) != "false" || string(value.Resource.Body["metadata"]) != "null" || string(value.Wire.Body["opaque"]) != "9999999999999999999999999" || len(*calls) != 2 || (*calls)[0].path != pythonServerGroupBase+"/web" || (*calls)[0].query != "" || (*calls)[1].query != "all_projects=True" {
				t.Fatal(value, err, *calls)
			}
		})
	}
	api, calls := pythonServerGroupAPI(t, func(*http.Request) *http.Response {
		return pythonServerGroupWire(200, `{"server_group":{"id":"g","name":"other","rules":null}}`)
	})
	value, err := api.FindRecord(context.Background(), "web", servergroups.WithGroupFindAllProjects(true))
	if err != nil || value == nil || string(value.Resource.Body["rules"]) != "null" || len(*calls) != 1 || (*calls)[0].query != "" {
		t.Fatal(value, err, *calls)
	}
}
func TestPythonGroupFindRecordMissingDefaultStrictAndDuplicate(t *testing.T) {
	body := `{"server_groups":[]}`
	api, calls := pythonServerGroupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path != pythonServerGroupBase {
			return pythonServerGroupWire(404, `{"error":"missing"}`)
		}
		return pythonServerGroupWire(200, body)
	})
	value, err := api.FindRecord(context.Background(), "missing")
	if err != nil || value != nil || len(*calls) != 2 {
		t.Fatal(value, err, *calls)
	}
	_, err = api.FindRecord(context.Background(), "missing", servergroups.WithGroupFindIgnoreMissing(false))
	if !errors.Is(err, resource.ErrNotFound) || len(*calls) != 4 {
		t.Fatal(err, *calls)
	}
	body = `{"server_groups":[{"id":"g1","name":"web"},{"id":"web","name":"other"}]}`
	_, err = api.FindRecord(context.Background(), "web")
	var duplicate *resource.AmbiguousError
	if !errors.As(err, &duplicate) || !reflect.DeepEqual(duplicate.IDs, []string{"g1", "web"}) || len(*calls) != 6 {
		t.Fatal(err, duplicate, *calls)
	}
}
func TestPythonGroupFindRecordReadsLaterPagesBeforeReturningCandidate(t *testing.T) {
	lateFailure := false
	api, calls := pythonServerGroupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path != pythonServerGroupBase {
			return pythonServerGroupWire(404, `{"error":"missing"}`)
		}
		if req.URL.Query().Get("marker") == "g1" {
			if lateFailure {
				return pythonServerGroupWire(503, `{"error":"late failure"}`)
			}
			return pythonServerGroupWire(200, `{"server_groups":[{"id":"g2","name":"web"}]}`)
		}
		return pythonServerGroupWire(200, `{"server_groups":[{"id":"g1","name":"web"}],"server_groups_links":[{"rel":"next","href":"?marker=g1"}]}`)
	})
	_, err := api.FindRecord(context.Background(), "web")
	var duplicate *resource.AmbiguousError
	if !errors.As(err, &duplicate) || len(*calls) != 3 {
		t.Fatal(err, *calls)
	}
	lateFailure = true
	_, err = api.FindRecord(context.Background(), "web")
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"late failure"}` || len(*calls) != 6 {
		t.Fatal(err, native, *calls)
	}
}
func TestPythonGroupFindRecordRejectedPhysicalErrorsPreventFallback(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, kind := range []string{"read", "close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, kind), func(t *testing.T) {
				api, _ := pythonServerGroupAPI(t, func(*http.Request) *http.Response { return pythonServerGroupWire(200, `{}`) })
				attempts := 0
				readErr, closeErr, cancelErr := errors.New("rejected read"), errors.New("rejected close"), errors.New("rejected cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				client := api.RawClient()
				original := client.Microversion
				defer func() { client.Microversion = original }()
				body := &groupFindFaultBody{data: []byte("physical rejection"), read: io.EOF}
				switch kind {
				case "read":
					body.read = readErr
				case "close":
					body.close = closeErr
				case "cancel":
					body.onClose = func() { cancel(cancelErr) }
				case "source":
					body.onRead = func() { client.Microversion = "2.100" }
					body.onClose = func() { client.Microversion = original }
				}
				client.ProviderClient.HTTPClient.Transport = pythonServerGroupTransport(func(*http.Request) (*http.Response, error) {
					attempts++
					return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"rejected"}}, Body: body}, nil
				})
				_, err := api.FindRecord(ctx, "web")
				var native gophercloud.ErrUnexpectedResponseCode
				if attempts != 1 || !errors.As(err, &native) || native.Actual != status || string(native.Body) != "physical rejection" || native.ResponseHeader.Get("X-Request-Id") != "rejected" {
					t.Fatal(err, native, attempts)
				}
				switch kind {
				case "read":
					if !errors.Is(err, readErr) {
						t.Fatal(err)
					}
				case "close":
					if !errors.Is(err, closeErr) {
						t.Fatal(err)
					}
				case "cancel":
					if !errors.Is(err, cancelErr) || !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				case "source":
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
func TestPythonGroupFindRecordCleanRetryErrorAllowsFallback(t *testing.T) {
	api, calls := pythonServerGroupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path != pythonServerGroupBase {
			return pythonServerGroupWire(404, `{"error":"missing"}`)
		}
		return pythonServerGroupWire(200, `{"server_groups":[{"id":"g1","name":"web"}]}`)
	})
	retries := 0
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		retries++
		return original
	}
	value, err := api.FindRecord(context.Background(), "web")
	if err != nil || value == nil || retries != 1 || len(*calls) != 2 {
		t.Fatal(value, err, retries, *calls)
	}
}
func TestPythonGroupFindRecordSnapshotsOptionsAndRejectsBeforeHTTP(t *testing.T) {
	api, calls := pythonServerGroupAPI(t, func(*http.Request) *http.Response { return pythonServerGroupWire(200, `{"server_group":{"id":"g"}}`) })
	var retained *servergroups.GroupFindOpts
	headers := map[string]string{"X-Owned": "old"}
	option := servergroups.WithGroupFindHeaders(headers)
	headers["X-Owned"] = "bad"
	_, err := api.FindRecord(context.Background(), "web", option, func(v *servergroups.GroupFindOpts) error { retained = v; return nil }, func(*servergroups.GroupFindOpts) error { retained.Headers["Authorization"] = "bad"; return nil })
	if err != nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
	for _, test := range []struct {
		name string
		call func() error
	}{
		{"nil option", func() error { _, err := api.FindRecord(context.Background(), "web", nil); return err }},
		{"invalid ID", func() error { _, err := api.FindRecord(context.Background(), ".."); return err }},
		{"auth header", func() error {
			_, err := api.FindRecord(context.Background(), "web", servergroups.WithGroupFindHeaders(map[string]string{"X-Auth-Token": "bad"}))
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 1 {
				t.Fatal(err, *calls)
			}
		})
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("caller cause"), errors.New("callback")
	_, err = api.FindRecord(ctx, "web", func(*servergroups.GroupFindOpts) error { cancel(cause); return callback })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}
func TestPythonGroupFindRecordSuccessfulReadClosePreservesProof(t *testing.T) {
	api, _ := pythonServerGroupAPI(t, func(*http.Request) *http.Response { return pythonServerGroupWire(200, `{}`) })
	readErr, closeErr := errors.New("accepted read"), errors.New("accepted close")
	attempts := 0
	api.RawClient().ProviderClient.HTTPClient.Transport = pythonServerGroupTransport(func(*http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"partial"}}, Body: &groupFindFaultBody{data: []byte(`{"server_group":`), read: readErr, close: closeErr}}, nil
	})
	value, err := api.FindRecord(context.Background(), "web")
	var receipt *resource.ResponseError
	if value != nil || attempts != 1 || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != `{"server_group":` || receipt.Header.Get("X-Request-Id") != "partial" {
		t.Fatal(value, err, receipt, attempts)
	}
}
func TestPythonGroupFindRecordDiscoversResourceCeilingAndHonorsSelectedVersion(t *testing.T) {
	api, _ := pythonServerGroupAPI(t, func(*http.Request) *http.Response { return pythonServerGroupWire(200, `{}`) })
	client := api.RawClient()
	client.Endpoint = client.ResourceBase
	client.Microversion = ""
	versions := []string{}
	client.ProviderClient.HTTPClient.Transport = pythonServerGroupTransport(func(req *http.Request) (*http.Response, error) {
		versions = append(versions, req.Header.Get("X-OpenStack-Nova-API-Version"))
		if req.URL.Path == "/nova/v2.1" || req.URL.Path == "/nova/v2.1/" {
			return pythonServerGroupWire(200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.100","links":[{"rel":"self","href":"`+client.ResourceBase+`"}]}}`), nil
		}
		return pythonServerGroupWire(200, `{"server_group":{"id":"g"}}`), nil
	})
	value, err := api.FindRecord(context.Background(), "web")
	if err != nil || value == nil || client.Microversion != "" || !reflect.DeepEqual(versions, []string{"", "2.64"}) {
		t.Fatal(value, err, versions)
	}
	client.Microversion = "2.1"
	versions = nil
	_, err = api.FindRecord(context.Background(), "web")
	if err != nil || !reflect.DeepEqual(versions, []string{"2.1"}) {
		t.Fatal(err, versions)
	}
}

type groupFindFaultBody struct {
	data            []byte
	read, close     error
	onRead, onClose func()
}

func (b *groupFindFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		if b.onRead != nil {
			b.onRead()
		}
		return n, nil
	}
	return 0, b.read
}
func (b *groupFindFaultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

var _ io.ReadCloser = (*groupFindFaultBody)(nil)
