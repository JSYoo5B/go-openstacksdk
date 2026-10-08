package compute_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestServerIPWorkflowsKeepAcceptedAssignmentModelOnProcessingError(t *testing.T) {
	for _, mode := range []string{"GetActive", "Wait", "Create", "Ensure"} {
		t.Run(mode, func(t *testing.T) {
			f := newAutomaticFixture(t)
			cause := errors.New("IP accepted Close failed")
			f.rawBody = func(int32) (int, string) {
				return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFixed + `}}`
			}
			base := f.cloud.Provider.HTTPClient.Transport
			f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
				response, err := base.RoundTrip(r)
				if err == nil && r.Method == "POST" && r.URL.Path == "/v2.0/floatingips" {
					response.Header.Set("X-IP-Proof", "accepted")
					response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error { return cause }}
				}
				return response, err
			})
			input := compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}
			var result *compute.AutomaticServerIPResult
			var err error
			switch mode {
			case "GetActive":
				result, err = f.service.GetActiveServer(context.Background(), input, serverReadyOptions()...)
			case "Wait":
				result, err = f.service.WaitForServer(context.Background(), input, serverReadyOptions()...)
			case "Ensure":
				result, err = f.service.EnsureServerFloatingIP(context.Background(), input, automaticOptions()...)
			case "Create":
				addAutomaticCreate(f, `{"server":{"id":"server","status":"BUILD"}}`, 202, nil)
				created, createErr := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), automaticCreateOptions())
				err = createErr
				if created == nil || created.Creation == nil {
					t.Fatal(created, err)
				}
				result = created.Automatic
			}
			var proof *resource.ResponseError
			if !errors.Is(err, cause) || result == nil || result.Server.ID != "server" || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP == nil || result.Assignment.FloatingIP.ID != "ip" || result.Assignment.FloatingIP.PortID != "port" || result.Assignment.FloatingIP.Status != "DOWN" || result.Observed || !errors.As(err, &proof) || proof.StatusCode != 201 || proof.Header.Get("X-IP-Proof") != "accepted" || f.posts.Load() != 1 {
				t.Fatal(result, err, proof, f.posts.Load())
			}
			wantRaw := int32(0)
			if mode == "Wait" || mode == "Create" {
				wantRaw = 1
			}
			if f.raw.Load() != wantRaw {
				t.Fatal("unexpected raw observation", f.raw.Load(), wantRaw)
			}
		})
	}
}
