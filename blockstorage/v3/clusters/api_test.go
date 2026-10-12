package clusters_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type clTransport func(*http.Request) (*http.Response, error)

func (f clTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, err
}

type clCall struct {
	method, path, query, body string
	header                    http.Header
}

const (
	clBase = "/cinder/v3/project/clusters"
	clRow  = `{"name":"c1@lvm","binary":"cinder-volume","state":"up","status":"enabled","frozen":false,"num_hosts":2,"num_down_hosts":0,"disabled_reason":null,"replication_status":"disabled","active_backend_id":null,"last_heartbeat":"t1","created_at":"t0","updated_at":"t2"}`
)

func clAPI(t *testing.T, version string, reply func(*http.Request) (int, string)) (*clusters.API, *[]clCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]clCall{}
	cloud.Provider.HTTPClient.Transport = clTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
		}
		*calls = append(*calls, clCall{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, body, r.Header.Clone()})
		code, text := 200, `{"version":{"id":"v3.0","status":"CURRENT","version":"3.70","min_version":"3.0"}}`
		if r.URL.Path != "/cinder/v3/" {
			code, text = reply(r)
		}
		return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(text))}, nil
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return clusters.New(client), calls
}

func clOperation(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "clusters" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if code != 0 && (!errors.As(err, &native) || native.Actual != code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func clVersion(t *testing.T, call clCall, want string) {
	t.Helper()
	if want == "" {
		if call.header.Get("OpenStack-API-Version") != "" || call.header.Get("X-OpenStack-Volume-API-Version") != "" {
			t.Fatalf("unexpected microversion headers: %v", call.header)
		}
		return
	}
	if call.header.Get("OpenStack-API-Version") != "volume "+want || call.header.Get("X-OpenStack-Volume-API-Version") != want {
		t.Fatalf("want microversion %s, got %v", want, call.header)
	}
}

func clCheck(t *testing.T, v *clusters.Cluster) {
	t.Helper()
	if v == nil || v.Name != "c1@lvm" || *v.Binary != "cinder-volume" || *v.State != "up" || *v.Status != "enabled" || *v.Frozen ||
		*v.NumHosts != 2 || *v.NumDownHosts != 0 || v.DisabledReason != nil || *v.ReplicationStatus != "disabled" ||
		v.ActiveBackendID != nil || *v.LastHeartbeat != "t1" || *v.CreatedAt != "t0" || *v.UpdatedAt != "t2" {
		t.Fatalf("decoded cluster: %#v", v)
	}
	if len(v.Body) != 14 {
		t.Fatalf("declared view: %#v", v.Body)
	}
}

func TestClusterGetNegotiatesMicroversion326(t *testing.T) {
	api, calls := clAPI(t, "", func(*http.Request) (int, string) { return 200, `{"cluster":` + clRow + `}` })
	v, err := api.Get(context.Background(), "c1@lvm")
	if err != nil || len(*calls) != 2 || (*calls)[1].method != http.MethodGet || (*calls)[1].path != clBase+"/c1@lvm" {
		t.Fatal(err, *calls)
	}
	clVersion(t, (*calls)[1], "3.26")
	clCheck(t, v)
	if string(v.Body["id"]) != `"c1@lvm"` {
		t.Fatalf("Python seeds the requested value as id: %s", v.Body["id"])
	}

	api, _ = clAPI(t, "3.7", func(*http.Request) (int, string) { return 404, `{"itemNotFound":{}}` })
	_, err = api.Get(context.Background(), "c1")
	clOperation(t, err, "Get", 404)
}

func TestClusterListDetailsQueryAndLocalFilters(t *testing.T) {
	api, calls := clAPI(t, "3.26", func(*http.Request) (int, string) {
		return 200, `{"clusters":[` + clRow + `,{"name":"c2","binary":"cinder-volume","state":"down"}]}`
	})
	values, err := api.All(context.Background(),
		clusters.WithListQuery("binary", "cinder-volume"),
		clusters.WithListQuery("is_up", true),
		clusters.WithListQuery("state", "up"),
		clusters.WithListQuery("unknown", 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0].path != clBase+"/detail" || (*calls)[0].query != "binary=cinder-volume&is_up=True" {
		t.Fatal(*calls)
	}
	clVersion(t, (*calls)[0], "3.26")
	if len(values) != 1 {
		t.Fatalf("local state filter: %#v", values)
	}
	clCheck(t, values[0])

	api, calls = clAPI(t, "3.26", func(*http.Request) (int, string) { return 200, `{"clusters":[{"name":"c2"}]}` })
	values, err = api.All(context.Background(), clusters.WithListDetails(false))
	if err != nil || len(values) != 1 || (*calls)[0].path != clBase || values[0].Name != "c2" {
		t.Fatal(err, *calls)
	}

	api, _ = clAPI(t, "3.26", func(*http.Request) (int, string) { return 403, `{}` })
	_, err = api.All(context.Background())
	clOperation(t, err, "List", 403)
}

func TestClusterFindFallsBackToDetailListByName(t *testing.T) {
	api, calls := clAPI(t, "3.26", func(r *http.Request) (int, string) {
		if r.URL.Path == clBase+"/c1@lvm" {
			return 404, `{}`
		}
		return 200, `{"clusters":[` + clRow + `]}`
	})
	v, err := api.Find(context.Background(), "c1@lvm")
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 2 || (*calls)[0].query != "" || (*calls)[1].path != clBase+"/detail" || (*calls)[1].query != "name=c1%40lvm" {
		t.Fatal(*calls)
	}
	clCheck(t, v)

	api, _ = clAPI(t, "3.26", func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/detail") {
			return 200, `{"clusters":[]}`
		}
		return 404, `{}`
	})
	if v, err = api.Find(context.Background(), "none"); err != nil || v != nil {
		t.Fatal(err, v)
	}
	_, err = api.Find(context.Background(), "none", clusters.WithFindIgnoreMissing(false))
	if !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	clOperation(t, err, "Find", 0)
}

func TestClusterEnableDisableUseConfiguredMicroversion(t *testing.T) {
	api, calls := clAPI(t, "", func(*http.Request) (int, string) { return 200, `{"cluster":` + clRow + `}` })
	v, err := api.Enable(context.Background(), clusters.NewTarget("c1@lvm", "cinder-volume"))
	if err != nil || len(*calls) != 1 || (*calls)[0].method != http.MethodPut || (*calls)[0].path != clBase+"/enable" || (*calls)[0].body != `{"binary":"cinder-volume","name":"c1@lvm"}` {
		t.Fatal(err, *calls)
	}
	clVersion(t, (*calls)[0], "")
	clCheck(t, v)

	api, calls = clAPI(t, "3.7", func(*http.Request) (int, string) {
		return 200, `{"cluster":{"name":"c1","status":"disabled","disabled_reason":"maint"}}`
	})
	v, err = api.Disable(context.Background(), clusters.NewTarget("c1", ""), clusters.WithDisableReason("maint"))
	if err != nil || (*calls)[0].path != clBase+"/disable" || (*calls)[0].body != `{"disabled_reason":"maint","name":"c1"}` {
		t.Fatal(err, *calls)
	}
	clVersion(t, (*calls)[0], "3.7")
	if *v.Status != "disabled" || *v.DisabledReason != "maint" {
		t.Fatalf("disable result: %#v", v)
	}

	// A plain Python string becomes the id, so the name is sent as null.
	api, calls = clAPI(t, "3.7", func(*http.Request) (int, string) { return 400, `{"badRequest":{}}` })
	_, err = api.Disable(context.Background(), clusters.Target{}, clusters.WithDisableReason(""))
	if (*calls)[0].body != `{"name":null}` {
		t.Fatal(*calls)
	}
	clOperation(t, err, "Disable", 400)

	// The target of a read cluster carries its name and binary.
	got := (&clusters.Cluster{Body: v.Body}).Target()
	if got.Name == nil || *got.Name != "c1" || got.Binary != nil {
		t.Fatalf("target: %#v", got)
	}
}
