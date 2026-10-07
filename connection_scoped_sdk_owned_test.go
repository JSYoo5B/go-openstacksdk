package gophercloudsdk_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secretconsumers"
	"github.com/JSYoo5B/gophercloudsdk/messaging/v2/subscriptions"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSDKOwnedSubscriptionAndConsumerScopes(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Errorf("catalog selection: %#v", options)
		}
		switch options.Type {
		case "key-manager":
			return cloud.Server.URL + "/catalog/barbican/v1/", nil
		case "message":
			return cloud.Server.URL + "/catalog/zaqar/v2/", nil
		default:
			t.Errorf("unexpected service lookup: %#v", options)
			return "", resource.ErrUnsupported
		}
	}
	const clientID = "11111111-1111-4111-8111-111111111111"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-SDK-Source") != "shared" {
			t.Error("shared source header missing", r.Header)
		}
		w.Header().Set("X-Request-ID", "scoped-response")
		switch r.Method + " " + r.URL.Path {
		case "POST /reverse/barbican/v1/secrets/secret-one/consumers", "DELETE /reverse/barbican/v1/secrets/secret-one/consumers":
			if r.Header.Get("X-Auth-Token") != "consumer-token" || r.URL.RawQuery != "" {
				t.Error("consumer source changed", r.Header, r.URL)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"resource_id":"volume-one","resource_type":"volume","service":"volume"}` {
				t.Error(string(body), err)
			}
			testcloud.JSON(w, 200, `{"name":"actual secret response","secret_ref":"https://never-follow.test/secrets/other","consumers":[],"vendor":9007199254740993}`)
		case "GET /reverse/barbican/v1/secrets/secret-one/consumers":
			if r.Header.Get("X-Auth-Token") != "consumer-token" || r.URL.Query().Has("marker") {
				t.Error("consumer source changed", r.Header, r.URL)
			}
			switch r.URL.Query().Get("offset") {
			case "":
				if r.URL.Query().Has("limit") {
					t.Error("invented first-page limit", r.URL)
				}
				testcloud.JSON(w, 200, `{"consumers":[{"service":"volume","resource_type":"volume","resource_id":"volume-one"}],"next":"?offset=1&limit=1"}`)
			case "1":
				if r.URL.Query().Get("limit") != "1" {
					t.Error("server paging limit lost", r.URL)
				}
				testcloud.JSON(w, 200, `{"consumers":[]}`)
			default:
				t.Error("unexpected offset", r.URL)
			}
		case "POST /reverse/zaqar/v2/queues/jobs/subscriptions", "GET /reverse/zaqar/v2/queues/jobs/subscriptions/sub-one", "GET /reverse/zaqar/v2/queues/jobs/subscriptions", "DELETE /reverse/zaqar/v2/queues/jobs/subscriptions/sub-one":
			if r.Header.Get("X-Auth-Token") != "subscription-token" || r.Header.Get("Client-ID") != clientID || r.Header.Get("X-PROJECT-ID") != "request-project" {
				t.Error("subscription source changed", r.Header)
			}
			switch r.Method {
			case http.MethodPost:
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != `{"subscriber":"https://subscriber.test/events"}` {
					t.Error("TTL/default or envelope invented", string(body), err)
				}
				w.Header().Set("Location", "/reverse/zaqar/v2/queues/jobs/subscriptions")
				testcloud.JSON(w, 201, `{"subscription_id":"sub-one"}`)
			case http.MethodDelete:
				w.WriteHeader(204)
			case http.MethodGet:
				if r.URL.Path == "/reverse/zaqar/v2/queues/jobs/subscriptions/sub-one" {
					testcloud.JSON(w, 200, `{"id":"sub-one","subscriber":"https://subscriber.test/events","ttl":null}`)
				} else if r.URL.Query().Get("marker") == "" {
					if r.URL.Query().Has("limit") {
						t.Error("invented initial list limit", r.URL)
					}
					testcloud.JSON(w, 200, `{"subscriptions":[{"id":"sub-one","ttl":9007199254740993}]}`)
				} else {
					if r.URL.Query().Get("marker") != "sub-one" || r.URL.Query().Get("limit") != "1" {
						t.Error("Python row-count/marker paging changed", r.URL)
					}
					testcloud.JSON(w, 200, `{"subscriptions":[]}`)
				}
			}
		default:
			t.Error("unexpected parent lookup, response ref follow or request", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithInterface(gophercloud.AvailabilityInternal), sdk.WithMessagingClientID(clientID))
	if err != nil {
		t.Fatal(err)
	}
	keymanager, err := conn.KeyManagerV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	messaging, err := conn.MessagingV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	kmAgain, err := conn.KeyManager(ctx)
	if err != nil || kmAgain != keymanager {
		t.Fatal(kmAgain, err)
	}
	msgAgain, err := conn.Messaging(ctx)
	if err != nil || msgAgain != messaging || lookups.Load() != 2 {
		t.Fatal(msgAgain, err, lookups.Load())
	}
	kmClient, msgClient := keymanager.RawClient(), messaging.RawClient()
	if keymanager.SecretConsumers.RawClient() != kmClient || keymanager.Secrets.RawClient() != kmClient || messaging.Subscriptions.RawClient() != msgClient || messaging.Queues.RawClient() != msgClient || kmClient.ProviderClient != cloud.Provider || msgClient.ProviderClient != cloud.Provider {
		t.Fatal("SDK-owned and native APIs must share their authenticated source client")
	}
	kmClient.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	msgClient.ResourceBase = cloud.Server.URL + "/reverse/zaqar/v2/"
	kmClient.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	msgClient.MoreHeaders["X-SDK-Source"] = "shared"
	msgClient.MoreHeaders["X-PROJECT-ID"] = "configured-project"
	consumerScope, err := keymanager.SecretConsumers.InSecret(ctx, resource.ID("secret-one"))
	if err != nil {
		t.Fatal(err)
	}
	queueScope, err := messaging.Subscriptions.InQueue(ctx, "jobs")
	if err != nil || consumerScope.RawClient() != kmClient || queueScope.RawClient() != msgClient || calls.Load() != 0 {
		t.Fatal(queueScope, err, calls.Load())
	}
	cloud.Provider.SetToken("consumer-token")
	association := secretconsumers.ConsumerOpts{Service: "volume", ResourceType: "volume", ResourceID: "volume-one"}
	createdConsumer, err := consumerScope.Create(ctx, association)
	if err != nil || createdConsumer == nil || createdConsumer.Name != "actual secret response" || string(createdConsumer.Body["vendor"]) != "9007199254740993" || createdConsumer.StatusCode != 200 || createdConsumer.Header.Get("X-Request-ID") != "scoped-response" {
		t.Fatal(createdConsumer, err)
	}
	consumers, err := consumerScope.All(ctx)
	if err != nil || len(consumers) != 1 || consumers[0].ResourceID != "volume-one" {
		t.Fatal(consumers, err)
	}
	deletedConsumer, err := consumerScope.Delete(ctx, association)
	if err != nil || deletedConsumer == nil || deletedConsumer.Name != "actual secret response" || consumerScope.SecretID() != "secret-one" {
		t.Fatal(deletedConsumer, err)
	}
	cloud.Provider.SetToken("subscription-token")
	createdSubscription, err := queueScope.Create(ctx, subscriptions.CreateOpts{Subscriber: "https://subscriber.test/events"}, subscriptions.WithCreateProjectID("request-project"))
	if err != nil || createdSubscription == nil || createdSubscription.ID != "sub-one" || createdSubscription.StatusCode != 201 || createdSubscription.Header.Get("Location") != "/reverse/zaqar/v2/queues/jobs/subscriptions" {
		t.Fatal(createdSubscription, err)
	}
	fetched, err := queueScope.Get(ctx, "sub-one", subscriptions.WithGetProjectID("request-project"))
	if err != nil || fetched == nil || string(fetched.Body["ttl"]) != "null" {
		t.Fatal(fetched, err)
	}
	listed, err := queueScope.All(ctx, subscriptions.WithListProjectID("request-project"))
	if err != nil || len(listed) != 1 || string(listed[0].Body["ttl"]) != "9007199254740993" {
		t.Fatal(listed, err)
	}
	if err := queueScope.Delete(ctx, "sub-one", subscriptions.WithDeleteProjectID("request-project")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 9 || lookups.Load() != 2 || msgClient.MoreHeaders["Client-ID"] != clientID || msgClient.MoreHeaders["X-PROJECT-ID"] != "configured-project" {
		t.Fatal("unexpected HTTP work or shared header mutation", calls.Load(), lookups.Load(), msgClient.MoreHeaders)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := keymanager.SecretConsumers.InSecret(canceled, resource.ID("secret-one")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := messaging.Subscriptions.InQueue(canceled, "jobs"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 9 {
		t.Fatal("canceled scope started HTTP", calls.Load())
	}
}
