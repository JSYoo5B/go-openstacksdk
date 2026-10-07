package gophercloudsdk

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"gophercloudsdk/resource"
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
			configuration, err := loadCloudConfiguration("dev", []string{path})
			if err != nil || configuration.defaultNetwork != tc.want {
				t.Fatalf("default=%q want=%q err=%v", configuration.defaultNetwork, tc.want, err)
			}
			if configuration.networkRoles.DefaultNetworkSelector() != tc.want {
				t.Fatalf("role selector=%q want=%q", configuration.networkRoles.DefaultNetworkSelector(), tc.want)
			}
			// Changes after loading cannot change auth or either settings layer.
			writeCloudConfigFile(t, path, "clouds: {dev: {auth: {username: changed}}}\n")
			writeCloudConfigFile(t, filepath.Join(filepath.Dir(path), "secure.yaml"), "clouds: {dev: {auth: {password: changed}}}\n")
			writeCloudConfigFile(t, publicPath, "public-clouds: {provider: {networks: []}}\n")
			if configuration.networkRoles.DefaultNetworkSelector() != tc.want {
				t.Fatal("role policy changed after file mutation")
			}
			auth, endpoint, tlsConfig, err := clouds.Parse(configuration.parseOptions...)
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
	configuration, err := loadCloudConfiguration("dev", nil)
	if err != nil || configuration.defaultNetwork != "env-network" {
		t.Fatalf("default=%q err=%v", configuration.defaultNetwork, err)
	}
	explicitPath := filepath.Join(t.TempDir(), "clouds.yaml")
	writeCloudConfigFile(t, explicitPath, "clouds: {dev: {networks: [{name: explicit-network, default_interface: true}]}}\n")
	configuration, err = loadCloudConfiguration("dev", []string{filepath.Join(t.TempDir(), "missing"), explicitPath, envPath})
	if err != nil || configuration.defaultNetwork != "explicit-network" {
		t.Fatalf("default=%q err=%v", configuration.defaultNetwork, err)
	}
	// The first opened file fails immediately; a directory is openable but not readable.
	if _, err := loadCloudConfiguration("dev", []string{t.TempDir(), explicitPath}); err == nil {
		t.Fatal("read failure incorrectly fell through to the next file")
	}
	if _, err := loadCloudConfiguration("absent", []string{explicitPath}); err == nil {
		t.Fatal("missing cloud accepted")
	}
	// Public files are irrelevant without a profile, even when malformed.
	writeCloudConfigFile(t, "clouds-public.yaml", "not: [valid")
	if _, err := loadCloudConfiguration("dev", []string{explicitPath}); err != nil {
		t.Fatal(err)
	}
	writeCloudConfigFile(t, filepath.Join(filepath.Dir(explicitPath), "secure.yaml"), "clouds: [invalid")
	if _, err := loadCloudConfiguration("dev", []string{explicitPath}); err == nil {
		t.Fatal("malformed secure config ignored")
	}
	if _, err := loadCloudConfiguration("dev", []string{filepath.Join(t.TempDir(), "absent")}); err == nil {
		t.Fatal("missing file accepted")
	}
	// With no explicit or environment file, cwd precedes XDG on every OS.
	t.Setenv("OS_CLIENT_CONFIG_FILE", "")
	writeCloudConfigFile(t, "clouds.yaml", "clouds: {dev: {networks: [{name: cwd-network, default_interface: true}]}}\n")
	xdgPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "openstack", "clouds.yaml")
	writeCloudConfigFile(t, xdgPath, "clouds: {dev: {networks: [{name: xdg-network, default_interface: true}]}}\n")
	configuration, err = loadCloudConfiguration("dev", nil)
	if err != nil || configuration.defaultNetwork != "cwd-network" {
		t.Fatalf("default=%q err=%v", configuration.defaultNetwork, err)
	}
	if err := os.Remove("clouds.yaml"); err != nil {
		t.Fatal(err)
	}
	configuration, err = loadCloudConfiguration("dev", nil)
	if err != nil || configuration.defaultNetwork != "xdg-network" {
		t.Fatalf("default=%q err=%v", configuration.defaultNetwork, err)
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
			configuration, err := loadCloudConfiguration("dev", []string{path})
			if tc.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("err=%v", err)
				}
			} else if err != nil || configuration.defaultNetwork != tc.want {
				t.Fatalf("default=%q want=%q err=%v", configuration.defaultNetwork, tc.want, err)
			}
		})
	}
}
