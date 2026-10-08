package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestDeleteImageRecordIdentityRejectsUnpairedPrivateCurrentID(t *testing.T) {
	for _, test := range []struct {
		name, raw          string
		store, changedView bool
	}{
		{"whole image high surrogate", `"\uD800"`, false, false},
		{"store copy low surrogate image ID", `"\uDC00"`, true, false},
		{"valid public view cannot hide private high followed by nonlow", `"\uD800\u0041"`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			body := `{"id":` + test.raw + `,"name":"passive response"}`
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
					t.Fatal("malformed private identity selected a DELETE target", calls, req.Method, req.URL)
				}
				return taskCoreJSON(req, 203, body), nil
			})
			service := New(client)
			seed, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
			if seed == nil || err != nil || string(seed.Resource.Body["id"]) != test.raw || string(seed.Wire.Body["id"]) != test.raw {
				t.Fatal("passive raw identity was not retained", seed, err)
			}
			if test.changedView {
				seed.Resource.Body["id"] = json.RawMessage(`"�"`)
			}
			options := []ImageRecordDeleteOption{}
			if test.store {
				options = append(options, WithImageRecordDeleteStoreID("store"))
			}
			got, err := service.DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{Record: seed}, options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || string(seed.Wire.Body["id"]) != test.raw || string(seed.Envelope) != body {
				t.Fatal("DELETE must reject private malformed ID before HTTP", got, err, calls, seed)
			}
		})
	}
}

func TestDeleteImageRecordIdentityRejectsUnpairedStoreRecordID(t *testing.T) {
	for _, test := range []struct{ name, raw string }{
		{"high surrogate", `"\uD800"`},
		{"low surrogate", `"\uDC00"`},
		{"high followed by nonlow", `"\uD800\u0041"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				t.Fatal("malformed store identity selected a DELETE target", req.URL)
				return nil, nil
			})
			store := imageRecordDeleteStore(test.raw)
			got, err := New(client).DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{ID: "fixed"}, WithImageRecordDeleteStoreRecord(store))
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || string(store.Resource.Body["id"]) != test.raw {
				t.Fatal("store ID must remain exact and reject before HTTP", got, err, calls, store)
			}
		})
	}
}

func TestDeleteImageRecordIdentityPreservesValidUnicodeTargets(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, test := range []struct{ name, raw, target string }{
			{"surrogate pair", `"\uD83D\uDE80"`, "🚀"},
			{"literal replacement character", `"�"`, "�"},
			{"escaped replacement character", `"\uFFFD"`, "�"},
			{"literal backslash u text", `"\\uD800"`, `\uD800`},
		} {
			name := "image/" + test.name
			if store {
				name = "store/" + test.name
			}
			t.Run(name, func(t *testing.T) {
				calls := 0
				const literalImageID = "image /%2F"
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if !store && calls == 1 {
						if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
							t.Fatal(req.Method, req.URL)
						}
						return taskCoreJSON(req, 203, `{"id":`+test.raw+`}`), nil
					}
					path := "/reverse/glance/v2/images/" + url.PathEscape(test.target)
					if store {
						path = "/reverse/glance/v2/stores/" + url.PathEscape(test.target) + "/" + url.PathEscape(literalImageID)
					}
					wantCalls := 2
					if store {
						wantCalls = 1
					}
					if calls != wantCalls || req.Method != http.MethodDelete || req.URL.EscapedPath() != path || req.URL.RawQuery != "" || req.Body != nil {
						t.Fatal("valid identity text changed during DELETE", calls, req.Method, req.URL)
					}
					return taskCoreJSON(req, 299, "opaque acknowledgement"), nil
				})
				service := New(client)
				input := ImageRecordDeleteRequest{ID: literalImageID}
				options := []ImageRecordDeleteOption{}
				if store {
					options = append(options, WithImageRecordDeleteStoreRecord(imageRecordDeleteStore(test.raw)))
				} else {
					seed, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
					if seed == nil || err != nil {
						t.Fatal(seed, err)
					}
					input = ImageRecordDeleteRequest{Record: seed}
				}
				got, err := service.DeleteImageRecord(context.Background(), input, options...)
				wantCalls, wantImage, wantStore := 2, test.target, ""
				if store {
					wantCalls, wantImage, wantStore = 1, literalImageID, test.target
				}
				if got == nil || got.Acknowledgement == nil || err != nil || calls != wantCalls {
					t.Fatal(got, err, calls)
				}
				th.AssertEquals(t, wantImage, got.Acknowledgement.ImageID)
				th.AssertEquals(t, wantStore, got.Acknowledgement.StoreID)
				th.AssertEquals(t, "opaque acknowledgement", string(got.Acknowledgement.Body))
				if store {
					if got.Record != nil {
						t.Fatal("store delete returned image projection", got.Record)
					}
				} else {
					if got.Record == nil {
						t.Fatal("whole delete lost owned image")
					}
					th.AssertEquals(t, test.target, taskCoreText(t, got.Record.Resource.Body["id"]))
				}
			})
		}
	}
}
