package compute_test

import (
	"bytes"
	"context"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2"
)

// Reuse the public-testhelper-backed Nova fixture. Final 1xx responses are
// synthetic Adapter evidence, not informational replies from a live server.
func TestCloudFlavorRAMSourceStatusListAndEnrichment(t *testing.T) {
	for _, tc := range []struct {
		name        string
		list, extra int
		fail        int
	}{
		{"list199", 199, 200, 0}, {"extra199", 200, 199, 0},
		{"bothLowerEdges", 100, 100, 0}, {"bothUpperEdges", 399, 399, 0},
		{"list400", 400, 200, 400}, {"extra400", 200, 400, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, c, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
				if r.URL.Path == flavorRAMListPath {
					return backupServerWire(tc.list, `{"flavors":[{"id":"f","name":"m1","ram":4}]}`, "source-list")
				}
				if r.URL.Path != "/nova/v2.1/flavors/f/os-extra_specs" {
					t.Error(r.URL)
				}
				return backupServerWire(tc.extra, `{"extra_specs":{"proof":"actual"}}`, "source-extra")
			})
			c.Microversion = "2.55"
			result, err := s.GetFlavorByRAM(context.Background(), 4)
			if tc.fail != 0 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != tc.fail || native.ResponseHeader.Get("X-Backup-Stage") == "" {
					t.Fatal(result, err, native)
				}
				return
			}
			if err != nil || flavorRAMID(t, result) != "f" || len(*calls) != 2 {
				t.Fatal(result, err, *calls)
			}
			row := result.Flavor
			if row.StatusCode != tc.list || row.Header.Get("X-Backup-Stage") != "source-list" || string(row.Envelope) != `{"id":"f","name":"m1","ram":4}` || row.Enrichment == nil || row.Enrichment.StatusCode != tc.extra || row.Enrichment.Header.Get("X-Backup-Stage") != "source-extra" || string(row.Enrichment.ExtraSpecs) != `{"proof":"actual"}` {
				t.Fatal(row)
			}
		})
	}
}

func TestCloudFlavorRAMSourceStatusPreservesExistingReaders(t *testing.T) {
	for _, operation := range []string{"AllFlavors", "SearchFlavors", "GetFlavor", "ListRecords", "FetchExtraSpecs", "AllFlavorsExtra"} {
		t.Run(operation, func(t *testing.T) {
			s, c, _ := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
				if r.URL.Path == flavorRAMListPath {
					status := 199
					if operation == "AllFlavorsExtra" {
						status = 200
					}
					return backupServerWire(status, `{"flavors":[{"id":"f","name":"m1","ram":4}]}`, "legacy-list")
				}
				if strings.HasSuffix(r.URL.Path, "/os-extra_specs") {
					return backupServerWire(199, `{"extra_specs":{"proof":"actual"}}`, "legacy-extra")
				}
				return backupServerWire(199, `{"id":"f","name":"m1","ram":4}`, "legacy-member")
			})
			c.Microversion = "2.55"
			var err error
			switch operation {
			case "AllFlavors":
				_, err = s.AllFlavors(context.Background())
			case "AllFlavorsExtra":
				_, err = s.AllFlavors(context.Background(), compute.WithFlavorQueryExtraSpecs(true))
			case "SearchFlavors":
				_, err = s.SearchFlavors(context.Background(), "m1")
			case "GetFlavor":
				_, err = s.GetFlavor(context.Background(), "f")
			case "ListRecords":
				for _, e := range s.API.Flavors.ListRecords(context.Background()) {
					if e != nil {
						err = e
						break
					}
				}
			case "FetchExtraSpecs":
				_, err = s.API.Flavors.FetchExtraSpecs(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "f"})
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 199 || native.ResponseHeader.Get("X-Backup-Stage") == "" || len(native.Body) == 0 {
				t.Fatal(err, native)
			}
		})
	}
}

// Keep the separately audited decoder gap explicit: status compatibility does
// not imply Requests-compatible response charset decoding.
func TestCloudFlavorRAMSourceStatusDoesNotClaimUTF16Decoding(t *testing.T) {
	for _, stage := range []string{"list", "extra"} {
		t.Run(stage, func(t *testing.T) {
			s, c, _ := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
				list := r.URL.Path == flavorRAMListPath
				body := `{"flavors":[{"id":"f","name":"m1","ram":4}]}`
				if !list {
					body = `{"extra_specs":{"proof":"actual"}}`
				}
				response := backupServerWire(199, body, "ascii")
				if list == (stage == "list") {
					data := []byte{0xff, 0xfe}
					for _, v := range body {
						data = append(data, byte(v), byte(v>>8))
					}
					response.Body = io.NopCloser(bytes.NewReader(data))
					response.Header.Set("Content-Type", "application/json;charset=UTF-16")
					response.Header.Set("X-Backup-Stage", "utf16-"+stage)
				}
				return response
			})
			c.Microversion = "2.55"
			result, err := s.GetFlavorByRAM(context.Background(), 4)
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || proof.StatusCode != 199 || proof.Header.Get("X-Backup-Stage") != "utf16-"+stage || len(proof.Body) < 2 || proof.Body[0] != 0xff || proof.Body[1] != 0xfe {
				t.Fatal(result, err, proof)
			}
		})
	}
}
