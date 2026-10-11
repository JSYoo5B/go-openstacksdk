package quotasets_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonRejudgeTransport func(*http.Request) (*http.Response, error)

func (transport pythonRejudgeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonRejudgeCall struct {
	method, path, query string
	body                any
}

func pythonRejudgeScope(t *testing.T, calls *[]pythonRejudgeCall, replies map[string]string) *quotasets.ProjectQuotaScope {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonRejudgeTransport(func(req *http.Request) (*http.Response, error) {
		var body any
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			if len(data) > 0 {
				if err := json.Unmarshal(data, &body); err != nil {
					t.Errorf("body %q: %v", data, err)
				}
			}
		}
		path := strings.TrimPrefix(req.URL.Path, "/cinder/v3")
		*calls = append(*calls, pythonRejudgeCall{req.Method, path, req.URL.RawQuery, body})
		reply, ok := replies[req.Method+" "+path]
		code := 200
		if !ok {
			code, reply = 404, `{"itemNotFound":{"code":404,"message":"not found"}}`
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(reply)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("volumev3", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("proj"))
	if err != nil || scope.ProjectID() != "proj" {
		t.Fatal(scope, err)
	}
	return scope
}

func pythonRejudgeJSON(t *testing.T, raw string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(raw, err)
	}
	return value
}

const pythonRejudgeQuotaSet = `{"id":"proj","volumes":10,"snapshots":10,"gigabytes":1000,"per_volume_gigabytes":-1,"backups":10,"backup_gigabytes":1000,"groups":10,"volumes_lvm":-1}`

// get_quota_set_defaults(project) fetches os-quota-sets/{project}/defaults.
func TestPythonRejudgeQuotaSetDefaults(t *testing.T) {
	var calls []pythonRejudgeCall
	scope := pythonRejudgeScope(t, &calls, map[string]string{"GET /os-quota-sets/proj/defaults": `{"quota_set":` + pythonRejudgeQuotaSet + `}`})
	got, err := scope.Defaults(context.Background())
	if err != nil || got.ProjectID != "proj" || got.Volumes != 10 || got.Gigabytes != 1000 || got.PerVolumeGigabytes != -1 || got.Groups != 10 || string(got.Body["volumes_lvm"]) != "-1" {
		t.Fatal(got, err)
	}
	want := []pythonRejudgeCall{{"GET", "/os-quota-sets/proj/defaults", "", nil}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls\n got %+v\nwant %+v", calls, want)
	}
}

// update_quota_set(project, volumes=5, gigabytes=0, volumes_lvm=3, force=True)
// PUTs only the passed attributes in the quota_set envelope.
func TestPythonRejudgeQuotaSetUpdate(t *testing.T) {
	var calls []pythonRejudgeCall
	scope := pythonRejudgeScope(t, &calls, map[string]string{"PUT /os-quota-sets/proj": `{"quota_set":` + pythonRejudgeQuotaSet + `}`})
	five, zero := 5, 0
	got, err := scope.Update(context.Background(), quotasets.UpdateOpts{Volumes: &five, Gigabytes: &zero},
		quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeVolumes, "lvm", 3), quotasets.WithUpdateForce(true))
	if err != nil || got.ProjectID != "proj" || got.Volumes != 10 || string(got.Body["volumes_lvm"]) != "-1" {
		t.Fatal(got, err)
	}
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{Snapshots: &five}, quotasets.WithUpdateField("backups_extra", 2)); err != nil {
		t.Fatal(err)
	}
	want := []pythonRejudgeCall{
		{"PUT", "/os-quota-sets/proj", "", pythonRejudgeJSON(t, `{"quota_set":{"volumes":5,"gigabytes":0,"volumes_lvm":3,"force":true}}`)},
		{"PUT", "/os-quota-sets/proj", "", pythonRejudgeJSON(t, `{"quota_set":{"snapshots":5,"backups_extra":2}}`)},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls\n got %+v\nwant %+v", calls, want)
	}
}
