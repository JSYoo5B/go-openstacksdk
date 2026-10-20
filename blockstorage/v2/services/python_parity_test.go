package services_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

type pythonServiceTransport func(*http.Request) (*http.Response, error)

func (transport pythonServiceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func TestPythonServicesListFiltersByBinaryAndHost(t *testing.T) {
	var paths []string
	var queries []url.Values
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonServiceTransport(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.Method+" "+req.URL.Path)
		queries = append(queries, req.URL.Query())
		body := `{"services":[{"binary":"cinder-volume","host":"host@lvm","zone":"nova","status":"enabled","state":"up","disabled_reason":null,"frozen":false,"replication_status":"disabled","active_backend_id":null,"updated_at":"2026-10-11T01:02:03.000000"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	// services(binary="cinder-volume", host="host@lvm")
	var rows []*services.Service
	for value, err := range services.New(client).List(context.Background(), services.WithListOptions(services.ListOpts{Binary: "cinder-volume", Host: "host@lvm"})) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 1 || rows[0].Binary != "cinder-volume" || rows[0].Host != "host@lvm" || rows[0].Zone != "nova" || rows[0].State != "up" || rows[0].Frozen || rows[0].ReplicationStatus != "disabled" || rows[0].UpdatedAt.Second() != 3 {
		t.Fatal(rows)
	}
	if !reflect.DeepEqual(paths, []string{"GET /cinder/v2/project/os-services"}) || !reflect.DeepEqual(queries[0], url.Values{"binary": {"cinder-volume"}, "host": {"host@lvm"}}) {
		t.Fatal(paths, queries)
	}
}
