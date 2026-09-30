package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type serviceSpec struct{ name, constant, version string }

var serviceSpecs = map[string]serviceSpec{
	"baremetal":              {"BareMetal", "BareMetal", "v1"},
	"baremetalintrospection": {"BareMetalIntrospection", "BareMetalIntrospection", "v1"},
	"blockstorage":           {"BlockStorage", "BlockStorage", "v3"},
	"compute":                {"Compute", "Compute", "v2"},
	"container":              {"Container", "Container", "v1"},
	"containerinfra":         {"ContainerInfra", "ContainerInfra", "v1"},
	"db":                     {"Database", "Database", "v1"},
	"dns":                    {"DNS", "DNS", "v2"},
	"identity":               {"Identity", "Identity", "v3"},
	"image":                  {"Image", "Image", "v2"},
	"keymanager":             {"KeyManager", "KeyManager", "v1"},
	"loadbalancer":           {"LoadBalancer", "LoadBalancer", "v2"},
	"messaging":              {"Messaging", "Messaging", "v2"},
	"metric":                 {"Metric", "Metric", "v1"},
	"network":                {"Network", "Network", "v2"},
	"objectstorage":          {"ObjectStorage", "ObjectStorage", "v1"},
	"orchestration":          {"Orchestration", "Orchestration", "v1"},
	"placement":              {"Placement", "Placement", "v1"},
	"reservation":            {"Reservation", "Reservation", "v1"},
	"sharedfilesystems":      {"SharedFileSystem", "SharedFileSystem", "v2"},
	"workflow":               {"Workflow", "Workflow", "v2"},
}

func registryField(path string) string {
	words := map[string]string{
		"apiversions": "APIVersions", "servergroups": "ServerGroups", "keypairs": "KeyPairs", "secgroups": "SecurityGroups", "instanceactions": "InstanceActions", "remoteconsoles": "RemoteConsoles", "attachinterfaces": "AttachInterfaces", "volumeattach": "VolumeAttachments", "availabilityzones": "AvailabilityZones", "quotasets": "QuotaSets",
		"extensions": "", "layer3": "", "bgp": "BGP", "qos": "QoS", "fwaas_v2": "Firewall", "vpnaas": "VPN", "taas": "TaaS", "networkipavailabilities": "NetworkIPAvailabilities", "rbacpolicies": "RBACPolicies", "subnetpools": "SubnetPools", "addressscopes": "AddressScopes", "floatingips": "FloatingIPs", "extraroutes": "ExtraRoutes", "portforwarding": "PortForwarding", "bgpvpns": "BGPVPNs", "attributestags": "AttributeTags", "tapmirrors": "TapMirrors", "addressgroups": "AddressGroups", "endpointgroups": "EndpointGroups", "siteconnections": "SiteConnections", "ipsecpolicies": "IPsecPolicies", "ikepolicies": "IKEPolicies", "ruletypes": "RuleTypes",
		"flavorprofiles": "FlavorProfiles", "l7policies": "L7Policies", "l7rules": "L7Rules", "loadbalancers": "LoadBalancers", "volumetypes": "VolumeTypes", "volumetransfers": "VolumeTransfers", "volumegroups": "VolumeGroups", "group_types": "GroupTypes", "recordsets": "RecordSets", "imagedata": "ImageData", "imageimport": "ImageImport", "roleassignments": "RoleAssignments", "registeredlimits": "RegisteredLimits", "applicationcredentials": "ApplicationCredentials", "domainconfigs": "DomainConfigs", "domainroles": "DomainRoles", "projectroles": "ProjectRoles", "userpassword": "UserPassword", "trusts": "Trusts", "portgroups": "PortGroups", "bulkdelete": "BulkDelete", "softwareconfigs": "SoftwareConfigs", "softwaredeployments": "SoftwareDeployments", "stacktemplates": "StackTemplates", "sharetypes": "ShareTypes", "sharenetworks": "ShareNetworks", "shareinstances": "ShareInstances", "sharereplicas": "ShareReplicas", "sharesnapshots": "ShareSnapshots", "sharegroups": "ShareGroups", "sharegroup_types": "ShareGroupTypes", "sharegroup_snapshots": "ShareGroupSnapshots", "securityservices": "SecurityServices", "shareaccessrules": "ShareAccessRules", "resourceproviders": "ResourceProviders", "resourceclasses": "ResourceClasses", "allocationcandidates": "AllocationCandidates", "crontriggers": "CronTriggers", "clustertemplates": "ClusterTemplates", "clusterstacks": "ClusterStacks",
	}
	if path == "extensions" {
		return "Extensions"
	}
	var name strings.Builder
	for _, part := range strings.Split(path, "/") {
		if word, ok := words[part]; ok {
			name.WriteString(word)
		} else {
			name.WriteString(title(part))
		}
	}
	return name.String()
}

func writeGo(root, path string, source []byte) error {
	formatted, err := format.Source(source)
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	target := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.WriteFile(target, formatted, 0644)
}

func (g *generator) generateServices() error {
	groups := map[string]map[string]bool{}
	for _, operation := range g.inventory.Operations {
		path := strings.TrimPrefix(operation.SDKPackage, "gophercloudsdk/")
		parts := strings.Split(path, "/")
		if len(parts) < 3 || !strings.HasPrefix(parts[1], "v") {
			continue
		}
		if _, ok := serviceSpecs[parts[0]]; !ok {
			continue
		}
		key := strings.Join(parts[:2], "/")
		if groups[key] == nil {
			groups[key] = map[string]bool{}
		}
		groups[key][strings.Join(parts[2:], "/")] = true
	}
	keys := []string{}
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var connection bytes.Buffer
	connection.WriteString("// Code generated by sdkgen; DO NOT EDIT.\npackage gophercloudsdk\nimport(\n\"context\"\n")
	for i, key := range keys {
		fmt.Fprintf(&connection, "service%d %q\n", i, "gophercloudsdk/"+key)
	}
	connection.WriteString(")\n")
	for i, key := range keys {
		parts := strings.Split(key, "/")
		spec := serviceSpecs[parts[0]]
		paths := []string{}
		for path := range groups[key] {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		var registry bytes.Buffer
		fmt.Fprintf(&registry, "// Code generated by sdkgen; DO NOT EDIT.\npackage %s\nimport(\ngophercloud %q\n", parts[1], upstreamModule)
		for j, path := range paths {
			fmt.Fprintf(&registry, "resource%d %q\n", j, "gophercloudsdk/"+key+"/"+path)
		}
		registry.WriteString(")\n// Service shares one authenticated client across its resource APIs.\ntype Service struct{client *gophercloud.ServiceClient\n")
		for j, path := range paths {
			fmt.Fprintf(&registry, "%s *resource%d.API\n", registryField(path), j)
		}
		registry.WriteString("}\nfunc New(client *gophercloud.ServiceClient)*Service{return &Service{client:client,\n")
		for j, path := range paths {
			fmt.Fprintf(&registry, "%s:resource%d.New(client),\n", registryField(path), j)
		}
		registry.WriteString("}}\nfunc(s *Service)RawClient()*gophercloud.ServiceClient{return s.client}\n")
		if err := writeGo(g.root, key+"/service_generated.go", registry.Bytes()); err != nil {
			return err
		}
		method := spec.name + strings.ToUpper(parts[1][:1]) + parts[1][1:]
		fmt.Fprintf(&connection, "// %s returns a cached, authenticated %s API proxy.\nfunc(c *Connection)%s(ctx context.Context)(*service%d.Service,error){return cachedService(ctx,c,%s,%q,service%d.New)}\n", method, key, method, i, spec.constant, parts[1], i)
		if parts[1] == spec.version && parts[0] != "compute" && parts[0] != "network" && parts[0] != "image" && parts[0] != "blockstorage" {
			fmt.Fprintf(&connection, "func(c *Connection)%s(ctx context.Context)(*service%d.Service,error){return c.%s(ctx)}\n", spec.name, i, method)
		}
	}
	return writeGo(g.root, "connection_services_generated.go", connection.Bytes())
}
