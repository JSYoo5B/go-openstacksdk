package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const bidVersionPath = "/catalog/proxy%20prefix/v3/"
const bidRootPath = "/catalog/proxy%20prefix/"
const bidBody = `{"backup-record":{"backup_service":"literal/backend","backup_url":"opaque://backup / % ?#"}}`

func bidClient(cloud *testcloud.Cloud, version string) *gophercloud.ServiceClient {
	client := bmwClient(cloud)
	client.Endpoint = cloud.Server.URL + bidVersionPath + "catalog-project/"
	client.Microversion = version
	return client
}
func bidInput() blockstorage.ImportVolumeBackupRequest {
	return blockstorage.ImportVolumeBackupRequest{BackupService: "literal/backend", BackupURL: "opaque://backup / % ?#"}
}
func bidPost(t *testing.T, req *http.Request, version, token string) {
	t.Helper()
	body := bmwRequest(t, req, http.MethodPost, bmwCollection+"/import_record", version, token)
	if string(body) != bidBody || req.URL.RawQuery != "" || req.ContentLength != int64(len(bidBody)) || len(req.TransferEncoding) != 0 || req.Header.Get("X-OpenStack-Volume-API-Version") != version {
		t.Error("import changed literal body, route, framing or microversion", string(body), req.URL, req.ContentLength, req.TransferEncoding, req.Header)
	}
	if req.Body != nil {
		if err := req.Body.Close(); err != nil {
			t.Error(err)
		}
	}
}
func bidGet(t *testing.T, req *http.Request, escapedPath, token string) {
	t.Helper()
	if req.Method != http.MethodGet || req.URL.EscapedPath() != escapedPath || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-OpenStack-Volume-API-Version") != "" || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
		t.Error("discovery changed captured route/header/body policy", req.Method, req.URL, req.Header, req.ContentLength, req.TransferEncoding)
	}
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil || len(body) != 0 {
			t.Error("discovery acquired a body", string(body), err)
		}
		if err := req.Body.Close(); err != nil {
			t.Error(err)
		}
	}
}
func bidNoApplied(t *testing.T, result *blockstorage.ImportVolumeBackupResult, err error, discoveries int) {
	t.Helper()
	if result == nil || err == nil || result.Applied != nil || result.Value != nil || result.Backup != nil || len(result.Discovery) != discoveries {
		t.Fatal(result, err, discoveries)
	}
	bmwOperation(t, err, "ImportVolumeBackup")
}

func TestImportVolumeBackupSelectedMicroversionWinsWithoutDiscoveryOrCap(t *testing.T) {
	for _, version := range []string{"3.40", "3.80", "latest"} {
		for _, explicit := range []bool{false, true} {
			t.Run(version+map[bool]string{true: "/explicit", false: "/generated"}[explicit], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bidClient(cloud, version)
				if explicit {
					client.MoreHeaders["OpenStack-API-Version"] = "volume " + version
					client.MoreHeaders["X-OpenStack-Volume-API-Version"] = version
				}
				original := make(map[string]string)
				for k, v := range client.MoreHeaders {
					original[k] = v
				}
				var calls atomic.Int32
				reply := `{"backup":{"id":"imported"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					if calls.Add(1) != 1 {
						t.Error("selected import performed discovery or replay", req.URL)
					}
					bidPost(t, req, version, "test-token")
					w.Header().Set("X-Proof", "selected")
					testcloud.JSON(w, 202, reply)
				})
				result, err := blockstorage.ImportVolumeBackup(bmwContext(t), client, bidInput())
				if err != nil || result == nil || result.Value == nil || result.Backup == nil || result.Microversion != version || len(result.Discovery) != 0 || string(result.BackupID) != `"imported"` || calls.Load() != 1 || client.Microversion != version || !reflect.DeepEqual(client.MoreHeaders, original) {
					t.Fatal(result, err, calls.Load(), client)
				}
				bmwPage(t, result.Applied, 202, reply, "selected")
			})
		}
	}
}

func TestImportVolumeBackupDiscoverySelectsBoundedMaximumAndOptionalMinimum(t *testing.T) {
	cases := []struct {
		name, reply, version string
		code                 int
	}{
		{"flat-below-cap", `{"id":"v3.0","status":"CURRENT","max_version":"3.40"}`, "3.40", 200},
		{"version-capped", `{"version":{"id":"v3.0","status":"SUPPORTED","max_version":"3.80","min_version":"3.0"}}`, "3.64", 300},
		{"array-max-only", `{"versions":[{"id":"v2.0","max_version":"2.99"},{"id":"v3.0","status":"STABLE","max_version":"3.40"}]}`, "3.40", 200},
		{"nested-version-fallback", `{"versions":{"values":[{"id":"v3.0","status":"DEPRECATED","max_version":"","version":"3.50"}]}}`, "3.50", 300},
		{"max-precedence", `{"id":"v3.0","max_version":"3.40","version":"3.80"}`, "3.40", 200},
		{"missing-max-ignores-bad-min", `{"id":"v3.0","min_version":"bad"}`, "", 200},
		{"minimum-above-cap", `{"id":"v3.0","max_version":"3.80","min_version":"3.65"}`, "", 200},
		{"longer-maximum", `{"id":"v3.0","max_version":"3.64.0"}`, "3.64", 300},
		{"longer-minimum", `{"id":"v3.0","max_version":"3.80","min_version":"3.64.0"}`, "", 200},
		{"latest-maximum", `{"id":"v3.0","max_version":"latest"}`, "3.64", 200},
		{"finite-latest-maximum", `{"id":"v3.0","max_version":"3.latest"}`, "3.64", 200},
		{"unicode-sign-underscore", `{"id":"v3.0","max_version":"vv+٣.٦_٤"}`, "3.64", 200},
		{"negative-minor", `{"id":"v3.0","max_version":"+3.-1"}`, "3.-1", 200},
		{"inverted-utility-range", `{"id":"v3.0","max_version":"3.20","min_version":"3.60"}`, "3.20", 200},
		{"cross-major-maximum", `{"id":"v3.0","max_version":"2.9"}`, "2.9", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bidClient(cloud, "")
			var calls atomic.Int32
			reply := `{"backup":{"id":"imported"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				switch calls.Add(1) {
				case 1:
					bidGet(t, req, bidVersionPath, "test-token")
					w.Header().Set("X-Proof", "discovery")
					testcloud.JSON(w, tc.code, tc.reply)
				case 2:
					bidPost(t, req, tc.version, "test-token")
					w.Header().Set("X-Proof", "import")
					testcloud.JSON(w, 203, reply)
				default:
					t.Error("import exceeded fixed phases", req.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.ImportVolumeBackup(bmwContext(t), client, bidInput())
			if err != nil || result == nil || result.Value == nil || result.Microversion != tc.version || len(result.Discovery) != 1 || calls.Load() != 2 || client.Microversion != "" {
				t.Fatal(result, err, calls.Load(), client.Microversion)
			}
			bmwPage(t, result.Discovery[0], tc.code, tc.reply, "discovery")
			bmwPage(t, result.Applied, 203, reply, "import")
			saved := bytes.Clone(result.Value)
			result.Discovery[0].Body[0] = '!'
			result.Discovery[0].Header.Set("X-Proof", "changed")
			if !bytes.Equal(saved, result.Value) {
				t.Fatal("discovery aliases normalized Backup")
			}
			bmwPage(t, result.Applied, 203, reply, "import")
		})
	}
}

func TestImportVolumeBackupDiscoveryIsFiniteAndCleanUnavailableFallsBack(t *testing.T) {
	for _, mode := range []string{"404-405", "empty-and-non-v3", "empty-then-v3"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bidClient(cloud, "")
			var calls atomic.Int32
			wantVersion, wantProofs := "", 2
			if mode == "404-405" {
				wantProofs = 0
			}
			if mode == "empty-then-v3" {
				wantVersion = "3.40"
			}
			second := `{"versions":[{"id":"v2.0","max_version":"2.99"}]}`
			if mode == "empty-then-v3" {
				second = `{"version":{"id":"v3.0","max_version":"3.40"}}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				switch n {
				case 1, 2:
					path := bidVersionPath
					if n == 2 {
						path = bidRootPath
					}
					bidGet(t, req, path, "test-token")
					w.Header().Set("X-Proof", map[int32]string{1: "first", 2: "second"}[n])
					if mode == "404-405" {
						testcloud.JSON(w, map[int32]int{1: 404, 2: 405}[n], `{"error":"unavailable"}`)
					} else if n == 1 {
						testcloud.JSON(w, 200, `{}`)
					} else {
						testcloud.JSON(w, 300, second)
					}
				case 3:
					bidPost(t, req, wantVersion, "test-token")
					testcloud.JSON(w, 202, `{"backup":{"id":"imported"}}`)
				default:
					t.Error("discovery escaped two candidates", req.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.ImportVolumeBackup(bmwContext(t), client, bidInput())
			if err != nil || result == nil || result.Applied == nil || result.Value == nil || result.Microversion != wantVersion || len(result.Discovery) != wantProofs || calls.Load() != 3 {
				t.Fatal(result, err, calls.Load())
			}
			if wantProofs != 0 {
				bmwPage(t, result.Discovery[0], 200, `{}`, "first")
				bmwPage(t, result.Discovery[1], 300, second, "second")
			}
		})
	}
}

func TestImportVolumeBackupInvalidAcceptedDiscoveryKeepsOnlyCurrentDiscoveryProof(t *testing.T) {
	for _, reply := range []string{"", "not JSON", `[]`, `null`, `{"versions":null}`, `{"version":null}`, `{"id":"v3.0","max_version":"bad","min_version":"9.0"}`, `{"id":"v3.0","max_version":"3.63.latest"}`, `{"id":"v3.0","max_version":"3.80","min_version":"bad"}`} {
		t.Run(reply, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bidClient(cloud, "")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				if calls.Add(1) != 1 {
					t.Error("accepted discovery error continued or posted", req.URL)
				}
				bidGet(t, req, bidVersionPath, "test-token")
				w.Header().Set("X-Proof", "invalid discovery")
				testcloud.JSON(w, 300, reply)
			})
			result, err := blockstorage.ImportVolumeBackup(bmwContext(t), client, bidInput())
			bidNoApplied(t, result, err, 1)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 300 || string(proof.Body) != reply || proof.Header.Get("X-Proof") != "invalid discovery" || calls.Load() != 1 {
				t.Fatal(result, err, proof, calls.Load())
			}
			bmwPage(t, result.Discovery[0], 300, reply, "invalid discovery")
			if len(proof.Body) != 0 {
				proof.Body[0] = '!'
			}
			proof.Header.Set("X-Proof", "changed error")
			bmwPage(t, result.Discovery[0], 300, reply, "invalid discovery")
		})
	}
}

func TestImportVolumeBackupDiscoveryFaultsSourceDriftAndCancellationAreTerminal(t *testing.T) {
	modes := []string{"accepted-read", "accepted-close", "accepted-joined", "404-read", "405-close", "404-joined", "404-source", "expanded-404", "expanded-405", "retry-joined"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bidClient(cloud, "")
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause, callbackCause := errors.New("discovery Read"), errors.New("discovery Close"), errors.New("discovery cancellation"), errors.New("discovery retry callback")
			reply := `{"id":"v3.0","max_version":"3.80"}`
			broken := &snapshotReadTransportBody{data: strings.NewReader(reply)}
			if strings.Contains(mode, "read") || strings.Contains(mode, "joined") && mode != "retry-joined" {
				broken.readError = readCause
			}
			if strings.Contains(mode, "close") || strings.Contains(mode, "joined") && mode != "retry-joined" {
				broken.closeError = closeCause
			}
			joined := strings.Contains(mode, "joined") && mode != "retry-joined"
			broken.onClose = func() {
				if joined || mode == "404-source" {
					client.ResourceBase += "changed/"
				}
				if joined {
					cancel(cancelCause)
				}
			}
			var calls, retries atomic.Int32
			expanded := strings.HasPrefix(mode, "expanded")
			if expanded || mode == "retry-joined" {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
					if !gophercloud.ResponseCodeIs(original, 503) {
						return original
					}
					if retries.Add(1) > 1 {
						return errors.Join(original, errors.New("bounded discovery fixture exhausted"))
					}
					if mode == "retry-joined" {
						client.Microversion = "3.61"
						cancel(cancelCause)
						return callbackCause
					}
					code := 404
					if mode == "expanded-405" {
						code = 405
					}
					opts.OkCodes = append(opts.OkCodes, code)
					return nil
				}
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				bidGet(t, req, bidVersionPath, "test-token")
				if (expanded || mode == "retry-joined") && n == 1 {
					return snapshotReadTransportResponse(req, 503, io.NopCloser(strings.NewReader(`{"error":"retry once"}`)), "retry"), nil
				}
				if n > 1 && !expanded {
					t.Error("fault advanced discovery", req.URL)
				}
				code := 200
				if strings.HasPrefix(mode, "404") || mode == "expanded-404" {
					code = 404
				}
				if strings.HasPrefix(mode, "405") || mode == "expanded-405" {
					code = 405
				}
				return snapshotReadTransportResponse(req, code, broken, "current discovery"), nil
			})
			result, err := blockstorage.ImportVolumeBackup(ctx, client, bidInput())
			wantCalls, wantProofs := int32(1), 0
			if expanded {
				wantCalls = 2
			}
			if strings.HasPrefix(mode, "accepted") {
				wantProofs = 1
			}
			bidNoApplied(t, result, err, wantProofs)
			if calls.Load() != wantCalls {
				t.Fatal(result, err, calls.Load())
			}
			if mode != "retry-joined" && broken.closes.Load() != 1 {
				t.Fatal("current body close count", broken.closes.Load(), err)
			}
			if broken.readError != nil && !errors.Is(err, readCause) || broken.closeError != nil && !errors.Is(err, closeCause) || joined && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption)) || mode == "404-source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("discovery fault was lost", err)
			}
			if mode == "retry-joined" && (!errors.Is(err, callbackCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || retries.Load() != 1) {
				t.Fatal(err, retries.Load())
			}
			if expanded {
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if retries.Load() != 1 || !errors.As(err, &native) || native.Actual != map[string]int{"expanded-404": 404, "expanded-405": 405}[mode] || errors.As(err, &accepted) {
					t.Fatal("expanded status became clean missing metadata", err, native, accepted, retries.Load())
				}
			}
			if wantProofs == 1 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || string(proof.Body) != reply || proof.StatusCode != 200 {
					t.Fatal(proof, err)
				}
				bmwPage(t, result.Discovery[0], 200, reply, "current discovery")
			}
		})
	}
}
