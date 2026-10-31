package cloudconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
)

func writeCloudConfigFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings(t *testing.T) {
	for _, tc := range []struct {
		name, base, secure, want string
	}{
		{"public inherited", "", "", "public-network"},
		{"base replaces", "    networks: [{name: base-network, default_interface: true}]\n", "", "base-network"},
		{"secure replaces", "    networks: [{name: base-network, default_interface: true}]\n", "    networks: [{name: secure-network, default_interface: true}]\n", "secure-network"},
		{"base empty", "    networks: []\n", "", ""},
		{"secure empty", "    networks: [{name: base-network, default_interface: true}]\n", "    networks: []\n", ""},
		{"secure changes profile", "", "    profile: alternate\n", "alternate-network"},
		{"secure clears metadata profile", "", "    profile: ''\n", ""},
		{"secure null metadata profile", "", "    profile: null\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			t.Chdir(cwd)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			for _, key := range []string{"OS_REGION_NAME", "OS_INTERFACE", "OS_AUTH_URL", "OS_USERNAME", "OS_PASSWORD", "OS_PROJECT_NAME", "OS_PROJECT_ID", "OS_USERID", "OS_DOMAIN_NAME", "OS_DOMAIN_ID"} {
				t.Setenv(key, "")
			}
			path := filepath.Join(t.TempDir(), "clouds.yaml")
			writeCloudConfigFile(t, path, "clouds:\n  dev:\n    profile: provider\n    region_name: region-file\n    interface: internal\n    verify: false\n    auth:\n      auth_url: https://identity.example/v3\n      username: snapshot-user\n      password: base-password\n      user_domain_name: Default\n"+tc.base)
			writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds:\n  dev:\n    auth: {password: secure-password}\n"+tc.secure)
			publicPath := filepath.Join(cwd, "clouds-public.yaml")
			writeCloudConfigFile(t, publicPath, "public-clouds:\n  provider:\n    auth: {project_name: inherited-project, project_domain_name: Default}\n    networks: [{name: public-network, default_interface: true}]\n  alternate:\n    auth: {project_name: inherited-project, project_domain_name: Default}\n    networks: [{name: alternate-network, default_interface: true}]\n  '':\n    networks: [{name: must-not-inherit, default_interface: true}]\n")
			configuration, err := Load("dev", []string{path}, nil)
			if err != nil || configuration.DefaultNetwork != tc.want {
				t.Fatalf("default=%q want=%q err=%v", configuration.DefaultNetwork, tc.want, err)
			}
			if configuration.NetworkRoles.DefaultNetworkSelector() != tc.want {
				t.Fatalf("role selector=%q want=%q", configuration.NetworkRoles.DefaultNetworkSelector(), tc.want)
			}
			// Changes after loading cannot change auth or either settings layer.
			writeCloudConfigFile(t, path, "clouds: {dev: {auth: {username: changed}}}\n")
			writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds: {dev: {auth: {password: changed}}}\n")
			writeCloudConfigFile(t, publicPath, "public-clouds: {provider: {networks: []}}\n")
			if configuration.NetworkRoles.DefaultNetworkSelector() != tc.want {
				t.Fatal("role policy changed after file mutation")
			}
			auth, endpoint, tlsConfig, err := clouds.Parse(configuration.ParseOptions...)
			if err != nil || auth.Username != "snapshot-user" || auth.Password != "secure-password" || auth.IdentityEndpoint != "https://identity.example/v3" || auth.TenantName != "inherited-project" || auth.Scope == nil || auth.Scope.ProjectName != "inherited-project" || auth.Scope.DomainName != "Default" || endpoint.Region != "region-file" || endpoint.Availability != gophercloud.AvailabilityInternal || tlsConfig == nil || !tlsConfig.InsecureSkipVerify {
				t.Fatalf("native snapshot settings incorrect; err=%v", err)
			}
		})
	}
}

func TestCloudNetworkConfigurationSearchAndReadFailures(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	envPath := filepath.Join(t.TempDir(), "clouds.yaml")
	writeCloudConfigFile(t, envPath, "clouds: {dev: {networks: [{name: env-network, default_interface: true}]}}\n")
	t.Setenv("OS_CLIENT_CONFIG_FILE", envPath)
	configuration, err := Load("dev", nil, nil)
	if err != nil || configuration.DefaultNetwork != "env-network" {
		t.Fatalf("default=%q err=%v", configuration.DefaultNetwork, err)
	}
	explicitPath := filepath.Join(t.TempDir(), "clouds.yaml")
	writeCloudConfigFile(t, explicitPath, "clouds: {dev: {networks: [{name: explicit-network, default_interface: true}]}}\n")
	configuration, err = Load("dev", []string{filepath.Join(t.TempDir(), "missing"), explicitPath, envPath}, nil)
	if err != nil || configuration.DefaultNetwork != "explicit-network" {
		t.Fatalf("default=%q err=%v", configuration.DefaultNetwork, err)
	}
	// The first opened file fails immediately; a directory is openable but not readable.
	if _, err := Load("dev", []string{t.TempDir(), explicitPath}, nil); err == nil {
		t.Fatal("read failure incorrectly fell through to the next file")
	}
	if _, err := Load("absent", []string{explicitPath}, nil); err == nil {
		t.Fatal("missing cloud accepted")
	}
	// Public files are irrelevant without a profile, even when malformed.
	writeCloudConfigFile(t, "clouds-public.yaml", "not: [valid")
	if _, err := Load("dev", []string{explicitPath}, nil); err != nil {
		t.Fatal(err)
	}
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(explicitPath), "secure.yaml"), "clouds: [invalid")
	if _, err := Load("dev", []string{explicitPath}, nil); err == nil {
		t.Fatal("malformed secure config ignored")
	}
	if _, err := Load("dev", []string{filepath.Join(t.TempDir(), "absent")}, nil); err == nil {
		t.Fatal("missing file accepted")
	}
	// With no explicit or environment file, cwd precedes XDG on every OS.
	t.Setenv("OS_CLIENT_CONFIG_FILE", "")
	writeCloudConfigFile(t, "clouds.yaml", "clouds: {dev: {networks: [{name: cwd-network, default_interface: true}]}}\n")
	xdgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "openstack", "clouds.yaml")
	writeCloudConfigFile(t, xdgPath, "clouds: {dev: {networks: [{name: xdg-network, default_interface: true}]}}\n")
	configuration, err = Load("dev", nil, nil)
	if err != nil || configuration.DefaultNetwork != "cwd-network" {
		t.Fatalf("default=%q err=%v", configuration.DefaultNetwork, err)
	}
	if err := os.Remove("clouds.yaml"); err != nil {
		t.Fatal(err)
	}
	configuration, err = Load("dev", nil, nil)
	if err != nil || configuration.DefaultNetwork != "xdg-network" {
		t.Fatalf("default=%q err=%v", configuration.DefaultNetwork, err)
	}
}

func TestCloudNetworkConfigurationValidation(t *testing.T) {
	for _, tc := range []struct {
		name, settings, want string
		invalid              bool
	}{
		{"absent", "{}", "", false},
		{"empty", "{networks: []}", "", false},
		{"no default", "{networks: [{name: private}]}", "", false},
		{"case insensitive true", "{networks: [{name: chosen, default_interface: 'TrUe'}]}", "chosen", false},
		{"string false", "{networks: [{name: ignored, default_interface: 'yes'}]}", "", false},
		{"no trim", "{networks: [{name: ignored, default_interface: ' true '}]}", "", false},
		{"null flag", "{networks: [{name: ignored, default_interface: null}]}", "", false},
		{"numeric flag", "{networks: [{name: chosen, default_interface: 1}]}", "", true},
		{"null list", "{networks: null}", "", true},
		{"wrong list", "{networks: chosen}", "", true},
		{"wrong entry", "{networks: [chosen]}", "", true},
		{"missing name", "{networks: [{default_interface: true}]}", "", true},
		{"blank name", "{networks: [{name: ' ', default_interface: true}]}", "", true},
		{"numeric name", "{networks: [{name: 10, default_interface: true}]}", "", true},
		{"duplicate default", "{networks: [{name: one, default_interface: true}, {name: two, default_interface: true}]}", "", true},
		{"duplicate NAT destination", "{networks: [{name: one, nat_destination: true}, {name: two, nat_destination: true}]}", "", true},
		{"multiple NAT sources permitted by loader", "{networks: [{name: one, nat_source: true}, {name: two, nat_source: true}]}", "", false},
		{"legacy external", "{external_network: legacy}", "legacy", false},
		{"legacy internal", "{internal_network: private}", "", false},
		{"legacy conflict", "{external_network: legacy, networks: []}", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "clouds.yaml")
			writeCloudConfigFile(t, path, "clouds:\n  dev: "+tc.settings+"\n")
			configuration, err := Load("dev", []string{path}, nil)
			if tc.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("err=%v", err)
				}
			} else if err != nil || configuration.DefaultNetwork != tc.want {
				t.Fatalf("default=%q want=%q err=%v", configuration.DefaultNetwork, tc.want, err)
			}
		})
	}
}

// TestCloudBuiltinVendorProfilesFeedRoleConfiguration covers the built-in
// branch of _expand_vendor_profile: a profile absent from clouds-public.yaml
// (or with no such file) resolves to the embedded pinned vendor profile, which
// supplies has_network, regions with network values and native authentication
// settings under the cloud's own settings. A clouds-public.yaml entry wins;
// shutdown and remote profiles are configuration errors, and an unknown
// profile without clouds-public.yaml keeps the native missing-file error.
func TestCloudBuiltinVendorProfilesFeedRoleConfiguration(t *testing.T) {
	write := func(t *testing.T, cloud, public string) string {
		t.Helper()
		cwd := t.TempDir()
		t.Chdir(cwd)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		path := filepath.Join(cwd, "clouds.yaml")
		writeCloudConfigFile(t, path, "clouds:\n  dev:\n"+cloud)
		if public != "" {
			writeCloudConfigFile(t, filepath.Join(cwd, "clouds-public.yaml"), public)
		}
		return path
	}
	t.Run("rackspace without public file", func(t *testing.T) {
		path := write(t, "    profile: rackspace\n    auth: {username: u, password: p}\n", "")
		configuration, err := Load("dev", []string{path}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if enabled, ok := configuration.ServiceFlags["network"]; !ok || enabled {
			t.Fatal(configuration.ServiceFlags)
		}
		if configuration.SecgroupSource == nil || *configuration.SecgroupSource != "None" {
			t.Fatal(configuration.SecgroupSource)
		}
		auth, _, _, err := clouds.Parse(configuration.ParseOptions...)
		if err != nil || auth.IdentityEndpoint != "https://identity.api.rackspacecloud.com/v2.0/" || auth.Username != "u" {
			t.Fatal(auth, err)
		}
	})
	t.Run("cloud settings override the profile", func(t *testing.T) {
		path := write(t, "    profile: rackspace\n    has_network: true\n    auth: {auth_url: 'https://identity.example/v3', username: u, password: p}\n", "")
		configuration, err := Load("dev", []string{path}, nil)
		if err != nil || !configuration.ServiceFlags["network"] {
			t.Fatal(configuration.ServiceFlags, err)
		}
		auth, _, _, err := clouds.Parse(configuration.ParseOptions...)
		if err != nil || auth.IdentityEndpoint != "https://identity.example/v3" {
			t.Fatal(auth, err)
		}
	})
	t.Run("public file entry wins", func(t *testing.T) {
		path := write(t, "    profile: rackspace\n    auth: {username: u, password: p}\n", "public-clouds:\n  rackspace:\n    auth: {auth_url: 'https://public.example/v3'}\n    has_network: true\n")
		configuration, err := Load("dev", []string{path}, nil)
		if err != nil || !configuration.ServiceFlags["network"] {
			t.Fatal(configuration.ServiceFlags, err)
		}
	})
	t.Run("public file without the profile falls back", func(t *testing.T) {
		path := write(t, "    profile: rackspace\n    auth: {username: u, password: p}\n", "public-clouds:\n  other: {}\n")
		configuration, err := Load("dev", []string{path}, nil)
		if err != nil || configuration.ServiceFlags["network"] {
			t.Fatal(configuration.ServiceFlags, err)
		}
	})
	t.Run("profile regions carry network values", func(t *testing.T) {
		path := write(t, "    profile: limestonenetworks\n", "")
		for _, region := range []string{"", "us-slc"} {
			var explicit *string
			if region != "" {
				explicit = &region
			}
			configuration, err := Load("dev", []string{path}, explicit)
			if err != nil || configuration.DefaultNetwork != "Public Internet" || configuration.NetworkRoles.DefaultNetworkSelector() != "Public Internet" {
				t.Fatal(region, configuration.DefaultNetwork, err)
			}
		}
		unknown := "us-nowhere"
		if _, err := Load("dev", []string{path}, &unknown); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	})
	for _, name := range []string{"betacloud", "ovh"} {
		t.Run(name+" rejected", func(t *testing.T) {
			path := write(t, "    profile: "+name+"\n", "")
			if _, err := Load("dev", []string{path}, nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	t.Run("unknown profile without public file", func(t *testing.T) {
		path := write(t, "    profile: nowhere\n", "")
		if _, err := Load("dev", []string{path}, nil); err == nil || errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	})
}

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
			policy, err := ServerAddresses(scenario.cloud, scenario.client)
			if err != nil {
				t.Fatal(err)
			}
			checkAddressConfiguration(t, policy, scenario.private, scenario.force)
		})
	}
	t.Setenv("OS_PREFER_IPV6", "false")
	policy, err := ServerAddresses(map[string]any{"force_ipv4": false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkAddressConfiguration(t, policy, false, false)
	for _, settings := range []map[string]any{{"private": 1}, {"floating_ip_source": 1}, {"floating_ip_source": "unknown"}, {"force_ipv4": []any{}}} {
		if _, err := ServerAddresses(settings, nil); err == nil {
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
	configuration, err := Load("dev", []string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkAddressConfiguration(t, configuration.ServerAddresses, false, false)
	writeCloudConfigFile(t, path, "clouds: {dev: {private: true, force_ipv4: true}}\n")
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds: {dev: {private: true, force_ipv4: true}}\n")
	checkAddressConfiguration(t, configuration.ServerAddresses, false, false)
}
