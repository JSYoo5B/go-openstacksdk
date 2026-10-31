package connectiontest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionBackupServerBindsImageLazily(t *testing.T) {
	for _, status := range []int{202, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			var computeLoads, imageLoads, actions, imageGets atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				switch opts.Type {
				case "compute":
					computeLoads.Add(1)
					return cloud.Server.URL + "/compute/v2.1/project/", nil
				case "image":
					imageLoads.Add(1)
					if actions.Load() != 1 {
						t.Error("Glance bound before createBackup action")
					}
					return cloud.Server.URL + "/image/", nil
				default:
					t.Error(opts.Type)
					return "", errors.New("unexpected service lookup")
				}
			}
			cloud.Mux.HandleFunc("GET /compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.100","links":[{"rel":"self","href":"https://advertised.invalid/v2.1/"}]}}`)
			})
			cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers/s1/action", func(w http.ResponseWriter, r *http.Request) {
				actions.Add(1)
				data, err := io.ReadAll(r.Body)
				if err != nil || string(data) != `{"createBackup":{"backup_type":"daily","name":"backup","rotation":3}}` || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.45" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(string(data), err, r.Header)
				}
				w.Header().Set("X-Backup-Stage", "action")
				testcloud.JSON(w, status, `{"image_id":"img1"}`)
			})
			cloud.Mux.HandleFunc("GET /image/v2/images/img1", func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.Header)
				}
				state := "queued"
				if imageGets.Add(1) > 1 {
					state = "active"
				}
				w.Header().Set("X-Backup-Stage", "image")
				testcloud.JSON(w, 200, `{"id":"img1","status":"`+state+`"}`)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.96"))
			if err != nil {
				t.Fatal(err)
			}
			if computeLoads.Load() != 0 || imageLoads.Load() != 0 {
				t.Fatal("eager service binding")
			}
			service, err := conn.Compute(context.Background())
			if err != nil || computeLoads.Load() != 1 || imageLoads.Load() != 0 {
				t.Fatal(err, computeLoads.Load(), imageLoads.Load())
			}
			result, err := service.BackupServer(context.Background(), "s1", compute.BackupServerRequest{Name: "backup", BackupType: "daily", Rotation: 3}, compute.WithBackupServerWait(true), compute.WithBackupServerPollInterval(time.Nanosecond))
			if status == 500 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 500 || result == nil || result.Action == nil || imageLoads.Load() != 0 || imageGets.Load() != 0 {
					t.Fatal(result, err, imageLoads.Load(), imageGets.Load())
				}
				return
			}
			if err != nil || result == nil || result.Action == nil || result.Action.StatusCode != 202 || result.Action.Header.Get("X-Backup-Stage") != "action" || result.Image == nil || string(result.Image.Resource.Body["status"]) != `"active"` || result.Image.Header.Get("X-Backup-Stage") != "image" || result.Wait == nil || result.Wait.Lookups != 1 || imageLoads.Load() != 1 || imageGets.Load() != 2 || actions.Load() != 1 {
				t.Fatal(result, err, imageLoads.Load(), imageGets.Load(), actions.Load())
			}
			glance, err := conn.Image(context.Background())
			if err != nil || glance.RawClient().Endpoint != cloud.Server.URL+"/image/" || glance.RawClient().ResourceBaseURL() != cloud.Server.URL+"/image/v2/" || imageLoads.Load() != 1 {
				t.Fatal("lazy image cache must retain the default factory route", glance, err, imageLoads.Load())
			}
			if service.RawClient().Microversion != "2.96" {
				t.Fatal(service.RawClient().Microversion)
			}
			if data := string(result.Action.Body); !strings.Contains(data, `"image_id":"img1"`) {
				t.Fatal(data)
			}
		})
	}
}
