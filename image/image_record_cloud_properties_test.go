package image

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCloudImagePropertiesSelectsImageOrNameOrIDAndDelegates(t *testing.T) {
	var handler taskCoreTransport
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","vendor":"keep"}`, &handler)
	handler = func(req *http.Request) (*http.Response, error) {
		t.Fatal("cached properties need no request", req.Method, req.URL)
		return nil, nil
	}
	ctx := context.Background()
	// A Record wins over both string forms, which are never looked up.
	got, err := service.UpdateCloudImageProperties(ctx, ImageRecordCloudPropertiesRequest{Record: seed, ID: " ", NameOrID: " "})
	if err != nil || got == nil || !got.Updated || calls != 1 || imageRecordText(got.Record, "id") != "fixed" {
		t.Fatal(got, err, calls)
	}
	direct := func(id string) string {
		_, err := service.UpdateImagePropertiesRecord(ctx, ImageRecordPropertiesRequest{ID: id})
		if err == nil {
			t.Fatal("literal identities have no cached properties", id)
		}
		return strings.Replace(err.Error(), "UpdateImagePropertiesRecord", "UpdateCloudImageProperties", 1)
	}
	for _, test := range []struct {
		name  string
		input ImageRecordCloudPropertiesRequest
		want  string
	}{
		{"literal image ID wins over name_or_id", ImageRecordCloudPropertiesRequest{ID: "fixed", NameOrID: " "}, direct("fixed")},
		{"name_or_id is a literal identity", ImageRecordCloudPropertiesRequest{NameOrID: "by-name"}, direct("by-name")},
		{"empty selection", ImageRecordCloudPropertiesRequest{}, direct("")},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.UpdateCloudImageProperties(ctx, test.input)
			var operation *resource.OperationError
			if got != nil || err == nil || err.Error() != test.want || !errors.As(err, &operation) || operation.Operation != "UpdateCloudImageProperties" || errors.As(operation.Cause, &operation) || calls != 1 {
				t.Fatal(got, err, test.want, calls)
			}
		})
	}
	if _, err := service.UpdateCloudImageProperties(nil, ImageRecordCloudPropertiesRequest{Record: seed}); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(err)
	}
}
