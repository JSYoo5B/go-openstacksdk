package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

var cinderMemberRejectionEntries = []string{"Backup Get", "Snapshot Get", "Snapshot ByID"}

type cinderMemberRejectionResult struct {
	value    json.RawMessage
	selected *resource.RawResource
	observed bool
	pages    int
}

func cinderMemberRejectionCall(ctx context.Context, client *gophercloud.ServiceClient, entry string) (*cinderMemberRejectionResult, error) {
	switch entry {
	case "Backup Get":
		value, err := blockstorage.GetVolumeBackup(ctx, client, blockstorage.GetVolumeBackupRequest{NameOrID: "literal"})
		if value == nil {
			return nil, err
		}
		return &cinderMemberRejectionResult{value: value.Value, selected: value.Backup, observed: value.Observed != nil, pages: len(value.Pages)}, err
	case "Snapshot Get":
		value, err := blockstorage.GetVolumeSnapshot(ctx, client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "literal"})
		if value == nil {
			return nil, err
		}
		return &cinderMemberRejectionResult{value: value.Value, selected: value.Snapshot, observed: value.Observed != nil, pages: len(value.Pages)}, err
	case "Snapshot ByID":
		value, err := blockstorage.GetVolumeSnapshotByID(ctx, client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "literal"})
		if value == nil {
			return nil, err
		}
		return &cinderMemberRejectionResult{value: value.Value, selected: value.Snapshot, observed: value.Observed != nil, pages: len(value.Pages)}, err
	default:
		panic("unknown cinder rejection fixture entry")
	}
}
func cinderMemberRejectionRoute(entry string) (route, singular, plural, operation, kind string) {
	if entry == "Backup Get" {
		return "backups", "backup", "backups", "GetVolumeBackup", "volume backup"
	}
	operation = "GetVolumeSnapshot"
	if entry == "Snapshot ByID" {
		operation = "GetVolumeSnapshotByID"
	}
	return "snapshots", "snapshot", "snapshots", operation, "volume snapshot"
}
func cinderMemberRejectionContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
func cinderMemberRejectionAssert(t *testing.T, entry string, result *cinderMemberRejectionResult, err error, status int, body string) {
	t.Helper()
	_, _, _, operation, kind := cinderMemberRejectionRoute(entry)
	var outer *resource.OperationError
	var native gophercloud.ErrUnexpectedResponseCode
	var admitted *resource.ResponseError
	if result == nil || result.value != nil || result.selected != nil || result.observed || result.pages != 0 || err == nil || !errors.As(err, &outer) || outer.Operation != operation || outer.Resource != kind || !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Proof") != "rejected-member" || errors.As(err, &admitted) {
		t.Fatal("rejected member became success, fallback or fabricated observation", result, err, outer, native, admitted)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("pure native IO failure required cancellation to be terminal", err)
	}
}

func TestCinderMemberCompatibleRejectionsKeepPureReadAndCloseFaultsTerminal(t *testing.T) {
	for _, entry := range cinderMemberRejectionEntries {
		for _, status := range []int{400, 403, 404} {
			for _, fault := range []string{"Read", "Close", "Read and Close"} {
				t.Run(fmt.Sprintf("%s/%d/%s", entry, status, fault), func(t *testing.T) {
					cloud := testcloud.New(t)
					client := snapshotReadContractClient(cloud)
					route, _, plural, _, _ := cinderMemberRejectionRoute(entry)
					readCause, closeCause := errors.New("rejected member Read"), errors.New("rejected member Close")
					rejected := `{"error":"current rejected body"}`
					body := &snapshotReadTransportBody{data: strings.NewReader(rejected)}
					if fault != "Close" {
						body.readError = readCause
					}
					if fault != "Read" {
						body.closeError = closeCause
					}
					var members, lists atomic.Int32
					cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
						if r.URL.Path == snapshotReadContractBase+route+"/literal" {
							members.Add(1)
							snapshotReadContractWire(t, r, snapshotReadContractBase+route+"/literal", "test-token")
							if r.URL.RawQuery != "" {
								t.Error(r.URL)
							}
							return snapshotReadTransportResponse(r, status, body, "rejected-member"), nil
						}
						lists.Add(1)
						return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"`+plural+`":[]}`)), "unexpected-fallback"), nil
					})
					result, err := cinderMemberRejectionCall(cinderMemberRejectionContext(t), client, entry)
					cinderMemberRejectionAssert(t, entry, result, err, status, rejected)
					if (fault != "Close" && !errors.Is(err, readCause)) || (fault != "Read" && !errors.Is(err, closeCause)) || (fault == "Close" && errors.Is(err, readCause)) || (fault == "Read" && errors.Is(err, closeCause)) || members.Load() != 1 || lists.Load() != 0 || body.closes.Load() != 1 {
						t.Fatal("pure IO cause lost or member was replayed/fell back", err, members.Load(), lists.Load(), body.closes.Load())
					}
				})
			}
		}
	}
}

func TestCinderMemberCleanCompatibleRejectionsRetainFallbackAndByIDContrast(t *testing.T) {
	for _, entry := range cinderMemberRejectionEntries {
		for _, status := range []int{400, 403, 404} {
			t.Run(fmt.Sprintf("%s/%d", entry, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := snapshotReadContractClient(cloud)
				route, _, plural, _, _ := cinderMemberRejectionRoute(entry)
				rejected := `{"error":"clean native rejection"}`
				body := &snapshotReadTransportBody{data: strings.NewReader(rejected)}
				var members, lists atomic.Int32
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == snapshotReadContractBase+route+"/literal" {
						members.Add(1)
						snapshotReadContractWire(t, r, snapshotReadContractBase+route+"/literal", "test-token")
						if r.URL.RawQuery != "" {
							t.Error(r.URL)
						}
						return snapshotReadTransportResponse(r, status, body, "rejected-member"), nil
					}
					lists.Add(1)
					snapshotReadContractWire(t, r, snapshotReadContractBase+route+"/detail", "test-token")
					if r.URL.Query().Get("name") != "literal" || len(r.URL.Query()) != 1 {
						t.Error(r.URL)
					}
					return snapshotReadTransportResponse(r, 203, io.NopCloser(strings.NewReader(`{"`+plural+`":[{"id":"actual","name":"literal"}]}`)), "current-list"), nil
				})
				result, err := cinderMemberRejectionCall(cinderMemberRejectionContext(t), client, entry)
				if entry == "Snapshot ByID" {
					cinderMemberRejectionAssert(t, entry, result, err, status, rejected)
					if lists.Load() != 0 {
						t.Fatal("ByID acquired fallback", lists.Load())
					}
				} else {
					if err != nil || result == nil || result.selected == nil || result.value == nil || result.observed || result.pages != 1 || string(result.selected.Body["id"]) != `"actual"` || result.selected.StatusCode != 203 || result.selected.Header.Get("X-Proof") != "current-list" || lists.Load() != 1 {
						t.Fatal("clean compatible rejection stopped exact fallback or borrowed member proof", result, err, lists.Load())
					}
				}
				if members.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(members.Load(), body.closes.Load())
				}
			})
		}
	}
}

func TestCinderMemberNativeRetryCannotSwallowRejectedIOOrReplaceOriginalStatus(t *testing.T) {
	for _, entry := range cinderMemberRejectionEntries {
		for _, status := range []int{400, 403, 404} {
			for _, callbackFails := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/callback-fails-%t", entry, status, callbackFails), func(t *testing.T) {
					cloud := testcloud.New(t)
					client := snapshotReadContractClient(cloud)
					route, singular, plural, _, _ := cinderMemberRejectionRoute(entry)
					rejected := `{"error":"retry original rejected body"}`
					readCause, closeCause, callbackCause := errors.New("native retry original Read"), errors.New("native retry original Close"), errors.New("native retry caller sentinel")
					body := &snapshotReadTransportBody{data: strings.NewReader(rejected), readError: readCause, closeError: closeCause}
					var requests, callbacks, lists atomic.Int32
					cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
						if r.URL.Path != snapshotReadContractBase+route+"/literal" {
							lists.Add(1)
							return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"`+plural+`":[]}`)), "unexpected-fallback"), nil
						}
						call := requests.Add(1)
						snapshotReadContractWire(t, r, snapshotReadContractBase+route+"/literal", "test-token")
						if r.URL.RawQuery != "" {
							t.Error(r.URL)
						}
						if call == 1 {
							return snapshotReadTransportResponse(r, status, body, "rejected-member"), nil
						}
						// If IO is swallowed, give the retry a valid success so the regression
						// fails promptly instead of relying on its five-second context bound.
						return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"`+singular+`":{"id":"literal"}}`)), "unexpected-resend"), nil
					})
					cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
						callback := callbacks.Add(1)
						var native gophercloud.ErrUnexpectedResponseCode
						if method != "GET" || target != client.ServiceURL(route, "literal") || options == nil || options.JSONBody != nil || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody || !errors.As(original, &native) || native.Actual != status || string(native.Body) != rejected || native.ResponseHeader.Get("X-Proof") != "rejected-member" || count != 1 {
							t.Error("native callback lost original rejected status or fixed ownership", method, target, options, original, count)
						}
						if callback > 1 {
							return errors.New("bounded regression callback: unexpected second retry")
						}
						if callbackFails {
							return callbackCause
						}
						return nil
					}
					result, err := cinderMemberRejectionCall(cinderMemberRejectionContext(t), client, entry)
					cinderMemberRejectionAssert(t, entry, result, err, status, rejected)
					if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || (callbackFails && !errors.Is(err, callbackCause)) || (!callbackFails && errors.Is(err, callbackCause)) || requests.Load() != 1 || callbacks.Load() != 1 || lists.Load() != 0 || body.closes.Load() != 1 {
						t.Fatal("retry swallowed faults/callback cause or resent rejected member", err, requests.Load(), callbacks.Load(), lists.Load(), body.closes.Load())
					}
				})
			}
		}
	}
}
