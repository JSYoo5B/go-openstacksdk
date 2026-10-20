package openstack

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func checkAddressConfiguration(t *testing.T, policy compute.ServerAddressPolicy, wantPrivate, wantForce bool) {
	t.Helper()
	service := compute.New(nil, compute.Dependencies{ServerAddresses: policy})
	server := &compute.Server{Status: "BUILD", AccessIPv4: "8.8.8.8", AccessIPv6: "2001:db8::1", Addresses: map[string]any{"private": []any{map[string]any{"version": 4, "addr": "10.0.0.1", "OS-EXT-IPS:type": "fixed"}}}}
	view, err := service.ExpandServerInterfaces(context.Background(), server, compute.WithLocalIPv6(true), compute.WithAddressReachability(false))
	wantAccess, wantV6 := "8.8.8.8", "2001:db8::1"
	if wantPrivate {
		wantAccess = "10.0.0.1"
	}
	if wantForce {
		wantV6 = ""
	}
	if err != nil || view.AccessIPv4 != wantAccess || view.PublicIPv6 != wantV6 {
		t.Fatalf("view=%+v err=%v wantAccess=%s wantV6=%s", view, err, wantAccess, wantV6)
	}
}

func TestCloudServerAddressConfigurationPriorityNormalizedBooleansAndValidation(t *testing.T) {
	t.Setenv("OS_FORCE_IPV4", "false")
	t.Setenv("OS_PREFER_IPV6", "true")
	for _, scenario := range []struct {
		name           string
		client, cloud  map[string]any
		force, private bool
	}{
		{"default", nil, nil, false, false},
		{"cloud force", nil, map[string]any{"force_ipv4": true}, true, false},
		{"cloud prefer wins", nil, map[string]any{"force_ipv4": false, "prefer_ipv6": false}, true, false},
		{"normalized false strings", nil, map[string]any{"private": "false", "force_ipv4": "false", "prefer_ipv6": "true"}, false, false},
		{"private normalized true", nil, map[string]any{"private": "TRUE", "force_ipv4": nil}, false, true},
		{"hyphen cloud alias", nil, map[string]any{"force-ipv4": true}, true, false},
		{"client alias overridden by env", map[string]any{"broken-ipv6": true}, nil, false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			policy, err := configuredServerAddresses(scenario.cloud, scenario.client)
			if err != nil {
				t.Fatal(err)
			}
			checkAddressConfiguration(t, policy, scenario.private, scenario.force)
		})
	}
	t.Setenv("OS_PREFER_IPV6", "false")
	policy, err := configuredServerAddresses(map[string]any{"force_ipv4": false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkAddressConfiguration(t, policy, false, false)
	for _, settings := range []map[string]any{{"private": 1}, {"floating_ip_source": 1}, {"floating_ip_source": "unknown"}, {"force_ipv4": []any{}}} {
		if _, err := configuredServerAddresses(settings, nil); err == nil {
			t.Fatalf("invalid settings accepted: %#v", settings)
		}
	}
}

func TestCloudServerAddressConfigurationUsesFrozenProfileBaseSecureLayers(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OS_FORCE_IPV4", "")
	t.Setenv("OS_PREFER_IPV6", "true")
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	writeCloudConfigFile(t, "clouds-public.yaml", "public-clouds:\n  provider:\n    private: true\n    force_ipv4: true\n    floating_ip_source: nova\n")
	writeCloudConfigFile(t, path, "client: {prefer_ipv6: false}\nclouds:\n  dev:\n    profile: provider\n    private: false\n")
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "client: {prefer_ipv6: true}\nclouds:\n  dev:\n    force_ipv4: false\n    floating_ip_source: null\n")
	configuration, err := loadCloudConfiguration("dev", []string{path})
	if err != nil {
		t.Fatal(err)
	}
	checkAddressConfiguration(t, configuration.serverAddresses, false, false)
	writeCloudConfigFile(t, path, "clouds: {dev: {private: true, force_ipv4: true}}\n")
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds: {dev: {private: true, force_ipv4: true}}\n")
	checkAddressConfiguration(t, configuration.serverAddresses, false, false)
}
