package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMessagingSDKOwnedRegistryAndQueueScopeCapabilities(t *testing.T) {
	want := []collectionRecord{{Package: "github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions", Source: "sdk_owned", Model: "Subscription", Kind: "queue_subscription", Scope: "InQueue", Parent: "github.com/JSYoo5B/go-openstacksdk/messaging/v2/queues"}}
	var actual []collectionRecord
	for _, record := range sdkOwnedCollections {
		if strings.HasPrefix(record.Package, "github.com/JSYoo5B/go-openstacksdk/messaging/") {
			actual = append(actual, record)
		}
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("messaging registry invented a native or common capability: %#v", actual)
	}
	root := t.TempDir()
	g := generator{root: root, inventory: inventory{Operations: []operation{{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/messaging/v2/queues", Name: "List"}}}, collections: actual}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	read := func(path string) string {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, path := range []string{"messaging/v2/service_generated.go", "connection_services_generated.go"} {
		if _, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, path), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	registry := read("messaging/v2/service_generated.go")
	for _, part := range []string{"Subscriptions", "Queues", "github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions", "New(client)"} {
		if !strings.Contains(registry, part) {
			t.Fatalf("missing shared-client registry %q: %s", part, registry)
		}
	}
	connection := read("connection_services_generated.go")
	for _, part := range []string{"MessagingV2", "Messaging(ctx", "cachedService(ctx, c, Messaging"} {
		if !strings.Contains(connection, part) {
			t.Fatalf("missing cached authenticated facade %q: %s", part, connection)
		}
	}
	docs := read("messaging/v2/README.md")
	for _, part := range []string{"Subscriptions.InQueue", "고정 이름의 Create·Get·List/All·Delete", "subscriptions/api.go", "subscriptions/README.md", "Location", "행 수 limit", "Client-ID", "Resources·Find·Wait·Update 없음"} {
		if !strings.Contains(docs, part) {
			t.Fatalf("missing actual documented capability %q: %s", part, docs)
		}
	}
	for _, invented := range []string{"Subscriptions.Resources", "Subscriptions.Find(", "Subscriptions.Wait(", "Subscriptions.Update("} {
		if strings.Contains(docs, invented) {
			t.Fatalf("documentation invented unsupported capability %q: %s", invented, docs)
		}
	}
}
