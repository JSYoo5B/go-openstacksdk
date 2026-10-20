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

// Encoding is fixture-only: the shared normalizer owns codec behavior; these
// integration cases prove both actual RAM read stages use it and retain proof.
func flavorRAMCharsetBytes(text, charset string) []byte {
	out := []byte{}
	width, little := 2, true
	switch charset {
	case "ISO-8859-1":
		for _, r := range text {
			out = append(out, byte(r))
		}
		return out
	case "UTF-16":
		out = append(out, 0xff, 0xfe)
	case "UTF-16BE":
		little = false
	case "UTF-32":
		width = 4
		out = append(out, 0xff, 0xfe, 0, 0)
	case "UTF-32LE":
		width = 4
	case "UTF-32BE":
		width, little = 4, false
	}
	for _, r := range text {
		for i := 0; i < width; i++ {
			shift := i * 8
			if !little {
				shift = (width - 1 - i) * 8
			}
			out = append(out, byte(uint32(r)>>shift))
		}
	}
	return out
}

func TestCloudFlavorRAMExplicitCharsetsListAndEnrichment(t *testing.T) {
	for _, charset := range []string{"ISO-8859-1", "UTF-16", "UTF-16LE", "UTF-16BE", "UTF-32", "UTF-32LE", "UTF-32BE"} {
		for _, stage := range []string{"list", "extra", "both"} {
			t.Run(charset+"/"+stage, func(t *testing.T) {
				var physicalList, physicalExtra []byte
				s, c, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
					list := r.URL.Path == flavorRAMListPath
					body := `{"flavors":[{"id":"f","name":"café","ram":4}]}`
					if !list {
						body = `{"extra_specs":{"proof":"café"}}`
					}
					response := backupServerWire(199, body, "charset-proof")
					raw := []byte(body)
					if stage == "both" || list == (stage == "list") {
						raw = flavorRAMCharsetBytes(body, charset)
						response.Header.Set("Content-Type", "application/json;charset="+charset)
					}
					response.Body = io.NopCloser(bytes.NewReader(raw))
					if list {
						physicalList = bytes.Clone(raw)
					} else {
						physicalExtra = bytes.Clone(raw)
					}
					return response
				})
				c.Microversion = "2.55"
				result, err := s.GetFlavorByRAM(context.Background(), 4, compute.WithFlavorRAMInclude("café"))
				if err != nil || flavorRAMID(t, result) != "f" || len(*calls) != 2 {
					t.Fatal(result, err, *calls)
				}
				row := result.Flavor
				if string(row.Resource.Body["name"]) != `"café"` || row.StatusCode != 199 || row.Header.Get("X-Backup-Stage") != "charset-proof" || row.Enrichment == nil || !bytes.Equal(row.Enrichment.Envelope, physicalExtra) || string(row.Enrichment.ExtraSpecs) != `{"proof":"café"}` {
					t.Fatal(row)
				}
				if !bytes.Equal(row.PageEnvelope, physicalList) {
					t.Fatal("physical list page lost", row)
				}
				row.PageEnvelope[0] ^= 1
				if bytes.Equal(row.PageEnvelope, result.Inventory[0].PageEnvelope) {
					t.Fatal("selected and inventory page proof alias")
				}
				row.Enrichment.Envelope[0] ^= 1
				if bytes.Equal(row.Enrichment.Envelope, result.Inventory[0].Enrichment.Envelope) {
					t.Fatal("selected and inventory enrichment alias")
				}
			})
		}
	}
}

func TestCloudFlavorRAMCharsetFailuresKeepPhysicalEvidence(t *testing.T) {
	for _, stage := range []string{"list", "extra"} {
		for _, charset := range []string{"cp1006", "UTF-16", "UTF-32BE", "malformed;="} {
			t.Run(stage+"/"+charset, func(t *testing.T) {
				var physical []byte
				s, c, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
					list := r.URL.Path == flavorRAMListPath
					body := `{"flavors":[{"id":"f","name":"m1","ram":4}]}`
					if !list {
						body = `{"extra_specs":{"proof":"actual"}}`
					}
					response := backupServerWire(199, body, "original-"+stage)
					if list == (stage == "list") {
						physical = []byte(`{bad-json`)
						if charset != "cp1006" && charset != "malformed;=" {
							physical = flavorRAMCharsetBytes(`{bad-json`, charset)
						}
						response.Header.Set("Content-Type", "application/json;charset="+charset)
						response.Body = io.NopCloser(bytes.NewReader(physical))
					}
					return response
				})
				c.Microversion = "2.55"
				result, err := s.GetFlavorByRAM(context.Background(), 4)
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || !bytes.Equal(proof.Body, physical) || proof.StatusCode != 199 || proof.Header.Get("X-Backup-Stage") != "original-"+stage {
					t.Fatal(result, err, proof)
				}
				want := 1
				if stage == "extra" {
					want = 2
					if result == nil || len(result.Inventory) != 1 || result.Inventory[0].Enrichment == nil || !bytes.Equal(result.Inventory[0].Enrichment.Envelope, physical) {
						t.Fatal(result)
					}
				}
				if len(*calls) != want {
					t.Fatal(*calls)
				}
				if charset == "cp1006" && !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestCloudFlavorRAMCharsetLaterPageAndSemanticFailuresKeepActualProof(t *testing.T) {
	for _, mode := range []string{"next-rejected", "next-bad-json", "ram-null"} {
		t.Run(mode, func(t *testing.T) {
			var first, last []byte
			s, c, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
				if r.URL.Query().Get("marker") == "f" {
					last = flavorRAMCharsetBytes(`{bad-json`, "UTF-32BE")
					response := backupServerWire(200, "", "last-page")
					response.Header.Set("Content-Type", "application/json;charset=UTF-32BE")
					if mode == "next-rejected" {
						response.StatusCode = 403
					}
					response.Body = io.NopCloser(bytes.NewReader(last))
					return response
				}
				body := `{"flavors":[{"id":"f","name":"café","ram":null}]}`
				if mode != "ram-null" {
					body = `{"flavors":[{"id":"f","name":"café","ram":4}],"flavors_links":[{"rel":"next","href":"?marker=f"}]}`
				}
				first = flavorRAMCharsetBytes(body, "UTF-16")
				response := backupServerWire(200, "", "first-page")
				response.Header.Set("Content-Type", "application/json;charset=UTF-16")
				response.Body = io.NopCloser(bytes.NewReader(first))
				return response
			})
			c.Microversion = "2.55"
			result, err := s.GetFlavorByRAM(context.Background(), 4, compute.WithFlavorRAMGetExtra(false))
			if err == nil || result == nil || len(result.Inventory) != 1 || !bytes.Equal(result.Inventory[0].PageEnvelope, first) {
				t.Fatal(result, err)
			}
			if mode == "next-rejected" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 403 || !bytes.Equal(native.Body, last) || native.ResponseHeader.Get("X-Backup-Stage") != "last-page" {
					t.Fatal(err, native)
				}
			} else {
				var proof *resource.ResponseError
				want, stage := last, "last-page"
				if mode == "ram-null" {
					want, stage = first, "first-page"
				}
				if !errors.As(err, &proof) || !bytes.Equal(proof.Body, want) || proof.Header.Get("X-Backup-Stage") != stage {
					t.Fatal(err, proof)
				}
			}
			wantCalls := 2
			if mode == "ram-null" {
				wantCalls = 1
			}
			if len(*calls) != wantCalls {
				t.Fatal(*calls)
			}
		})
	}
}

func TestCloudFlavorRAMCharsetDoesNotChangeLegacyReaders(t *testing.T) {
	for _, operation := range []string{"AllFlavors", "FetchExtraSpecs"} {
		t.Run(operation, func(t *testing.T) {
			s, c, _ := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
				body := `{"flavors":[{"id":"f","name":"m1","ram":4}]}`
				if operation == "FetchExtraSpecs" {
					body = `{"extra_specs":{"proof":"actual"}}`
				}
				response := backupServerWire(200, "", "legacy-charset")
				response.Header.Set("Content-Type", "application/json;charset=UTF-16")
				response.Body = io.NopCloser(bytes.NewReader(flavorRAMCharsetBytes(body, "UTF-16")))
				return response
			})
			c.Microversion = "2.55"
			var err error
			if operation == "AllFlavors" {
				_, err = s.AllFlavors(context.Background())
			} else {
				_, err = s.API.Flavors.FetchExtraSpecs(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "f"})
			}
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || !bytes.HasPrefix(proof.Body, []byte{0xff, 0xfe}) || proof.Header.Get("X-Backup-Stage") != "legacy-charset" {
				t.Fatal(err, proof)
			}
		})
	}
}
