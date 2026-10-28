package serverexternalevents_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/serverexternalevents"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func eventAPI(t *testing.T, reply string, status int) (*serverexternalevents.API, *gophercloud.ServiceClient, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/v2.1")
	client.Microversion = "2.96"
	calls := &[]string{}
	cloud.Mux.HandleFunc("/v2.1/os-server-external-events", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*calls = append(*calls, r.Method+" "+string(body))
		if r.Header.Get("OpenStack-API-Version") != "compute 2.96" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, status, reply)
	})
	return serverexternalevents.New(client), client, calls
}
func TestPythonExternalEventsBatchRawWireAndDeclaredProjection(t *testing.T) {
	reply := `{"events":[{"name":"network-changed","server_uuid":"server","code":"200","hosts":"host","metadata":[1],"extension":9007199254740993}]}`
	api, _, calls := eventAPI(t, reply, 203)
	input := []serverexternalevents.Event{{"name": "network-changed", "server_uuid": "server", "code": "-1", "hosts": "host", "metadata": [1]int{1}, "opaque": func() {}, "unknown": "drop"}}
	mutate := func(*request.Config[serverexternalevents.CreateOpts]) error { input[0]["name"] = "changed"; return nil }
	rows, err := api.CreateRecords(context.Background(), input, mutate)
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0] != `POST {"events":[{"code":"-1","name":"network-changed","server_uuid":"server"}]}` {
		t.Fatal(*calls)
	}
	count := 0
	for row, err := range rows {
		if err != nil {
			t.Fatal(err)
		}
		count++
		if row.StatusCode != 203 || string(row.Envelope) != reply || string(row.Resource.Body["code"]) != "200" || row.Resource.Body["hosts"] != nil || row.Resource.Body["metadata"] != nil || row.Resource.Body["extension"] != nil || string(row.Wire.Body["extension"]) != "9007199254740993" {
			t.Fatal(row)
		}
	}
	if count != 1 {
		t.Fatal(count)
	}
}
func TestPythonExternalEventsPartialRowsRemainLazyWithReceipt(t *testing.T) {
	reply := `{"events":[{"tag":"first"},{"self":"collision"}]}`
	api, _, calls := eventAPI(t, reply, 202)
	rows, err := api.CreateRecords(context.Background(), []serverexternalevents.Event{{"name": "seed"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 {
		t.Fatal("POST must precede iteration")
	}
	count := 0
	for row, err := range rows {
		if err != nil {
			var proof *resource.ResponseError
			if count != 1 || row != nil || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != reply {
				t.Fatal(count, row, err, proof)
			}
			return
		}
		count++
		if string(row.Resource.Body["name"]) != "null" || string(row.Resource.Body["tag"]) != `"first"` {
			t.Fatal(row)
		}
	}
	t.Fatal("missing lazy constructor failure")
}
func TestPythonExternalEventsRejectInputBeforeHTTP(t *testing.T) {
	api, _, calls := eventAPI(t, `{"events":[]}`, 200)
	inputs := [][]serverexternalevents.Event{nil, {nil}, {{"self": nil}}, {{"connection": nil}}, {{"_synchronized": false}}, {{"cls": nil}}, {{"code": "²"}}}
	for _, input := range inputs {
		if _, err := api.CreateRecords(context.Background(), input); err == nil {
			t.Fatal(input)
		}
	}
	if _, err := api.CreateRecords(context.Background(), []serverexternalevents.Event{{}}, nil); err == nil {
		t.Fatal("nil option")
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
}
func TestPythonExternalEventsEnvelopeAndLazyShapeErrors(t *testing.T) {
	for _, reply := range []string{"", `[]`, `{}`, `{"events":null}`, `{"events":[null]}`, `{"events":{}}`} {
		t.Run(reply, func(t *testing.T) {
			api, _, _ := eventAPI(t, reply, 202)
			rows, err := api.CreateRecords(context.Background(), []serverexternalevents.Event{{}})
			if err == nil {
				for _, cause := range rows {
					if cause != nil {
						err = cause
						break
					}
				}
			}
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || string(proof.Body) != reply || proof.StatusCode != 202 {
				t.Fatal(err, proof)
			}
		})
	}
}
func TestPythonExternalEventsSourceGuardAndCanceledIteration(t *testing.T) {
	api, client, calls := eventAPI(t, `{"events":[{}]}`, 200)
	option := func(*request.Config[serverexternalevents.CreateOpts]) error {
		client.Endpoint = "https://foreign.invalid/"
		return nil
	}
	if _, err := api.CreateRecords(context.Background(), []serverexternalevents.Event{{}}, option); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	api, _, _ = eventAPI(t, `{"events":[{}]}`, 200)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop")
	rows, err := api.CreateRecords(ctx, []serverexternalevents.Event{{}})
	if err != nil {
		t.Fatal(err)
	}
	cancel(cause)
	for row, err := range rows {
		if row != nil || !errors.Is(err, cause) {
			t.Fatal(row, err)
		}
	}
}

func TestPythonExternalEventsHTTPAndAcceptedCloseAreTerminal(t *testing.T) {
	for _, status := range []int{403, 202} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			api, client, _ := eventAPI(t, `{"events":[]}`, 200)
			cause := errors.New("close")
			attempts := 0
			client.ProviderClient.HTTPClient.Transport = testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				attempts++
				return &http.Response{StatusCode: status, Header: http.Header{"X-Proof": {"yes"}}, Request: r, Body: &testcloud.Body{Data: `{"events":[]}`, CloseErr: cause}}, nil
			})
			_, err := api.CreateRecords(context.Background(), []serverexternalevents.Event{{}})
			if attempts != 1 || !errors.Is(err, cause) {
				t.Fatal(attempts, err)
			}
			if status == 403 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "yes" || string(native.Body) != `{"events":[]}` {
					t.Fatal(err, native)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || proof.Header.Get("X-Proof") != "yes" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestPythonExternalEventsSelectedAPIReplacementAndIterationBreak(t *testing.T) {
	api, client, calls := eventAPI(t, `{"events":[{}, {"self":null}]}`, 200)
	copyClient := *client
	replace := func(*request.Config[serverexternalevents.CreateOpts]) error {
		*api = *serverexternalevents.New(&copyClient)
		return nil
	}
	if _, err := api.CreateRecords(context.Background(), []serverexternalevents.Event{{}}, replace); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	api, _, _ = eventAPI(t, `{"events":[{}, {"self":null}]}`, 200)
	rows, err := api.CreateRecords(context.Background(), []serverexternalevents.Event{{}})
	if err != nil {
		t.Fatal(err)
	}
	for row, err := range rows {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		break
	}
}
