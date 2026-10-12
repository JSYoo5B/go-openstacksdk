package services_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type svcActionTransport func(*http.Request) (*http.Response, error)

func (f svcActionTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, err
}

type svcActionCall struct {
	method, path, body string
	header             http.Header
}

const svcActionBase = "/cinder/v3/project/os-services"

func svcActionAPI(t *testing.T, version, maximum string, reply func(*http.Request) (int, string)) (*services.API, *[]svcActionCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]svcActionCall{}
	cloud.Provider.HTTPClient.Transport = svcActionTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
		}
		*calls = append(*calls, svcActionCall{r.Method, r.URL.EscapedPath(), body, r.Header.Clone()})
		code, text := 200, `{"version":{"id":"v3.0","status":"CURRENT","version":"`+maximum+`","min_version":"3.0"}}`
		if r.URL.Path != "/cinder/v3/" {
			code, text = reply(r)
		}
		return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(text))}, nil
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return services.New(client), calls
}

func svcActionOperation(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "services" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if code != 0 && (!errors.As(err, &native) || native.Actual != code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func svcActionVersion(t *testing.T, call svcActionCall, want string) {
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

func TestPythonServiceActionsSendServiceBodies(t *testing.T) {
	target := services.NewServiceTarget("cinder-volume", "host@lvm")
	for _, item := range []struct {
		name, path, body string
		call             func(*services.API) (*services.ServiceRecord, error)
	}{
		{"Enable", "/enable", `{"binary":"cinder-volume","host":"host@lvm"}`, func(a *services.API) (*services.ServiceRecord, error) {
			return a.Enable(context.Background(), target)
		}},
		{"Disable", "/disable", `{"binary":"cinder-volume","host":"host@lvm"}`, func(a *services.API) (*services.ServiceRecord, error) {
			return a.Disable(context.Background(), target, services.WithDisableReason(""))
		}},
		{"Disable", "/disable-log-reason", `{"binary":"cinder-volume","disabled_reason":"maint","host":"host@lvm"}`, func(a *services.API) (*services.ServiceRecord, error) {
			return a.Disable(context.Background(), target, services.WithDisableReason("maint"))
		}},
		{"Thaw", "/thaw", `{"host":"host@lvm"}`, func(a *services.API) (*services.ServiceRecord, error) {
			return a.Thaw(context.Background(), target)
		}},
		{"Freeze", "/freeze", `{"host":"host@lvm"}`, func(a *services.API) (*services.ServiceRecord, error) {
			return a.Freeze(context.Background(), target)
		}},
	} {
		t.Run(item.path, func(t *testing.T) {
			api, calls := svcActionAPI(t, "", "3.70", func(*http.Request) (int, string) {
				return 200, `{"disabled":true,"status":"disabled","host":"host@lvm","service":"cinder-volume","binary":"cinder-volume","disabled_reason":"maint"}`
			})
			v, err := item.call(api)
			if err != nil || len(*calls) != 1 || (*calls)[0].method != http.MethodPut || (*calls)[0].path != svcActionBase+item.path || (*calls)[0].body != item.body {
				t.Fatal(err, *calls)
			}
			svcActionVersion(t, (*calls)[0], "")
			if *v.Binary != "cinder-volume" || *v.Host != "host@lvm" || *v.Status != "disabled" || string(v.Body["name"]) != `"cinder-volume"` || string(v.Wire["service"]) != `"cinder-volume"` || v.StatusCode != 200 {
				t.Fatalf("service record: %#v", v)
			}

			api, calls = svcActionAPI(t, "3.0", "3.70", func(*http.Request) (int, string) { return 400, `{"badRequest":{}}` })
			_, err = item.call(api)
			svcActionVersion(t, (*calls)[0], "3.0")
			svcActionOperation(t, err, item.name, 400)
		})
	}

	// A Python service given only as an ID string sends null attributes.
	api, calls := svcActionAPI(t, "", "3.70", func(*http.Request) (int, string) { return 200, `` })
	v, err := api.Enable(context.Background(), services.ServiceTarget{})
	if err != nil || (*calls)[0].body != `{"binary":null,"host":null}` || v.Binary != nil || v.Wire != nil {
		t.Fatal(err, *calls, v)
	}
	listed := &services.Service{Binary: "cinder-backup", Host: "b1"}
	if got := services.TargetOf(listed); *got.Binary != "cinder-backup" || *got.Host != "b1" {
		t.Fatal(got)
	}
}

func TestPythonServiceFailoverChoosesActionBySupportedMicroversion(t *testing.T) {
	target := services.NewServiceTarget("cinder-volume", "host@lvm")
	for _, item := range []struct {
		version, maximum, path, body string
		options                      []services.FailoverOption
	}{
		{"", "3.70", "/failover", `{"backend_id":"b2","cluster":"c1","host":"host@lvm"}`, []services.FailoverOption{services.WithFailoverCluster("c1"), services.WithFailoverBackendID("b2")}},
		{"", "3.20", "/failover_host", `{"host":"host@lvm"}`, nil},
		{"3.0", "3.70", "/failover_host", `{"host":"host@lvm"}`, []services.FailoverOption{services.WithFailoverCluster("")}},
		{"3.26", "3.70", "/failover", `{"host":"host@lvm"}`, nil},
	} {
		api, calls := svcActionAPI(t, item.version, item.maximum, func(*http.Request) (int, string) { return 202, `` })
		v, err := api.Failover(context.Background(), target, item.options...)
		if err != nil || len(*calls) != 2 || (*calls)[0].path != "/cinder/v3/" || (*calls)[1].path != svcActionBase+item.path || (*calls)[1].body != item.body {
			t.Fatal(item, err, *calls)
		}
		svcActionVersion(t, (*calls)[1], item.version)
		if v.StatusCode != 202 || len(v.Discovery) != 1 || *v.Host != "host@lvm" {
			t.Fatalf("failover result: %#v", v)
		}
	}
	api, _ := svcActionAPI(t, "", "3.70", func(*http.Request) (int, string) { return 404, `{}` })
	_, err := api.Failover(context.Background(), target)
	svcActionOperation(t, err, "Failover", 404)
}

func TestPythonServiceLogLevelsRequireMicroversion332(t *testing.T) {
	api, calls := svcActionAPI(t, "", "3.70", func(*http.Request) (int, string) { return 202, `` })
	r, err := api.SetLogLevels(context.Background(), "DEBUG")
	if err != nil || len(*calls) != 2 || (*calls)[1].method != http.MethodPut || (*calls)[1].path != svcActionBase+"/set-log" || (*calls)[1].body != `{"binary":"","level":"DEBUG","prefix":null,"server":null}` {
		t.Fatal(err, *calls)
	}
	svcActionVersion(t, (*calls)[1], "3.32")
	if r.StatusCode != 202 || len(r.Discovery) != 1 {
		t.Fatalf("receipt: %#v", r)
	}

	api, calls = svcActionAPI(t, "3.40", "3.70", func(*http.Request) (int, string) {
		return 200, `{"log_levels":[{"binary":"cinder-api","host":"h1","levels":{"cinder":"DEBUG"}}]}`
	})
	levels, err := api.GetLogLevels(context.Background(), services.WithLogBinary("cinder-api"), services.WithLogServer("h1"), services.WithLogPrefix(""))
	if err != nil || len(*calls) != 1 || (*calls)[0].path != svcActionBase+"/get-log" || (*calls)[0].body != `{"binary":"cinder-api","prefix":"","server":"h1"}` {
		t.Fatal(err, *calls)
	}
	svcActionVersion(t, (*calls)[0], "3.40")
	if len(levels.Levels) != 1 || levels.Levels[0].Binary != "cinder-api" || levels.Levels[0].Host != "h1" || !reflect.DeepEqual(levels.Levels[0].Levels, map[string]string{"cinder": "DEBUG"}) || levels.StatusCode != 200 {
		t.Fatalf("levels: %#v", levels)
	}

	for _, item := range []struct{ version, maximum string }{{"", "3.31"}, {"3.31", "3.70"}} {
		api, calls = svcActionAPI(t, item.version, item.maximum, func(*http.Request) (int, string) {
			t.Fatal("log request below 3.32")
			return 0, ""
		})
		_, err = api.SetLogLevels(context.Background(), "INFO")
		if !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(item, err)
		}
		svcActionOperation(t, err, "SetLogLevels", 0)
	}

	for _, body := range []string{`{}`, `{"log_levels":{"a":1}}`, `{"log_levels":[{"binary":"x","host":"h"}]}`, `[]`} {
		api, _ = svcActionAPI(t, "3.32", "3.70", func(*http.Request) (int, string) { return 200, body })
		levels, err = api.GetLogLevels(context.Background())
		var accepted *resource.ResponseError
		if !errors.As(err, &accepted) || levels == nil || string(levels.Envelope) != body {
			t.Fatal(body, err, levels)
		}
		svcActionOperation(t, err, "GetLogLevels", 0)
	}
	api, _ = svcActionAPI(t, "3.32", "3.70", func(*http.Request) (int, string) { return 403, `{}` })
	_, err = api.GetLogLevels(context.Background())
	svcActionOperation(t, err, "GetLogLevels", 403)
}
