package image

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

const imageRecordDownloadABCMD5 = "900150983cd24fb0d6963f7d28e17f72"
const imageRecordDownloadABCSHA256 = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

// Only a borrowed writer callback is added; physical HTTP/reader/Close faults
// use the existing taskCore and downloadCore engines.
type imageRecordDownloadWriterFunc func([]byte) (int, error)

func (f imageRecordDownloadWriterFunc) Write(p []byte) (int, error) { return f(p) }

func imageRecordDownloadResource(t *testing.T, raw string) *resource.RawResource {
	t.Helper()
	var body map[string]json.RawMessage
	th.AssertNoErr(t, json.Unmarshal([]byte(raw), &body))
	return &resource.RawResource{Metadata: resource.Metadata{Body: body}}
}

func TestImageRecordDownloadAllOutputModesAlwaysFetchBeforeBinaryAndOutputWinsStream(t *testing.T) {
	for _, mode := range []string{"memory", "stream", "writer", "writer and stream", "file", "file and stream"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			metadataBody := &taskCoreBody{reader: strings.NewReader(`{"id":"fetched","status":"saving","os_hash_algo":"sha256","os_hash_value":"` + imageRecordDownloadABCSHA256 + `"}`)}
			binaryBody := &taskCoreBody{reader: strings.NewReader("abc")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				th.TestMethod(t, req, http.MethodGet)
				th.TestHeaderUnset(t, req, "X-OpenStack-Image-Size")
				if req.Body != nil || req.URL.RawQuery != "" {
					t.Fatal(req.Body, req.URL)
				}
				if calls == 1 {
					if req.URL.EscapedPath() != "/reverse/glance/v2/images/initial" {
						t.Fatal(req.URL)
					}
					response := taskCoreHTTP(req, 203, metadataBody)
					response.Header.Set("OpenStack-image-import-methods", "metadata, only")
					return response, nil
				}
				if calls != 2 || req.URL.EscapedPath() != "/reverse/glance/v2/images/fetched/file" {
					t.Fatal(calls, req.URL)
				}
				response := taskCoreHTTP(req, 299, binaryBody)
				response.Header.Set("OpenStack-image-import-methods", "binary ignored")
				response.Header.Set("Location", "https://foreign.test/decoy")
				return response, nil
			})
			request := ImageRecordDownloadRequest{ID: "initial"}
			writer := &downloadCoreWriter{}
			filename := filepath.Join(t.TempDir(), "image.bin")
			switch mode {
			case "writer", "writer and stream":
				request.Output = writer
			case "file", "file and stream":
				request.Filename = filename
			}
			options := []ImageRecordDownloadOption{WithImageRecordDownloadChunkSize(2)}
			if strings.Contains(mode, "stream") {
				options = append(options, WithImageRecordDownloadStream(true))
			}
			got, err := New(client).DownloadImageRecord(context.Background(), request, options...)
			if err != nil || got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded == nil || calls != 2 || metadataBody.closes != 1 || got.Metadata.StatusCode != 203 || got.Record.StatusCode != 203 || got.Downloaded.StatusCode != 299 || string(got.Record.Resource.Body["id"]) != `"fetched"` || string(got.Record.Resource.Body["status"]) != `"saving"` || len(got.Record.Resource.Body) != 65 {
				t.Fatal(got, err, calls, metadataBody.closes)
			}
			th.CheckDeepEquals(t, []string{"metadata", " only"}, got.Record.ImportMethods)
			if got.Checksum == nil || got.Checksum.Algorithm != "sha256" || string(got.Checksum.Expected) != `"`+imageRecordDownloadABCSHA256+`"` {
				t.Fatal(got.Checksum)
			}
			if mode == "stream" {
				if got.Downloaded.Stream == nil || got.Downloaded.Body != nil || got.BytesWritten != 0 || got.Checksum.Complete || got.Checksum.Verified || binaryBody.closes != 0 {
					t.Fatal(got, binaryBody.closes)
				}
				actual, readErr := io.ReadAll(got.Downloaded.Stream)
				th.AssertNoErr(t, readErr)
				th.AssertEquals(t, "abc", string(actual))
				th.AssertNoErr(t, got.Downloaded.Stream.Close())
				th.AssertNoErr(t, got.Downloaded.Stream.Close())
				if binaryBody.closes != 1 || got.Checksum.Complete || got.Checksum.Verified || got.BytesWritten != 0 {
					t.Fatal("stream consumption changed returned proof", got, binaryBody.closes)
				}
			} else {
				if got.Downloaded.Stream != nil || got.BytesWritten != 3 || !got.Checksum.Complete || !got.Checksum.Verified || binaryBody.closes != 1 {
					t.Fatal(got, binaryBody.closes)
				}
				if mode == "memory" {
					th.AssertEquals(t, "abc", string(got.Downloaded.Body))
				} else if strings.HasPrefix(mode, "writer") {
					if got.Downloaded.Body != nil || writer.String() != "abc" || writer.closed != 0 || writer.seeks != 0 || writer.readerFromInvoked {
						t.Fatal(got, writer)
					}
					th.CheckDeepEquals(t, []int{2, 1}, writer.writes)
				} else {
					actual, readErr := os.ReadFile(filename)
					th.AssertNoErr(t, readErr)
					th.AssertEquals(t, "abc", string(actual))
					if got.Downloaded.Body != nil {
						t.Fatal(got.Downloaded)
					}
				}
			}
			got.Metadata.Body[0] = '!'
			got.Metadata.Header.Set("X-Task-Proof", "metadata changed")
			got.Downloaded.Header.Set("X-Task-Proof", "binary changed")
			if got.Record.Header.Get("X-Task-Proof") != "actual" || string(got.Record.Envelope)[0] != '{' {
				t.Fatal("phase evidence alias")
			}
		})
	}
}

func TestImageRecordDownloadRawSynchronizedConstructorProjectsAllFieldsWithoutUploadHooks(t *testing.T) {
	body := map[string]json.RawMessage{}
	for _, key := range strings.Fields(imageRecordViewKeys) {
		if key != "location" {
			body[key] = json.RawMessage("null")
		}
	}
	body["id"] = json.RawMessage(`"initial"`)
	body["size"] = json.RawMessage(`"04"`)
	body["is_protected"] = json.RawMessage(`"false"`)
	body["__conflicting_attrs"] = json.RawMessage(`{"id":"hook must not run"}`)
	body["base_path"] = json.RawMessage(`"ordinary metadata"`)
	body["resource_type"] = json.RawMessage(`"ordinary type"`)
	body["filename"] = json.RawMessage(`"/never-open-attribute"`)
	resourceSeed := &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: http.Header{"X-Task-Proof": {"seed decoy"}}, StatusCode: 599}}
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if req.URL.EscapedPath() != "/reverse/glance/v2/images/initial" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 203, "invalid metadata JSON"), nil
		}
		if calls != 2 || req.URL.EscapedPath() != "/reverse/glance/v2/images/initial/file" {
			t.Fatal(calls, req.URL)
		}
		return taskCoreJSON(req, 200, "abc"), nil
	})
	got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{Resource: resourceSeed})
	if err != nil || got == nil || got.Record == nil || got.Record.Wire != nil || calls != 2 || len(got.Record.Resource.Body) != 65 || len(got.Record.bodyState.dirty) != 0 || !reflect.DeepEqual(got.Record.bodyState.current, got.Record.bodyState.original) || string(got.Record.Resource.Body["size"]) != "4" || string(got.Record.bodyState.current["size"]) != `"04"` || string(got.Record.Resource.Body["is_protected"]) != "true" || got.Record.StatusCode != 203 {
		t.Fatal(got, err, calls)
	}
	for _, key := range strings.Fields(imageRecordViewKeys) {
		if _, ok := got.Record.Resource.Body[key]; !ok {
			t.Fatal("declared field absent", key)
		}
	}
	props := imageRecordProperties(t, got.Record)
	for key, expected := range map[string]string{"__conflicting_attrs": `{"id":"hook must not run"}`, "base_path": `"ordinary metadata"`, "resource_type": `"ordinary type"`, "filename": `"/never-open-attribute"`} {
		th.AssertEquals(t, expected, string(props[key]))
	}
	got.Record.bodyState.current["size"][1] = '!'
	got.Record.Resource.Body["name"][0] = '!'
	if string(resourceSeed.Body["size"]) != `"04"` || resourceSeed.Header.Get("X-Task-Proof") != "seed decoy" || resourceSeed.StatusCode != 599 {
		t.Fatal("caller Munch bytes changed", resourceSeed)
	}
}

func TestImageRecordDownloadPrivatePendingRecordIgnoresPublicEditsAndFetchControlsBaseline(t *testing.T) {
	for _, fetch := range []string{"invalid metadata JSON", `{"id":"fetched","name":"remote"}`} {
		t.Run(fetch, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","name":"before","status":"queued","os_hash_algo":"sha256","os_hash_value":"`+imageRecordDownloadABCSHA256+`"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				return taskCoreJSON(req, 203, "invalid update JSON"), nil
			}
			pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"name": "pending"}})
			th.AssertNoErr(t, err)
			reader := strings.NewReader("retained input")
			pending.data = reader
			pending.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
			pending.Wire = nil
			pending.Header.Set("X-Decoy", "public")
			before := cloneImageRecord(pending)
			handler = func(req *http.Request) (*http.Response, error) {
				if calls == 3 {
					if req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
						t.Fatal(req.URL)
					}
					response := taskCoreJSON(req, 203, fetch)
					response.Header.Set("OpenStack-image-import-methods", "fetch only")
					return response, nil
				}
				id := "fixed"
				if json.Valid([]byte(fetch)) {
					id = "fetched"
				}
				if calls != 4 || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+id+"/file" {
					t.Fatal(calls, req.URL)
				}
				response := taskCoreJSON(req, 200, "abc")
				response.Header.Set("OpenStack-image-import-methods", "binary never consume")
				return response, nil
			}
			got, err := service.DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{Record: pending})
			if err != nil || got == nil || got.Record == nil || calls != 4 || got.Record.data != reader || reader.Len() != len("retained input") || !reflect.DeepEqual(before, pending) || got.Checksum == nil || !got.Checksum.Verified {
				t.Fatal(got, err, calls)
			}
			th.CheckDeepEquals(t, []string{"fetch only"}, got.Record.ImportMethods)
			if !json.Valid([]byte(fetch)) {
				if !reflect.DeepEqual(got.Record.bodyState, before.bodyState) || got.Record.Wire != nil {
					t.Fatal(got.Record)
				}
				handler = func(req *http.Request) (*http.Response, error) {
					if calls != 5 || req.Method != http.MethodPatch {
						t.Fatal(calls, req.Method)
					}
					imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/name","value":"pending"}]`)
					return taskCoreJSON(req, 200, `{}`), nil
				}
			} else {
				if len(got.Record.bodyState.dirty) != 0 || string(got.Record.Resource.Body["name"]) != `"remote"` {
					t.Fatal(got.Record)
				}
				handler = func(req *http.Request) (*http.Response, error) {
					t.Fatal("clean fetch replayed pending update", req.URL)
					return nil, nil
				}
			}
			updated, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
			th.AssertNoErr(t, err)
			if updated == nil || updated.data != reader {
				t.Fatal(updated)
			}
		})
	}
}

func TestImageRecordDownloadFetchedIdentityIsOnceEscapedAndNeverUsesResponseLinks(t *testing.T) {
	for _, rawID := range []string{`"a /한:%?\\b"`, `"\ud83d\ude00"`, `"\ufffd"`, `"\\ud800"`} {
		t.Run(rawID, func(t *testing.T) {
			calls := 0
			var id string
			th.AssertNoErr(t, json.Unmarshal([]byte(rawID), &id))
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":`+rawID+`,"self":"https://foreign.test/decoy","file":"https://foreign.test/file","schema":"foreign"}`), nil
				}
				if calls != 2 || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/file" || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal(calls, req.URL, req.Body)
				}
				response := taskCoreJSON(req, 200, "bytes")
				response.Header.Set("Location", "https://foreign.test/redirect")
				return response, nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "initial"})
			if err != nil || got == nil || got.Downloaded == nil || string(got.Downloaded.Body) != "bytes" || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, rawID := range []string{`null`, `false`, `42`, `[]`, `""`, `" "`, `"."`, `".."`, `"line\ncontrol"`, `"\ud800"`, `"\udc00"`} {
		t.Run("invalid/"+rawID, func(t *testing.T) {
			calls := 0
			writer := &downloadCoreWriter{}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 {
					t.Fatal("unsafe fetched id routed", req.URL)
				}
				return taskCoreJSON(req, 203, `{"id":`+rawID+`}`), nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "initial", Output: writer})
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || len(writer.writes) != 0 {
				t.Fatal(got, err, calls, writer)
			}
		})
	}
}

func TestImageRecordDownloadStorePreferencesPreserveRawOrderAndSourceJoinSemantics(t *testing.T) {
	for _, preferences := range [][]string{nil, {}, {"fast", "fast", "slow"}, {"", " a,b ", "bad\nstore", "한&?%"}} {
		t.Run(fmt.Sprint(preferences), func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.URL.RawQuery != "" {
						t.Fatal(req.URL)
					}
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				}
				expected := ""
				if len(preferences) > 0 {
					expected = (url.Values{"prefer": {strings.Join(preferences, ",")}}).Encode()
				}
				if calls != 2 || req.URL.RawQuery != expected || req.URL.Query().Get("prefer") != strings.Join(preferences, ",") {
					t.Fatal(calls, req.URL, expected)
				}
				return taskCoreJSON(req, 200, "bytes"), nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadStorePreferences(preferences...))
			if err != nil || got == nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordDownloadRawHashSelectionOccursAfterGETAndKeepsLiteralExpected(t *testing.T) {
	for _, test := range []struct {
		name, metadata, header, algorithm, expected string
		mismatch                                    bool
	}{
		{"primary wins", `{"hash_algo":"sha256","hash_value":"` + imageRecordDownloadABCSHA256 + `","checksum":"wrong"}`, "wrong", "sha256", `"` + imageRecordDownloadABCSHA256 + `"`, false},
		{"unknown primary falls back", `{"os_hash_algo":"future","os_hash_value":"wrong","checksum":"` + imageRecordDownloadABCMD5 + `"}`, "wrong", "md5", `"` + imageRecordDownloadABCMD5 + `"`, false},
		{"nonstring algorithm Python repr fallback", `{"hash_algo":{"future":true},"hash_value":42,"checksum":"` + imageRecordDownloadABCMD5 + `"}`, "wrong", "md5", `"` + imageRecordDownloadABCMD5 + `"`, false},
		{"falsey primary value", `{"hash_algo":"sha256","hash_value":[],"checksum":"` + imageRecordDownloadABCMD5 + `"}`, "wrong", "md5", `"` + imageRecordDownloadABCMD5 + `"`, false},
		{"nonstring primary expected", `{"hash_algo":"sha256","hash_value":42}`, "", "sha256", "42", true},
		{"nonstring checksum expected", `{"checksum":["raw"]}`, "wrong", "md5", `["raw"]`, true},
		{"falsey checksum", `{"checksum":{}}`, "", "", "", false},
		{"header fallback", `{}`, imageRecordDownloadABCMD5, "md5", `"` + imageRecordDownloadABCMD5 + `"`, false},
		{"no hash", `{}`, "", "", "", false},
		{"literal uppercase mismatch", `{"checksum":"900150983CD24FB0D6963F7D28E17F72"}`, "", "md5", `"900150983CD24FB0D6963F7D28E17F72"`, true},
		{"metadata checksum raw boolean precedence", `{"checksum":true}`, imageRecordDownloadABCMD5, "md5", "true", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			writer := &downloadCoreWriter{}
			body := &taskCoreBody{reader: strings.NewReader("abc")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					var fields map[string]json.RawMessage
					th.AssertNoErr(t, json.Unmarshal([]byte(test.metadata), &fields))
					fields["id"] = json.RawMessage(`"fixed"`)
					raw, _ := json.Marshal(fields)
					return taskCoreJSON(req, 203, string(raw)), nil
				}
				if calls != 2 {
					t.Fatal("hash error preempted binaryGET/replayed", calls)
				}
				response := taskCoreHTTP(req, 200, body)
				if test.header != "" {
					response.Header.Set("Content-MD5", test.header)
				}
				return response, nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed", Output: writer})
			if got == nil || got.Record == nil || got.Downloaded == nil || calls != 2 || writer.String() != "abc" || got.BytesWritten != 3 || body.closes != 1 {
				t.Fatal(got, err, calls, writer, body.closes)
			}
			if test.algorithm == "" {
				if err != nil || got.Checksum != nil {
					t.Fatal(got, err)
				}
				return
			}
			if got.Checksum == nil || got.Checksum.Algorithm != test.algorithm || string(got.Checksum.Expected) != test.expected || !got.Checksum.Complete || got.Checksum.Verified == test.mismatch {
				t.Fatal(got.Checksum, err)
			}
			if test.mismatch {
				var mismatch *ImageRecordDownloadChecksumMismatchError
				if !errors.Is(err, ErrChecksumMismatch) || !errors.As(err, &mismatch) || string(mismatch.Expected) != test.expected || mismatch.Actual != got.Checksum.Actual || mismatch.Algorithm != test.algorithm {
					t.Fatal(got, err, mismatch)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadSupportedHashRegistryUsesIndependentDigestVectors(t *testing.T) {
	for _, test := range []struct{ algorithm, digest string }{
		{"md5", "900150983cd24fb0d6963f7d28e17f72"},
		{"sha1", "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{"sha224", "23097d223405d8228642a477bda255b32aadbce4bda0b3f7e36c9da7"},
		{"sha256", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"sha384", "cb00753f45a35e8bb5a03d699ac65007272c32ab0eded1631a8b605a43ff5bed8086072ba1e7cc2358baeca134c825a7"},
		{"sha512", "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f"},
		{"sha512_224", "4634270f707b6a54daae7530460842e20e37ed265ceee9a43e8924aa"},
		{"sha512_256", "53048e2681941ef99b2e29b76b4c7dabe4c2d0c634fc6d46e0e2f13107e7af23"},
		{"sha3_224", "e642824c3f8cf24ad09234ee7d3c766fc9a3a5168d0c94ad73b46fdf"},
		{"sha3_256", "3a985da74fe225b2045c172d6bd390bd855f086e3e9d525b46bfe24511431532"},
		{"sha3_384", "ec01498288516fc926459f58e2c6ad8df9b473cb0fc08c2596da7cf0e49be4b298d88cea927ac7f539f1edf228376d25"},
		{"sha3_512", "b751850b1a57168a5693cd924b6b096e08f621827444f70d884f5d0240d2712e10e116e9192af3c91a7ec57647e3934057340b4cf408d5a56592f8274eec53f0"},
		{"blake2b", "ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d17d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923"},
		{"blake2s", "508c5e8c327c14e2e1a72ba34eeb452f37458b209ed63a294d999b4c86675982"},
		{"SHA256", imageRecordDownloadABCSHA256},
		{"SHA3-256", "3a985da74fe225b2045c172d6bd390bd855f086e3e9d525b46bfe24511431532"},
		{"BLAKE2b512", "ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d17d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923"},
	} {
		t.Run(test.algorithm, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","hash_algo":"`+test.algorithm+`","hash_value":"`+test.digest+`","checksum":"wrong fallback"}`), nil
				}
				return taskCoreJSON(req, 200, "abc"), nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"})
			if err != nil || got == nil || got.Checksum == nil || !got.Checksum.Complete || !got.Checksum.Verified || got.Checksum.Actual != test.digest || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordDownloadCustomPrimaryFactoryRunsOnceAfterBinaryAndKeepsFallbackOwned(t *testing.T) {
	for _, mode := range []string{"extension", "unsupported fallback", "factory error", "nil hash", "source", "cancel", "outer", "verification disabled"} {
		t.Run(mode, func(t *testing.T) {
			calls, factories := 0, 0
			marker := errors.New("download hash factory cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			bodyReader := strings.NewReader("abc")
			body := &taskCoreBody{reader: bodyReader}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","hash_algo":"extension","hash_value":"`+imageRecordDownloadABCMD5+`","checksum":"`+imageRecordDownloadABCMD5+`"}`), nil
				}
				return taskCoreHTTP(req, 200, body), nil
			})
			factory := func(algorithm string) (hash.Hash, error) {
				factories++
				if calls != 2 || algorithm != "extension" {
					t.Fatal("factory timing or fallback doublecallback", calls, algorithm)
				}
				switch mode {
				case "unsupported fallback":
					return nil, ErrImageRecordDownloadHashUnsupported
				case "factory error":
					return nil, marker
				case "nil hash":
					return nil, nil
				case "source":
					client.Endpoint = "https://foreign.test/"
				case "cancel":
					cancel(marker)
				case "outer":
					outerBad = true
				}
				return md5.New(), nil
			}
			options := []ImageRecordDownloadOption{WithImageRecordDownloadHashFactory(factory)}
			if mode == "verification disabled" {
				options = append(options, WithImageRecordDownloadChecksumVerification(false))
			}
			got, err := New(client).DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed"}, options...)
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded == nil || calls != 2 || body.closes != 1 {
				t.Fatal(got, err, calls, body.closes)
			}
			if mode == "verification disabled" {
				if err != nil || factories != 0 || got.Checksum != nil || string(got.Downloaded.Body) != "abc" {
					t.Fatal(got, err, factories)
				}
				return
			}
			if factories != 1 {
				t.Fatal(factories)
			}
			if mode == "extension" || mode == "unsupported fallback" {
				if err != nil || got.Checksum == nil || !got.Checksum.Verified || string(got.Downloaded.Body) != "abc" {
					t.Fatal(got, err)
				}
				expected := "extension"
				if mode == "unsupported fallback" {
					expected = "md5"
				}
				th.AssertEquals(t, expected, got.Checksum.Algorithm)
				return
			}
			if err == nil || bodyReader.Len() != 3 || got.BytesWritten != 0 {
				t.Fatal(got, err, bodyReader.Len())
			}
			if mode == "source" || mode == "nil hash" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadSHAKESelectsStreamButConsumedModeRequiresDigestLength(t *testing.T) {
	for _, algorithm := range []string{"shake_128", "shake_256"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%v", algorithm, stream), func(t *testing.T) {
				calls := 0
				body := &taskCoreBody{reader: strings.NewReader("abc")}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return taskCoreJSON(req, 203, `{"id":"fixed","hash_algo":"`+algorithm+`","hash_value":"expected"}`), nil
					}
					return taskCoreHTTP(req, 200, body), nil
				})
				got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadStream(stream))
				if got == nil || got.Checksum == nil || got.Downloaded == nil || calls != 2 || got.Checksum.Algorithm != algorithm || got.Checksum.Actual != "" || got.Checksum.Verified {
					t.Fatal(got, err, calls)
				}
				if stream {
					if err != nil || got.Checksum.Complete || got.Downloaded.Stream == nil || body.closes != 0 {
						t.Fatal(got, err, body.closes)
					}
					actual, readErr := io.ReadAll(got.Downloaded.Stream)
					th.AssertNoErr(t, readErr)
					th.AssertEquals(t, "abc", string(actual))
					th.AssertNoErr(t, got.Downloaded.Stream.Close())
				} else if !errors.Is(err, ErrImageRecordDownloadHashLengthRequired) || !got.Checksum.Complete || string(got.Downloaded.Body) != "abc" || got.BytesWritten != 3 || body.closes != 1 {
					t.Fatal(got, err, body.closes)
				}
			})
		}
	}
}

func TestImageRecordDownloadStreamMD5CompatibilityKeepsActualHeaderAndRawExpectedSeparate(t *testing.T) {
	for _, expected := range []string{`"` + imageRecordDownloadABCMD5 + `"`, `true`, `["raw",null]`} {
		t.Run(expected, func(t *testing.T) {
			calls := 0
			rawReader := strings.NewReader("abc")
			body := &taskCoreBody{reader: rawReader}
			var actualHeader http.Header
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","checksum":`+expected+`}`), nil
				}
				response := taskCoreHTTP(req, 200, body)
				response.Header.Set("Content-MD5", "actual header decoy")
				actualHeader = response.Header
				return response, nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadStream(true))
			if err != nil || got == nil || got.Downloaded == nil || got.Downloaded.Stream == nil || got.Checksum == nil || calls != 2 || body.closes != 0 || rawReader.Len() != 3 || got.Checksum.Complete || got.Checksum.Verified || string(got.Downloaded.ContentMD5) != expected || got.Downloaded.Header.Get("Content-MD5") != "actual header decoy" {
				t.Fatal(got, err, calls, body.closes, rawReader.Len())
			}
			compat := map[string]string{`true`: "True", `["raw",null]`: "['raw', None]"}[expected]
			if compat == "" {
				compat = imageRecordDownloadABCMD5
			}
			th.AssertEquals(t, compat, got.Downloaded.CompatibilityHeader.Get("Content-MD5"))
			got.Downloaded.CompatibilityHeader.Set("Content-MD5", "changed")
			got.Downloaded.ContentMD5[0] = '!'
			if string(got.Checksum.Expected) != expected || got.Downloaded.Header.Get("Content-MD5") != "actual header decoy" || actualHeader.Get("Content-MD5") != "actual header decoy" {
				t.Fatal("compatibility projection aliases actual/raw evidence")
			}
			th.AssertNoErr(t, got.Downloaded.Stream.Close())
			th.AssertNoErr(t, got.Downloaded.Stream.Close())
			if body.closes != 1 || rawReader.Len() != 3 {
				t.Fatal(body.closes, rawReader.Len())
			}
		})
	}
}

func TestImageRecordDownloadPreflightRejectsAmbiguousSelectorsAndUnsafeLocalInputs(t *testing.T) {
	cases := []struct {
		name    string
		request ImageRecordDownloadRequest
		option  ImageRecordDownloadOption
	}{
		{"missing selector", ImageRecordDownloadRequest{}, nil}, {"ID and Record", ImageRecordDownloadRequest{ID: "fixed", Record: &ImageRecord{}}, nil},
		{"ID and Resource", ImageRecordDownloadRequest{ID: "fixed", Resource: imageRecordDownloadResource(t, `{"id":"fixed"}`)}, nil},
		{"Record and Resource", ImageRecordDownloadRequest{Record: &ImageRecord{}, Resource: imageRecordDownloadResource(t, `{"id":"fixed"}`)}, nil},
		{"handcrafted Record", ImageRecordDownloadRequest{Record: &ImageRecord{Resource: imageRecordDownloadResource(t, `{"id":"fixed"}`)}}, nil},
		{"missing raw identity", ImageRecordDownloadRequest{Resource: imageRecordDownloadResource(t, `{"name":"not an ID"}`)}, nil},
		{"null raw identity", ImageRecordDownloadRequest{Resource: imageRecordDownloadResource(t, `{"id":null}`)}, nil},
		{"unsafe literal", ImageRecordDownloadRequest{ID: ".."}, nil}, {"literal UTF8", ImageRecordDownloadRequest{ID: string([]byte{255})}, nil},
		{"both outputs", ImageRecordDownloadRequest{ID: "fixed", Output: &bytes.Buffer{}, Filename: "out"}, nil},
		{"filename UTF8", ImageRecordDownloadRequest{ID: "fixed", Filename: string([]byte{255})}, nil},
		{"descriptor overflow", ImageRecordDownloadRequest{Resource: imageRecordDownloadResource(t, `{"id":"fixed","size":1e9999}`)}, nil},
		{"boolstr descriptor", ImageRecordDownloadRequest{Resource: imageRecordDownloadResource(t, `{"id":"fixed","is_hw_vif_multiqueue_enabled":"invalid"}`)}, nil},
		{"zero chunk", ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadChunkSize(0)},
		{"negative chunk", ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadChunkSize(-1)},
		{"large chunk", ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadChunkSize(64*1024*1024 + 1)},
		{"preferences UTF8", ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadStorePreferences(string([]byte{255}))},
		{"owned auth header", ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadHeader("X-Auth-Token", "spoof")},
		{"owned media header", ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadHeader("Accept", "spoof")},
	}
	var typedNil *downloadCoreWriter
	cases = append(cases, struct {
		name    string
		request ImageRecordDownloadRequest
		option  ImageRecordDownloadOption
	}{"typed nil output", ImageRecordDownloadRequest{ID: "fixed", Output: typedNil}, nil})
	for _, key := range []string{"self", "connection", "_synchronized", "microversion"} {
		cases = append(cases, struct {
			name    string
			request ImageRecordDownloadRequest
			option  ImageRecordDownloadOption
		}{"constructor " + key, ImageRecordDownloadRequest{Resource: imageRecordDownloadResource(t, `{"id":"fixed","`+key+`":null}`)}, nil})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("preflight failure dispatched")
				return nil, nil
			})
			options := []ImageRecordDownloadOption{}
			if test.option != nil {
				options = append(options, test.option)
			}
			got, err := New(client).DownloadImageRecord(context.Background(), test.request, options...)
			if got != nil || err == nil || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("nil option", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
		got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"}, nil)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestImageRecordDownloadMetadataFailuresKeepActualEvidenceAndNeverOpenOutput(t *testing.T) {
	for _, mode := range []string{"native404", "native503", "read", "close", "cancel", "source restored on Close", "null", "array", "projection", "invalidUTF8"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			marker := errors.New("download metadata phase")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			raw := `{"id":"fixed"}`
			code := 203
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			var client *gophercloud.ServiceClient
			switch mode {
			case "native404":
				code = 404
				raw = "native metadata rejected"
			case "native503":
				code = 503
				raw = "native metadata rejected"
			case "read":
				body.reader = &taskCoreReader{body: raw, err: marker}
			case "close":
				body.closeErr = marker
			case "cancel":
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: func() { cancel(marker) }}
			case "source restored on Close":
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: func() { client.Endpoint = "https://foreign.test/" }}
			case "null":
				raw = `null`
			case "array":
				raw = `[]`
			case "projection":
				raw = `{"id":"fixed","size":1e9999}`
			case "invalidUTF8":
				raw = "{\"id\":\"fixed\",\"name\":\"\xff\"}"
			}
			if mode != "read" && mode != "cancel" && mode != "source restored on Close" {
				body.reader = strings.NewReader(raw)
			}
			var selected io.ReadCloser = body
			if mode == "source restored on Close" {
				selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/" }}
			}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 {
					t.Fatal("metadata failure downloaded", req.URL)
				}
				return taskCoreHTTP(req, code, selected), nil
			})
			filename := filepath.Join(t.TempDir(), "must-not-open")
			got, err := New(client).DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed", Filename: filename})
			if err == nil || calls != 1 || body.closes != 1 {
				t.Fatal(got, err, calls, body.closes)
			}
			if _, statErr := os.Stat(filename); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatal("metadata failure opened output", statErr)
			}
			if strings.HasPrefix(mode, "native") {
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != raw {
					t.Fatal(got, err, native)
				}
				return
			}
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded != nil || got.Metadata.StatusCode != 203 || string(got.Metadata.Body) != raw {
				t.Fatal(got, err)
			}
			taskCoreProof(t, err, 203, raw)
			if mode == "read" || mode == "close" || mode == "cancel" {
				if !errors.Is(err, marker) {
					t.Fatal(err)
				}
			} else if strings.HasPrefix(mode, "source") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadAcceptedStatusRangeAndEmptyBodyUseWholeChecksumProfile(t *testing.T) {
	for _, code := range []int{200, 204, 206, 299, 300, 304, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, redirects := 0, 0
			payload := "abc"
			if code == 204 {
				payload = ""
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				th.TestHeader(t, req, "Range", "bytes=0-2")
				th.TestHeader(t, req, "If-Range", "etag")
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				}
				if calls != 2 {
					t.Fatal(calls)
				}
				response := taskCoreJSON(req, code, payload)
				response.Header.Set("Content-Range", "bytes 0-2/100")
				response.Header.Set("Location", "https://foreign.test/ignored")
				return response, nil
			})
			client.MoreHeaders = map[string]string{"Range": "bytes=0-2"}
			client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects++; return nil }
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadHeader("If-Range", "etag"))
			if err != nil || got == nil || got.Downloaded == nil || got.Downloaded.StatusCode != code || string(got.Downloaded.Body) != payload || got.BytesWritten != int64(len(payload)) || redirects != 0 || calls != 2 || got.Downloaded.Header.Get("Content-Range") == "" {
				t.Fatal(got, err, calls, redirects)
			}
		})
	}
	for _, expected := range []string{"d41d8cd98f00b204e9800998ecf8427e", imageRecordDownloadABCMD5} {
		t.Run("204/checksum="+expected, func(t *testing.T) {
			calls := 0
			body := &taskCoreBody{reader: strings.NewReader("")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","checksum":"`+expected+`"}`), nil
				}
				return taskCoreHTTP(req, 204, body), nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"})
			match := expected == "d41d8cd98f00b204e9800998ecf8427e"
			if got == nil || got.Checksum == nil || !got.Checksum.Complete || got.Checksum.Verified != match || got.Checksum.Actual != "d41d8cd98f00b204e9800998ecf8427e" || got.BytesWritten != 0 || body.closes != 1 || calls != 2 {
				t.Fatal(got, err, calls)
			}
			if match {
				th.AssertNoErr(t, err)
			} else if !errors.Is(err, ErrChecksumMismatch) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadMemoryWriterFaultsRetainPartialProgressAndCompletedCloseProof(t *testing.T) {
	readCause, writeCause, closeCause := errors.New("binary read"), errors.New("borrowed write"), errors.New("binary Close")
	for _, mode := range []string{"memory read", "writer read and write", "writer short write", "memory Close only", "writer Close only", "bad read count", "no progress"} {
		t.Run(mode, func(t *testing.T) {
			calls, reads := 0, 0
			body := &taskCoreBody{reader: strings.NewReader("abc"), closeErr: closeCause}
			var output io.Writer
			expectedBytes := int64(3)
			complete := false
			switch mode {
			case "memory read":
				body.reader = &downloadCoreReadFailure{data: "abc", cause: readCause}
			case "writer read and write":
				body.reader = &downloadCoreReadFailure{data: "abc", cause: readCause}
				output = downloadCoreWriteFailure{count: 2, cause: writeCause}
				expectedBytes = 2
			case "writer short write":
				body.reader = &downloadCoreReadFailure{data: "abc", cause: readCause}
				output = downloadCoreWriteFailure{count: 2}
				expectedBytes = 2
			case "memory Close only":
				complete = true
			case "writer Close only":
				output = &downloadCoreWriter{}
				complete = true
			case "bad read count":
				expectedBytes = 0
				body.reader = deleteCoreReader(func(p []byte) (int, error) { return len(p) + 1, nil })
			case "no progress":
				expectedBytes = 0
				body.reader = deleteCoreReader(func([]byte) (int, error) { reads++; return 0, nil })
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","checksum":"`+imageRecordDownloadABCMD5+`"}`), nil
				}
				return taskCoreHTTP(req, 200, body), nil
			})
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed", Output: output}, WithImageRecordDownloadChunkSize(64))
			if got == nil || got.Downloaded == nil || got.Record == nil || got.Metadata == nil || got.BytesWritten != expectedBytes || got.Checksum == nil || got.Checksum.Complete != complete || got.Checksum.Verified != complete || err == nil || !errors.Is(err, closeCause) || calls != 2 || body.closes != 1 {
				t.Fatal(got, err, calls, body.closes)
			}
			if mode == "memory read" {
				if string(got.Downloaded.Body) != "abc" || !errors.Is(err, readCause) {
					t.Fatal(got, err)
				}
			}
			if strings.HasPrefix(mode, "writer read") && (!errors.Is(err, readCause) || !errors.Is(err, writeCause)) {
				t.Fatal(err)
			}
			if mode == "writer short write" && (!errors.Is(err, readCause) || !errors.Is(err, io.ErrShortWrite)) {
				t.Fatal(err)
			}
			if mode == "no progress" && (!errors.Is(err, io.ErrNoProgress) || reads != 100) {
				t.Fatal(err, reads)
			}
			if complete && got.Checksum.Actual != imageRecordDownloadABCMD5 || !complete && got.Checksum.Actual != "" {
				t.Fatal(got.Checksum)
			}
		})
	}
	t.Run("default 1MiB bounded writer preserves append cursor", func(t *testing.T) {
		calls := 0
		payload := strings.Repeat("x", 1024*1024+3)
		writer := &downloadCoreWriter{}
		_, _ = writer.Buffer.WriteString("prefix-")
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
			}
			return taskCoreJSON(req, 200, payload), nil
		})
		got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed", Output: writer})
		if err != nil || got == nil || got.BytesWritten != int64(len(payload)) || writer.String() != "prefix-"+payload || writer.closed != 0 || writer.seeks != 0 || writer.readerFromInvoked {
			t.Fatal(got, err, writer)
		}
		th.CheckDeepEquals(t, []int{1024 * 1024, 3}, writer.writes)
	})
}

func TestImageRecordDownloadOwnedOutputPathOpensAfterHashSelectionAndLeavesPartialFiles(t *testing.T) {
	for _, mode := range []string{"truncate success", "mismatch persists", "partial read persists", "missing directory", "directory", "factory error before open"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			marker := errors.New("output path phase")
			dir := t.TempDir()
			filename := filepath.Join(dir, "image.bin")
			th.AssertNoErr(t, os.WriteFile(filename, []byte("previous-long-content"), 0600))
			if mode == "missing directory" {
				filename = filepath.Join(dir, "missing", "image.bin")
			}
			if mode == "directory" {
				filename = dir
			}
			body := &taskCoreBody{reader: strings.NewReader("abc")}
			if mode == "partial read persists" {
				body.reader = &downloadCoreReadFailure{data: "abc", cause: marker}
			}
			digest := imageRecordDownloadABCMD5
			if mode == "mismatch persists" {
				digest = "wrong"
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","hash_algo":"extension","hash_value":"`+digest+`"}`), nil
				}
				if calls != 2 {
					t.Fatal(calls)
				}
				if mode == "truncate success" || mode == "mismatch persists" || mode == "partial read persists" || mode == "factory error before open" {
					old, err := os.ReadFile(filename)
					th.AssertNoErr(t, err)
					th.AssertEquals(t, "previous-long-content", string(old))
				}
				return taskCoreHTTP(req, 200, body), nil
			})
			factory := func(string) (hash.Hash, error) {
				if calls != 2 {
					t.Fatal("file opened beforebinary/hash", calls)
				}
				if mode == "factory error before open" {
					return nil, marker
				}
				return md5.New(), nil
			}
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed", Filename: filename}, WithImageRecordDownloadHashFactory(factory), WithImageRecordDownloadChunkSize(64))
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded == nil || got.Downloaded.Body != nil || got.Downloaded.Stream != nil || calls != 2 || body.closes != 1 {
				t.Fatal(got, err, calls, body.closes)
			}
			switch mode {
			case "truncate success":
				th.AssertNoErr(t, err)
				actual, readErr := os.ReadFile(filename)
				th.AssertNoErr(t, readErr)
				th.AssertEquals(t, "abc", string(actual))
				if got.BytesWritten != 3 || got.Checksum == nil || !got.Checksum.Verified {
					t.Fatal(got)
				}
			case "mismatch persists", "partial read persists":
				actual, readErr := os.ReadFile(filename)
				th.AssertNoErr(t, readErr)
				th.AssertEquals(t, "abc", string(actual))
				if got.BytesWritten != 3 || err == nil {
					t.Fatal(got, err)
				}
				if mode == "mismatch persists" {
					if !errors.Is(err, ErrChecksumMismatch) || !got.Checksum.Complete {
						t.Fatal(got, err)
					}
				} else if !errors.Is(err, marker) || got.Checksum.Complete {
					t.Fatal(got, err)
				}
			case "missing directory", "directory":
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) || got.BytesWritten != 0 {
					t.Fatal(got, err)
				}
			case "factory error before open":
				actual, readErr := os.ReadFile(filename)
				th.AssertNoErr(t, readErr)
				th.AssertEquals(t, "previous-long-content", string(actual))
				if !errors.Is(err, marker) || got.BytesWritten != 0 {
					t.Fatal(got, err)
				}
			}
		})
	}
}

func TestImageRecordDownloadCapturesConstructorOptionsLocationAndHeadersBeforeCallbacks(t *testing.T) {
	calls, callbacks, locations, factories := 0, 0, 0, 0
	cloud := "captured cloud"
	chunk := 2
	verify := true
	headers := map[string]string{"X-Note": "captured"}
	preferences := []string{"first", "first", " a,b "}
	seed := imageRecordDownloadResource(t, `{"id":"fixed","name":"before","vendor":["before"]}`)
	option := WithImageRecordDownloadOpts(ImageRecordDownloadOpts{ChunkSize: &chunk, VerifyChecksum: &verify, Headers: headers, StorePreferences: preferences, HashFactory: func(algorithm string) (hash.Hash, error) {
		factories++
		th.AssertEquals(t, "sha256", algorithm)
		return sha256.New(), nil
	}})
	chunk = 99
	verify = false
	headers["X-Note"] = "external changed"
	preferences[0] = "external changed"
	var retained *ImageRecordDownloadOpts
	options := []ImageRecordDownloadOption{option, func(config *ImageRecordDownloadOpts) error {
		callbacks++
		retained = config
		return WithImageRecordDownloadHeader("X-Added", "yes")(config)
	}}
	writer := &downloadCoreWriter{}
	var client *gophercloud.ServiceClient
	client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Source") != "captured source" || req.Header.Get("X-Note") != "captured" || req.Header.Get("X-Added") != "yes" || req.Header.Get("Content-Type") != "configured type" || req.Header.Get("Accept") != "configured accept" {
			t.Fatal(req.Header)
		}
		if calls == 1 {
			th.AssertEquals(t, "live first", req.Header.Get("X-Auth-Token"))
			if req.URL.RawQuery != "" || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
				t.Fatal(req.URL)
			}
			*retained.ChunkSize = 999
			*retained.VerifyChecksum = false
			retained.StorePreferences[0] = "retained changed"
			retained.Headers["X-Note"] = "retained changed"
			client.MoreHeaders["X-Source"] = "later"
			client.SetToken("live second")
			cloud = "later cloud"
			return taskCoreJSON(req, 203, `{"hash_algo":"sha256","hash_value":"`+imageRecordDownloadABCSHA256+`"}`), nil
		}
		if calls != 2 || req.URL.RawQuery != (url.Values{"prefer": {"first,first, a,b "}}).Encode() || req.Header.Get("X-Auth-Token") != "live second" {
			t.Fatal(calls, req.URL, req.Header)
		}
		return taskCoreJSON(req, 200, "abc"), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured source", "Content-Type": "configured type", "Accept": "configured accept"}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		seed.Body["id"] = json.RawMessage(`"external decoy"`)
		seed.Body["vendor"] = json.RawMessage(`"external decoy"`)
		seed.Body["name"] = json.RawMessage(`"external decoy"`)
		options[1] = func(*ImageRecordDownloadOpts) error { t.Fatal("uncaptured option slice"); return nil }
		client.SetToken("live first")
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
	got, err := service.DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{Resource: seed, Output: writer}, options...)
	if err != nil || got == nil || calls != 2 || callbacks != 1 || locations != 1 || factories != 1 || got.Checksum == nil || !got.Checksum.Verified || writer.String() != "abc" || string(got.Record.Resource.Body["name"]) != `"before"` || len(imageRecordProperties(t, got.Record)) != 0 {
		t.Fatal(got, err, calls, callbacks, locations, factories, writer)
	}
	th.CheckDeepEquals(t, []int{2, 1}, writer.writes)
	var location resource.CloudLocation
	th.AssertNoErr(t, json.Unmarshal(got.Record.Resource.Body["location"], &location))
	th.AssertEquals(t, "captured cloud", *location.Cloud)
}

func TestImageRecordDownloadOptionMergeReplacementReuseAndStickyCallbackGuards(t *testing.T) {
	for _, mode := range []string{"header merge", "full replace", "reusable helpers", "cancel", "source", "binding", "outer", "callback cause"} {
		t.Run(mode, func(t *testing.T) {
			calls, callbacks, later := 0, 0, 0
			marker := errors.New("download option guard")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			writer := &downloadCoreWriter{}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("X-Final") != "new" {
					t.Fatal(req.Header)
				}
				discarded := "old"
				if mode == "full replace" {
					discarded = ""
				}
				th.AssertEquals(t, discarded, req.Header.Get("X-Discarded"))
				if calls%2 == 1 {
					if req.URL.RawQuery != "" {
						t.Fatal(req.URL)
					}
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				}
				expected := ""
				if mode != "full replace" {
					expected = (url.Values{"prefer": {"before,before"}}).Encode()
				}
				if req.URL.RawQuery != expected {
					t.Fatal(req.URL)
				}
				return taskCoreJSON(req, 200, "abc"), nil
			})
			service := New(client)
			opts := []ImageRecordDownloadOption{WithImageRecordDownloadStream(true), WithImageRecordDownloadChunkSize(2), WithImageRecordDownloadStorePreferences("before", "before"), WithImageRecordDownloadHeader("X-Discarded", "old")}
			switch mode {
			case "header merge", "reusable helpers":
				opts = append(opts, WithImageRecordDownloadHeaders(map[string]string{"X-Final": "new"}))
			case "full replace":
				opts = append(opts, WithImageRecordDownloadOpts(ImageRecordDownloadOpts{Headers: map[string]string{"X-Final": "new"}}))
			default:
				opts = append(opts, func(*ImageRecordDownloadOpts) error {
					callbacks++
					switch mode {
					case "cancel":
						cancel(marker)
					case "source":
						client.Endpoint = "https://foreign.test/"
					case "binding":
						service.API = nil
					case "outer":
						outerBad = true
					case "callback cause":
						return marker
					}
					return nil
				}, func(*ImageRecordDownloadOpts) error { later++; return nil })
			}
			got, err := service.DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed", Output: writer}, opts...)
			if mode == "header merge" || mode == "full replace" || mode == "reusable helpers" {
				if err != nil || got == nil || calls != 2 || writer.String() != "abc" {
					t.Fatal(got, err, calls, writer)
				}
				expected := []int{2, 1}
				if mode == "full replace" {
					expected = []int{3}
				}
				th.CheckDeepEquals(t, expected, writer.writes)
				if mode == "reusable helpers" {
					writer = &downloadCoreWriter{}
					got, err = service.DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed", Output: writer}, opts...)
					if err != nil || got == nil || calls != 4 || writer.String() != "abc" {
						t.Fatal(got, err, calls, writer)
					}
					th.CheckDeepEquals(t, []int{2, 1}, writer.writes)
				}
				return
			}
			if got != nil || err == nil || calls != 0 || callbacks != 1 || later != 0 {
				t.Fatal(got, err, calls, callbacks, later)
			}
			if mode == "source" || mode == "binding" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadNativeRejectionsKeepPhaseEvidenceAndNeverConsumeOutput(t *testing.T) {
	for _, phase := range []string{"metadata", "binary"} {
		for _, code := range []int{400, 401, 404, 418, 503, 599} {
			t.Run(fmt.Sprintf("%s/%d", phase, code), func(t *testing.T) {
				calls := 0
				body := &taskCoreBody{reader: strings.NewReader("native download rejection")}
				writer := &downloadCoreWriter{}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if phase == "binary" && calls == 1 {
						return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
					}
					return taskCoreHTTP(req, code, body), nil
				})
				got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed", Output: writer})
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native download rejection" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || body.closes != 1 || len(writer.writes) != 0 {
					t.Fatal(got, err, native, body.closes, writer)
				}
				if phase == "metadata" {
					if got != nil || calls != 1 {
						t.Fatal(got, calls)
					}
				} else if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded != nil || calls != 2 {
					t.Fatal(got, calls)
				}
			})
		}
	}
}

func TestImageRecordDownloadNativeRetryAndReauthenticationPreserveFixedGETOwnership(t *testing.T) {
	for _, phase := range []string{"metadata", "binary"} {
		for _, mode := range []string{"ordinary retry", "reauth", "expanded OkCodes", "changed body", "KeepResponseBody", "source mutation", "hook cause"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				calls, retries, reauth := 0, 0, 0
				marker := errors.New("download native retry")
				first := &taskCoreBody{reader: strings.NewReader("initial rejection")}
				second := &taskCoreBody{reader: strings.NewReader("actual rejection")}
				attempts := 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					th.TestMethod(t, req, http.MethodGet)
					if req.Body != nil {
						t.Fatal("GET acquired callback body", req.URL)
					}
					isBinary := strings.HasSuffix(req.URL.Path, "/file")
					if phase == "binary" && !isBinary {
						return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
					}
					if phase == "metadata" && isBinary {
						return taskCoreJSON(req, 200, "abc"), nil
					}
					attempts++
					expectedQuery := ""
					if isBinary {
						expectedQuery = (url.Values{"prefer": {" a,b ,first"}}).Encode()
					}
					if req.URL.RawQuery != expectedQuery {
						t.Fatal(req.URL)
					}
					if attempts == 1 {
						code := 503
						if mode == "reauth" {
							code = 401
						}
						return taskCoreHTTP(req, code, first), nil
					}
					if attempts != 2 || req.Header.Get("X-Auth-Token") != "retry token" {
						t.Fatal(attempts, req.Header)
					}
					if mode != "reauth" {
						th.AssertEquals(t, "ordinary", req.Header.Get("X-Retry"))
					}
					if mode == "expanded OkCodes" {
						return taskCoreHTTP(req, 418, second), nil
					}
					if isBinary {
						return taskCoreJSON(req, 203, "abc"), nil
					}
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				})
				client.RetryFunc = func(_ context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
					retries++
					if count > 1 || mode == "reauth" {
						return original
					}
					if method != http.MethodGet || strings.HasSuffix(target, "/file?"+(url.Values{"prefer": {" a,b ,first"}}).Encode()) != (phase == "binary") {
						t.Fatal(method, target)
					}
					client.SetToken("retry token")
					opts.MoreHeaders = map[string]string{"X-Retry": "ordinary"}
					switch mode {
					case "expanded OkCodes":
						opts.OkCodes = append(opts.OkCodes, 418)
					case "changed body":
						opts.JSONBody = map[string]any{"spoof": true}
					case "KeepResponseBody":
						opts.KeepResponseBody = false
					case "source mutation":
						client.Endpoint = "https://foreign.test/"
					case "hook cause":
						return errors.Join(original, marker)
					}
					return nil
				}
				if mode == "reauth" {
					client.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("retry token"); return nil }
				}
				got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadStorePreferences(" a,b ", "first"))
				if mode == "ordinary retry" || mode == "reauth" {
					if err != nil || got == nil || got.Downloaded == nil || string(got.Downloaded.Body) != "abc" || calls != 3 || attempts != 2 || first.closes != 1 {
						t.Fatal(got, err, calls, attempts, first.closes)
					}
					if mode == "reauth" && reauth != 1 {
						t.Fatal(reauth)
					}
					return
				}
				code, body, expectedAttempts := 503, "initial rejection", 1
				if mode == "expanded OkCodes" {
					code, body, expectedAttempts = 418, "actual rejection", 2
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Task-Proof") != "actual" || attempts != expectedAttempts || first.closes != 1 || (mode == "expanded OkCodes" && second.closes != 1) {
					t.Fatal(got, err, native, calls, attempts, retries, first.closes, second.closes)
				}
				if phase == "metadata" {
					if got != nil {
						t.Fatal(got)
					}
				} else if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded != nil {
					t.Fatal(got)
				}
				if mode == "changed body" || mode == "KeepResponseBody" || mode == "source mutation" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if mode == "hook cause" && !errors.Is(err, marker) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestImageRecordDownloadDeferredStreamReadAndClosePreserveCausesWithoutInventedBodyProof(t *testing.T) {
	for _, mode := range []string{"literal EOF", "reader cause", "cancel before Read", "source before Read", "outer before Read", "source restored on Close", "close cause", "cancel during Close"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			marker := errors.New("download deferred stream")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			raw := &taskCoreReader{body: "abc", err: io.EOF}
			inner := &taskCoreBody{reader: raw}
			var client *gophercloud.ServiceClient
			original := ""
			if mode == "reader cause" {
				raw.err = marker
			}
			if mode == "close cause" {
				inner.closeErr = marker
			}
			body := io.ReadCloser(inner)
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","checksum":"`+imageRecordDownloadABCMD5+`"}`), nil
				}
				return taskCoreHTTP(req, 206, body), nil
			})
			original = client.Endpoint
			if mode == "source restored on Close" {
				raw.action = func() { client.Endpoint = "https://foreign.test/" }
				body = &imageRecordCloseBody{taskCoreBody: inner, after: func() { client.Endpoint = original }}
			}
			if mode == "cancel during Close" {
				body = &imageRecordCloseBody{taskCoreBody: inner, after: func() { cancel(marker) }}
			}
			got, err := New(client).DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadStream(true))
			if err != nil || got == nil || got.Downloaded == nil || got.Downloaded.Stream == nil || calls != 2 || inner.closes != 0 || raw.read {
				t.Fatal(got, err, calls, inner.closes, raw.read)
			}
			switch mode {
			case "cancel before Read":
				cancel(marker)
			case "source before Read":
				client.Endpoint = "https://foreign.test/"
			case "outer before Read":
				outerBad = true
			}
			buffer := make([]byte, 8)
			n, readErr := got.Downloaded.Stream.Read(buffer)
			if mode == "literal EOF" || mode == "close cause" || mode == "cancel during Close" {
				if n != 3 || string(buffer[:n]) != "abc" || readErr != io.EOF {
					t.Fatal(n, readErr, string(buffer))
				}
			} else {
				var operation *resource.OperationError
				var proof *resource.ResponseError
				if !errors.As(readErr, &operation) || errors.As(readErr, &proof) || operation.Operation != "DownloadImageRecord" {
					t.Fatal(readErr)
				}
				expected := marker
				if mode == "source before Read" || mode == "source restored on Close" {
					expected = resource.ErrInvalidOption
				}
				if !errors.Is(readErr, expected) {
					t.Fatal(readErr)
				}
				if mode == "cancel before Read" || mode == "source before Read" || mode == "outer before Read" {
					if n != 0 || raw.read {
						t.Fatal(n, raw.read)
					}
				}
			}
			closeErr := got.Downloaded.Stream.Close()
			again := got.Downloaded.Stream.Close()
			if inner.closes != 1 || calls != 2 || closeErr != again || got.Checksum.Complete || got.Checksum.Verified || got.Checksum.Actual != "" || got.BytesWritten != 0 || got.Downloaded.Header.Get("X-Task-Proof") != "actual" || got.Downloaded.StatusCode != 206 {
				t.Fatal(got, closeErr, again, inner.closes, calls)
			}
			if mode == "literal EOF" || mode == "reader cause" {
				th.AssertNoErr(t, closeErr)
			} else {
				expected := marker
				if strings.Contains(mode, "source") {
					expected = resource.ErrInvalidOption
				}
				if !errors.Is(closeErr, expected) {
					t.Fatal(closeErr)
				}
			}
		})
	}
}

func TestImageRecordDownloadBorrowedWriterCallbacksKeepStickyGuardsAndPartialCount(t *testing.T) {
	for _, mode := range []string{"cancel", "source restored on Close", "binding", "outer", "writer cause"} {
		t.Run(mode, func(t *testing.T) {
			calls, writes := 0, 0
			marker := errors.New("download borrowed writer")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			inner := &taskCoreBody{reader: strings.NewReader("abc")}
			var client *gophercloud.ServiceClient
			body := io.ReadCloser(inner)
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","checksum":"`+imageRecordDownloadABCMD5+`"}`), nil
				}
				if calls != 2 {
					t.Fatal("accepted body replayed", calls)
				}
				return taskCoreHTTP(req, 203, body), nil
			})
			original := client.Endpoint
			if mode == "source restored on Close" {
				body = &imageRecordCloseBody{taskCoreBody: inner, after: func() { client.Endpoint = original }}
			}
			service := New(client)
			var output bytes.Buffer
			writer := imageRecordDownloadWriterFunc(func(p []byte) (int, error) {
				writes++
				n, _ := output.Write(p)
				switch mode {
				case "cancel":
					cancel(marker)
				case "source restored on Close":
					client.Endpoint = "https://foreign.test/"
				case "binding":
					service.API = nil
				case "outer":
					outerBad = true
				case "writer cause":
					return n, marker
				}
				return n, nil
			})
			got, err := service.DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed", Output: writer}, WithImageRecordDownloadChunkSize(2))
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded == nil || got.BytesWritten != 2 || output.String() != "ab" || got.Checksum == nil || got.Checksum.Complete || got.Checksum.Actual != "" || writes != 1 || calls != 2 || inner.closes != 1 {
				t.Fatal(got, err, output.String(), writes, calls, inner.closes)
			}
			expected := marker
			if mode == "source restored on Close" || mode == "binding" {
				expected = resource.ErrInvalidOption
			}
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadDeferredBodyKeepsCapturedContextAndHTTPTimeoutAlive(t *testing.T) {
	for _, mode := range []string{"caller cancel", "HTTP timeout"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			marker := errors.New("download lifetime cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var requestContext context.Context
			started := make(chan struct{})
			inner := &taskCoreBody{reader: deleteCoreReader(func([]byte) (int, error) { close(started); <-requestContext.Done(); return 0, requestContext.Err() })}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				}
				requestContext = req.Context()
				return taskCoreHTTP(req, 200, inner), nil
			})
			if mode == "HTTP timeout" {
				client.HTTPClient.Timeout = 80 * time.Millisecond
			}
			got, err := New(client).DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadStream(true))
			if err != nil || got == nil || got.Downloaded.Stream == nil {
				t.Fatal(got, err)
			}
			done := make(chan error, 1)
			go func() { _, err := got.Downloaded.Stream.Read(make([]byte, 1)); done <- err }()
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("deferred Read never reached body")
			}
			if mode == "caller cancel" {
				cancel(marker)
			}
			select {
			case readErr := <-done:
				expected := error(context.DeadlineExceeded)
				if mode == "caller cancel" {
					expected = marker
				}
				if !errors.Is(readErr, expected) {
					t.Fatal(readErr)
				}
				var operation *resource.OperationError
				if !errors.As(readErr, &operation) {
					t.Fatal(readErr)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("deferred Read lost request context/timeout")
			}
			_ = got.Downloaded.Stream.Close()
			if calls != 2 || inner.closes != 1 {
				t.Fatal(calls, inner.closes)
			}
		})
	}
}

type imageRecordDownloadHashCallbacks struct {
	hash.Hash
	write func([]byte) (int, error)
	sum   func([]byte) []byte
}

func (h *imageRecordDownloadHashCallbacks) Write(p []byte) (int, error) {
	if h.write != nil {
		return h.write(p)
	}
	return h.Hash.Write(p)
}
func (h *imageRecordDownloadHashCallbacks) Sum(p []byte) []byte {
	if h.sum != nil {
		return h.sum(p)
	}
	return h.Hash.Sum(p)
}

func TestImageRecordDownloadCustomHashWriteAndSumCannotHideFaultsOrChangeCompletedProof(t *testing.T) {
	for _, mode := range []string{"write cause", "short hash write", "cancel Write", "source Write restored on Close", "source Sum", "cancel Sum"} {
		t.Run(mode, func(t *testing.T) {
			calls, writes, sums := 0, 0, 0
			marker := errors.New("download custom hash")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			inner := &taskCoreBody{reader: strings.NewReader("abc")}
			body := io.ReadCloser(inner)
			var client *gophercloud.ServiceClient
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed","hash_algo":"extension","hash_value":"`+imageRecordDownloadABCMD5+`"}`), nil
				}
				return taskCoreHTTP(req, 200, body), nil
			})
			original := client.Endpoint
			if mode == "source Write restored on Close" {
				body = &imageRecordCloseBody{taskCoreBody: inner, after: func() { client.Endpoint = original }}
			}
			digest := md5.New()
			custom := &imageRecordDownloadHashCallbacks{Hash: digest}
			custom.write = func(p []byte) (int, error) {
				writes++
				if mode == "write cause" {
					return 0, marker
				}
				if mode == "short hash write" {
					return len(p) - 1, nil
				}
				n, err := digest.Write(p)
				if mode == "cancel Write" {
					cancel(marker)
				}
				if mode == "source Write restored on Close" {
					client.Endpoint = "https://foreign.test/"
				}
				return n, err
			}
			custom.sum = func(p []byte) []byte {
				sums++
				if mode == "source Sum" {
					client.Endpoint = "https://foreign.test/"
				}
				if mode == "cancel Sum" {
					cancel(marker)
				}
				return digest.Sum(p)
			}
			got, err := New(client).DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed"}, WithImageRecordDownloadHashFactory(func(string) (hash.Hash, error) { return custom, nil }))
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded == nil || string(got.Downloaded.Body) != "abc" || got.BytesWritten != 3 || got.Checksum == nil || got.Checksum.Complete || got.Checksum.Verified || got.Checksum.Actual != "" || calls != 2 || inner.closes != 1 || writes != 1 {
				t.Fatal(got, err, calls, inner.closes, writes, sums)
			}
			expected := marker
			if strings.Contains(mode, "source") {
				expected = resource.ErrInvalidOption
			}
			if mode == "short hash write" {
				expected = io.ErrShortWrite
			}
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
			expectedSums := 0
			if strings.Contains(mode, "Sum") {
				expectedSums = 1
			}
			th.AssertEquals(t, expectedSums, sums)
		})
	}
}

func TestImageRecordDownloadAcceptedBinaryBoundaryFailuresKeepHeadersAndReleaseUnreadBody(t *testing.T) {
	for _, mode := range []string{"transport error", "source accepted restored on Close", "cancel accepted", "outer accepted"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			marker := errors.New("download physical boundary")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			raw := strings.NewReader("abc")
			inner := &taskCoreBody{reader: raw}
			body := io.ReadCloser(inner)
			var client *gophercloud.ServiceClient
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				}
				switch mode {
				case "transport error":
					return nil, marker
				case "source accepted restored on Close":
					client.Endpoint = "https://foreign.test/"
				case "cancel accepted":
					cancel(marker)
				case "outer accepted":
					outerBad = true
				}
				return taskCoreHTTP(req, 299, body), nil
			})
			original := client.Endpoint
			if mode == "source accepted restored on Close" {
				body = &imageRecordCloseBody{taskCoreBody: inner, after: func() { client.Endpoint = original }}
			}
			got, err := New(client).DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed"})
			if got == nil || got.Record == nil || got.Metadata == nil || got.BytesWritten != 0 || got.Checksum != nil || calls != 2 || raw.Len() != 3 {
				t.Fatal(got, err, calls, raw.Len())
			}
			if mode == "transport error" {
				if got.Downloaded != nil || inner.closes != 0 {
					t.Fatal(got, inner.closes)
				}
			} else if got.Downloaded == nil || got.Downloaded.StatusCode != 299 || got.Downloaded.Header.Get("X-Task-Proof") != "actual" || got.Downloaded.Body != nil || inner.closes != 1 {
				t.Fatal(got, inner.closes)
			}
			expected := marker
			if strings.Contains(mode, "source") {
				expected = resource.ErrInvalidOption
			}
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadNativeBackoffRetainsOriginalRejectionAcrossCallbackGuards(t *testing.T) {
	for _, mode := range []string{"source", "cancel", "callback cause"} {
		t.Run(mode, func(t *testing.T) {
			calls, backoffs := 0, 0
			marker := errors.New("download backoff callback")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &taskCoreBody{reader: strings.NewReader("initial rate limit")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				}
				if calls != 2 {
					t.Fatal("failed backoff replayed", calls)
				}
				return taskCoreHTTP(req, 429, body), nil
			})
			client.ProviderClient.MaxBackoffRetries = 1
			client.ProviderClient.RetryBackoffFunc = func(_ context.Context, native *gophercloud.ErrUnexpectedResponseCode, _ error, _ uint) error {
				backoffs++
				if native == nil || native.Actual != 429 {
					t.Fatal(native)
				}
				native.Actual = 418
				native.Body[0] = '!'
				native.ResponseHeader.Set("X-Task-Proof", "mutated callback view")
				switch mode {
				case "source":
					client.Endpoint = "https://foreign.test/"
				case "cancel":
					cancel(marker)
				case "callback cause":
					return marker
				}
				return nil
			}
			got, err := New(client).DownloadImageRecord(ctx, ImageRecordDownloadRequest{ID: "fixed"})
			var native gophercloud.ErrUnexpectedResponseCode
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded != nil || calls != 2 || backoffs != 1 || body.closes != 1 || !errors.As(err, &native) || native.Actual != 429 || string(native.Body) != "initial rate limit" || native.ResponseHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal(got, err, native, calls, backoffs, body.closes)
			}
			expected := marker
			if mode == "source" {
				expected = resource.ErrInvalidOption
			}
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordDownloadRejectedBinaryReadAndCloseFaultsPreventCleanReplay(t *testing.T) {
	for _, mode := range []string{"Read", "Close"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			marker := errors.New("native rejected download IO")
			body := &taskCoreBody{reader: strings.NewReader("native partial")}
			if mode == "Read" {
				body.reader = &taskCoreReader{body: "native partial", err: marker}
			} else {
				body.closeErr = marker
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 203, `{"id":"fixed"}`), nil
				}
				if calls != 2 {
					t.Fatal("faulty rejection replayed", calls)
				}
				return taskCoreHTTP(req, 503, body), nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			got, err := New(client).DownloadImageRecord(context.Background(), ImageRecordDownloadRequest{ID: "fixed"})
			var native gophercloud.ErrUnexpectedResponseCode
			if got == nil || got.Record == nil || got.Metadata == nil || got.Downloaded != nil || !errors.Is(err, marker) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "native partial" || calls != 2 || retries != 1 || body.closes != 1 {
				t.Fatal(got, err, native, calls, retries, body.closes)
			}
		})
	}
}
