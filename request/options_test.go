package request_test

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestDownloadsRetainBodiesAndCloseOnErrors(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader("payload")}
	download, err := request.OpenDownload(body, "metadata", nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(download)
	if err != nil || string(data) != "payload" || download.Header != "metadata" || body.closed {
		t.Fatalf("data=%s header=%s err=%v closed=%v", data, download.Header, err, body.closed)
	}
	if err := download.Close(); err != nil || !body.closed {
		t.Fatal(err)
	}
	failure := errors.New("invalid header")
	body = &trackedBody{Reader: strings.NewReader("payload")}
	if download, err := request.OpenDownload(body, "", failure); download != nil || !errors.Is(err, failure) || !body.closed {
		t.Fatalf("download=%v err=%v closed=%v", download, err, body.closed)
	}
}

type input struct {
	Name string `json:"name,omitempty"`
}

func TestExtensionSnapshotsAndEnvelopeProtection(t *testing.T) {
	value := map[string]any{"enabled": false}
	option := request.WithField[input]("vendor:setting", value)
	value["enabled"] = true
	config, err := request.Apply(input{Name: "port"}, option)
	if err != nil {
		t.Fatal(err)
	}
	original := map[string]any{"port": map[string]any{"name": "port"}}
	body, err := request.MergeFields(original, config.Fields)
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(serialized) != `{"port":{"name":"port","vendor:setting":{"enabled":false}}}` {
		t.Fatalf("body=%s", serialized)
	}
	if len(original["port"].(map[string]any)) != 1 {
		t.Fatal("source body mutated")
	}
	config, err = request.Apply(input{}, request.WithField[input]("name", "override"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := request.MergeFields(original, config.Fields); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestCoreFieldsCannotBeSmuggledThroughExtensions(t *testing.T) {
	config, err := request.Apply(input{}, request.WithField[input]("name", "override"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := request.MergeFieldsFor(map[string]any{"port": map[string]any{}}, config.Fields, config.Options); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := request.Apply((*input)(nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestExtraQueryRetainsBaseAndEncodesValues(t *testing.T) {
	config, err := request.Apply(input{}, request.WithQuery[input]("vendor:field", "a&b"), request.WithQuery[input]("name", "new"))
	if err != nil {
		t.Fatal(err)
	}
	query, err := request.ExtendQuery("?name=old&limit=5", config.Query)
	if err != nil || query != "?limit=5&name=new&vendor%3Afield=a%26b" {
		t.Fatalf("query=%q err=%v", query, err)
	}
	for _, option := range []request.Option[input]{nil, request.WithField[input]("bad", make(chan int)), request.WithQuery[input]("", "value")} {
		if _, err := request.Apply(input{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
}
