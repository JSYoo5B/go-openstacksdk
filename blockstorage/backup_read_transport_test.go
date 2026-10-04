package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const backupTransportBasic = snapshotReadContractBase + "backups"
const backupTransportDetail = backupTransportBasic + "/detail"

func backupTransportPage(rows, next string) string {
	if next == "" {
		return `{"backups":` + rows + `}`
	}
	href, _ := json.Marshal(next)
	return `{"backups":` + rows + `,"backups_links":[{"rel":"next","href":` + string(href) + `}]}`
}

func TestVolumeBackupReadOriginalOptionsOwnEveryPreparedValueAndSourceHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls, callbacks atomic.Int32
	filters := json.RawMessage(`{"status":"available","description":"wanted"}`)
	headers := map[string]string{"x-extra": "owned"}
	version := "latest"
	first := func(o *blockstorage.VolumeBackupListOpts) error {
		callbacks.Add(1)
		o.Filters = &filters
		o.Headers = headers
		o.Microversion = &version
		return nil
	}
	second := func(o *blockstorage.VolumeBackupListOpts) error {
		callbacks.Add(1)
		filters = json.RawMessage(`{"status":"changed","description":"changed"}`)
		headers["x-extra"] = "changed"
		version = "3.61"
		client.MoreHeaders["x-source"] = "later ordinary change"
		return nil
	}
	third := func(o *blockstorage.VolumeBackupListOpts) error {
		callbacks.Add(1)
		if o.Filters == nil || string(*o.Filters) != `{"status":"available","description":"wanted"}` || o.Headers["x-extra"] != "owned" || o.Microversion == nil || *o.Microversion != "latest" {
			t.Error("previous callback retained mutable inputs", o)
		}
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != backupTransportDetail || r.URL.Query().Get("status") != "available" || len(r.URL.Query()) != 1 || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Extra") != "owned" || r.Header.Get("OpenStack-API-Version") != "volume latest" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, backupTransportPage(`[{"id":"owned","description":"wanted"}]`, ""))
	})
	result, err := blockstorage.ListVolumeBackups(context.Background(), client, first, second, third)
	if err != nil || result == nil || len(result.Backups) != 1 || calls.Load() != 1 || callbacks.Load() != 3 || client.Microversion != "3.60" || client.MoreHeaders["x-source"] != "later ordinary change" {
		t.Fatal(result, err, calls.Load(), callbacks.Load())
	}
}

func TestVolumeBackupReadAcceptedReadCloseCancellationRetainsCurrentPhysicalStage(t *testing.T) {
	for _, stage := range []string{"member", "list", "member + source", "list + source"} {
		t.Run(stage, func(t *testing.T) {
			changed := strings.HasSuffix(stage, " + source")
			stage = strings.TrimSuffix(stage, " + source")
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("current read"), errors.New("current close"), errors.New("current cancellation")
			body := `{"backup":{"id":"wire"}}`
			if stage == "list" {
				body = backupTransportPage(`[{"id":"wire"}]`, "")
			}
			reader := &snapshotReadTransportBody{data: strings.NewReader(body), readError: readCause, closeError: closeCause, onClose: func() {
				if changed {
					client.ResourceBase += "changed/"
				}
				cancel(cancelCause)
			}}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return snapshotReadTransportResponse(r, 203, reader, "current"), nil
			})
			var observed *blockstorage.VolumeBackupsPage
			var pages []*blockstorage.VolumeBackupsPage
			var value json.RawMessage
			var err error
			if stage == "member" {
				result, e := blockstorage.GetVolumeBackup(ctx, client, blockstorage.GetVolumeBackupRequest{NameOrID: "literal"})
				err = e
				if result == nil || result.Backup != nil {
					t.Fatal(result, e)
				}
				observed, pages, value = result.Observed, result.Pages, result.Value
			} else {
				result, e := blockstorage.ListVolumeBackups(ctx, client)
				err = e
				if result == nil || result.Backups != nil {
					t.Fatal(result, e)
				}
				pages, value = result.Pages, result.Value
			}
			var physical *resource.ResponseError
			if value != nil || calls.Load() != 1 || reader.closes.Load() != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current" {
				t.Fatal(err, calls.Load(), reader.closes.Load())
			}
			var operation *resource.OperationError
			wantOperation := "GetVolumeBackup"
			if stage == "list" {
				wantOperation = "ListVolumeBackups"
			}
			if !errors.As(err, &operation) || operation.Operation != wantOperation || operation.Resource != "volume backup" {
				t.Fatal("backup failure borrowed another resource operation", err)
			}
			if changed && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("source drift lost under context failure", err)
			}
			proof := observed
			if stage == "list" {
				if observed != nil || len(pages) != 1 {
					t.Fatal(observed, pages)
				}
				proof = pages[0]
			} else if proof == nil || len(pages) != 0 {
				t.Fatal(proof, pages)
			}
			physical.Body[0] = '!'
			physical.Header.Set("X-Proof", "error mutated")
			if string(proof.Body) != body || proof.Header.Get("X-Proof") != "current" {
				t.Fatal("error aliases admitted proof")
			}
		})
	}
}

func TestVolumeBackupReadNativeReauthRetryKeepLiveTokenAndOwnedSourceOnce(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls, reauths, retries atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cloud.Provider.ReauthFunc = func(context.Context) error {
		reauths.Add(1)
		cloud.Provider.SetToken("reauth-token")
		client.MoreHeaders["x-source"] = "changed ordinary"
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, err error, _ uint) error {
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if retries.Add(1) > 1 {
			return errors.New("bounded retry fixture exhausted")
		}
		if method != http.MethodGet || !strings.HasSuffix(target, "backups/literal") || o.JSONBody != nil || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || len(o.OkCodes) != 300 || o.OkCodes[0] != 100 || o.OkCodes[299] != 399 {
			t.Error("owned request changed", method, target, o)
		}
		o.MoreHeaders["X-Native"] = "retry"
		cloud.Provider.SetToken("retry-token")
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		token := "test-token"
		if n == 2 {
			token = "reauth-token"
		} else if n == 3 {
			token = "retry-token"
		}
		snapshotReadContractWire(t, r, backupTransportBasic+"/literal", token)
		switch n {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 503, `{"error":"retry"}`)
		case 3:
			if r.Header.Get("X-Native") != "retry" {
				t.Error(r.Header)
			}
			w.Header().Set("X-Proof", "accepted once")
			testcloud.JSON(w, 203, `{"backup":{"id":"wire"}}`)
		default:
			t.Error("workflow replay", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeBackup(ctx, client, blockstorage.GetVolumeBackupRequest{NameOrID: "literal"})
	if err != nil || result == nil || result.Backup == nil || result.Observed == nil || calls.Load() != 3 || reauths.Load() != 1 || retries.Load() != 1 || result.Observed.StatusCode != 203 || result.Observed.Header.Get("X-Proof") != "accepted once" {
		t.Fatal(result, err, calls.Load(), reauths.Load(), retries.Load())
	}
}

func TestVolumeBackupReadNativeExpandedFallbackStatusIsTerminalOriginalPolicyRejection(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, retries atomic.Int32
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if retries.Add(1) > 1 {
					return errors.New("bounded fixture exhausted")
				}
				o.OkCodes = append(o.OkCodes, code)
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				snapshotReadContractWire(t, r, backupTransportBasic+"/literal", "test-token")
				n := calls.Add(1)
				if n == 1 {
					testcloud.JSON(w, 503, `{"error":"retry"}`)
					return
				}
				if n != 2 {
					t.Error("expanded status borrowed as clean fallback", r.URL)
				}
				w.Header().Set("X-Proof", "expanded current")
				testcloud.JSON(w, code, `{"error":"expanded policy"}`)
			})
			result, err := blockstorage.GetVolumeBackup(ctx, client, blockstorage.GetVolumeBackupRequest{NameOrID: "literal"})
			var physical *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Value != nil || result.Backup != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != 2 || retries.Load() != 1 || errors.As(err, &physical) || !errors.As(err, &native) || native.Actual != code || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.ResponseHeader.Get("X-Proof") != "expanded current" {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
		})
	}
}
