package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"gophercloudsdk/compute"
	"gophercloudsdk/image"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestWaitWorkflowAttributesAreValidatedBeforeLookupOrMutation(t *testing.T) {
	for _, policy := range []struct {
		name      string
		attribute string
		callback  bool
	}{
		{name: "missing field", attribute: "does_not_exist"},
		{name: "nonstring field"},
		{name: "nil callback", callback: true},
	} {
		for _, workflow := range []string{"server", "image", "floating IP"} {
			t.Run(workflow+"/"+policy.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					httpWaitWorkflowFailure(t, w, r)
				})
				attribute := policy.attribute
				if policy.name == "nonstring field" {
					switch workflow {
					case "server":
						attribute = "Progress"
					case "image":
						attribute = "SizeBytes"
					case "floating IP":
						attribute = "RevisionNumber"
					}
				}
				option := resource.WithStatusAttribute(attribute)
				if policy.callback {
					option = resource.WithProgressCallback(nil)
				}
				var err error
				switch workflow {
				case "server":
					service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{})
					value, failure := service.Servers.Create(context.Background(), compute.CreateServerRequest{Name: "worker", Image: resource.Name("image"), Flavor: resource.Name("flavor")}, compute.WithWait(option))
					if value != nil {
						t.Fatalf("invalid policy created server: %v", value)
					}
					err = failure
				case "image":
					data := strings.NewReader("image bytes")
					value, failure := image.New(cloud.Client("image", "/v2")).Upload(context.Background(), image.UploadImageRequest{Name: "image", Data: data}, image.WithWait(option))
					if value != nil || data.Len() != len("image bytes") {
						t.Fatalf("invalid policy consumed upload: value=%v remaining=%d", value, data.Len())
					}
					err = failure
				case "floating IP":
					service := network.New(cloud.Client("network", "/v2.0"))
					value, failure := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.Name("public")}, network.WithPort(resource.Name("port")), network.WithWait(option))
					if value != nil {
						t.Fatalf("invalid policy allocated floating IP: %v", value)
					}
					err = failure
				}
				if requests.Load() != 0 || (!errors.Is(err, resource.ErrUnsupported) && !errors.Is(err, resource.ErrInvalidOption)) {
					t.Fatalf("requests=%d error=%v", requests.Load(), err)
				}
			})
		}
	}
}

func httpWaitWorkflowFailure(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	t.Errorf("invalid wait policy reached HTTP %s %s", r.Method, r.URL)
	http.Error(w, "unexpected request", http.StatusInternalServerError)
}
