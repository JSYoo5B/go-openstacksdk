package openstack

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
)

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
		configuration, err := loadCloudConfiguration("dev", []string{path})
		if err != nil {
			t.Fatal(err)
		}
		if enabled, ok := configuration.serviceFlags["network"]; !ok || enabled {
			t.Fatal(configuration.serviceFlags)
		}
		if configuration.secgroupSource == nil || *configuration.secgroupSource != "None" {
			t.Fatal(configuration.secgroupSource)
		}
		auth, _, _, err := clouds.Parse(configuration.parseOptions...)
		if err != nil || auth.IdentityEndpoint != "https://identity.api.rackspacecloud.com/v2.0/" || auth.Username != "u" {
			t.Fatal(auth, err)
		}
	})
	t.Run("cloud settings override the profile", func(t *testing.T) {
		path := write(t, "    profile: rackspace\n    has_network: true\n    auth: {auth_url: 'https://identity.example/v3', username: u, password: p}\n", "")
		configuration, err := loadCloudConfiguration("dev", []string{path})
		if err != nil || !configuration.serviceFlags["network"] {
			t.Fatal(configuration.serviceFlags, err)
		}
		auth, _, _, err := clouds.Parse(configuration.parseOptions...)
		if err != nil || auth.IdentityEndpoint != "https://identity.example/v3" {
			t.Fatal(auth, err)
		}
	})
	t.Run("public file entry wins", func(t *testing.T) {
		path := write(t, "    profile: rackspace\n    auth: {username: u, password: p}\n", "public-clouds:\n  rackspace:\n    auth: {auth_url: 'https://public.example/v3'}\n    has_network: true\n")
		configuration, err := loadCloudConfiguration("dev", []string{path})
		if err != nil || !configuration.serviceFlags["network"] {
			t.Fatal(configuration.serviceFlags, err)
		}
	})
	t.Run("public file without the profile falls back", func(t *testing.T) {
		path := write(t, "    profile: rackspace\n    auth: {username: u, password: p}\n", "public-clouds:\n  other: {}\n")
		configuration, err := loadCloudConfiguration("dev", []string{path})
		if err != nil || configuration.serviceFlags["network"] {
			t.Fatal(configuration.serviceFlags, err)
		}
	})
	t.Run("profile regions carry network values", func(t *testing.T) {
		path := write(t, "    profile: limestonenetworks\n", "")
		for _, region := range []string{"", "us-slc"} {
			var explicit *string
			if region != "" {
				explicit = &region
			}
			configuration, err := loadCloudRegionConfiguration("dev", []string{path}, explicit)
			if err != nil || configuration.defaultNetwork != "Public Internet" || configuration.networkRoles.DefaultNetworkSelector() != "Public Internet" {
				t.Fatal(region, configuration.defaultNetwork, err)
			}
		}
		unknown := "us-nowhere"
		if _, err := loadCloudRegionConfiguration("dev", []string{path}, &unknown); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	})
	for _, name := range []string{"betacloud", "ovh"} {
		t.Run(name+" rejected", func(t *testing.T) {
			path := write(t, "    profile: "+name+"\n", "")
			if _, err := loadCloudConfiguration("dev", []string{path}); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	t.Run("unknown profile without public file", func(t *testing.T) {
		path := write(t, "    profile: nowhere\n", "")
		if _, err := loadCloudConfiguration("dev", []string{path}); err == nil || errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	})
}
