# Parent-scoped resources

Scope objects fix parent identities rather than repeatedly combining parent/child arguments. Explicit parent names resolve exactly with duplicate detection; ordinary explicit IDs are retained without fetching the parent. A later child request preserves the actual parent-not-found error.

Heat is an exception because routes require stack name and canonical ID: InStack with an ID fetches the stack identity; ForStack accepts a known StackIdentity without lookup. Connection quota helpers resolve project names through a separate Keystone client. Singleton quota access and separate project-quota lists are distinct APIs.

Scope Create/Update methods accept the same concrete options as their underlying APIs. Scope identity, actual response IDs, and embedded references remain distinct; do not follow arbitrary response links as request targets.

## Scope access

| Resource | Go access |
|---|---|
| DNS record set | `dns.RecordSets.InZone(ctx, zone)` |
| Nova interface | `compute.AttachInterfaces.InServer(ctx, server)` |
| Nova volume attachment | `compute.VolumeAttachments.InServer(ctx, server)` |
| Nova instance action | `compute.InstanceActions.InServer(ctx, server)` |
| Nova server tags | `compute.Tags.InServer(ctx, server)` |
| Nova project quota | `conn.ProjectQuotas(ctx, project)` / `CurrentProjectQuotas(ctx)` |
| Nova user quota | `projectScope.InUser(ctx, user)` |
| Cinder project quota | `conn.BlockStorageProjectQuotas(ctx, project)` / `CurrentBlockStorageProjectQuotas(ctx)` |
| Neutron project quota | `conn.NetworkProjectQuotas(ctx, project)` / `CurrentNetworkProjectQuotas(ctx)` |
| Octavia project quota | `conn.LoadBalancerProjectQuotas(ctx, project)` / `CurrentLoadBalancerProjectQuotas(ctx)` |
| Manila project quota | `conn.SharedFileSystemProjectQuotas(ctx, project)` / `CurrentSharedFileSystemProjectQuotas(ctx)` |
| Manila user/share type quota | `projectScope.InUser(ctx, user)` / `InShareType(ctx, shareType)` |
| Manila quota class | `shared.QuotaClassSets.InClass(ctx, "default")` |
| Designate project quota | `conn.DNSProjectQuotas(ctx, project)` / `CurrentDNSProjectQuotas(ctx)` |
| Magnum project/resource quota | `conn.ContainerInfraProjectQuotas(ctx, project)` → `ForResource(quotas.Cluster)` |
| Nova project limits | `conn.ProjectLimits(ctx, project)` / `CurrentProjectLimits(ctx)` |
| Cinder project limits | `conn.BlockStorageProjectLimits(ctx, project)` / `CurrentBlockStorageProjectLimits(ctx)` |
| Heat stack | `orchestration.Stacks.InStack(ctx, ref)` / `ForStack(identity)` |
| Heat stack resource | `orchestration.StackResources.InStack(ctx, ref)` / `ForStack(identity)` |
| Heat stack event | `orchestration.StackEvents.InStack(ctx, ref)` → `ForResource(name)` |
| Manila share access rule | `shared.ShareAccessRules.InShare(ctx, share)` |
| Trove database | `database.Databases.InInstance(ctx, instance)` |
| Trove user | `database.Users.InInstance(ctx, instance)` |
| Magnum node group | `magnum.NodeGroups.InCluster(ctx, cluster)` |
| Barbican secret consumer | `keymanager.SecretConsumers.InSecret(ctx, secret)` |
| Zaqar subscription | `messaging.Subscriptions.InQueue(ctx, queueName)` |
| Masakari host | `ha.Hosts.InSegment(ctx, segment)` |
| Masakari VMove | `ha.VMoves.InNotification(ctx, notification)` |
| Keystone application credential | `identity.ApplicationCredentials.InUser(ctx, user)` |
| Keystone access rule | `identity.ApplicationCredentials.AccessRules(ctx, user)` |
| Keystone EC2 credential | `identity.EC2Credentials.InUser(ctx, user)` |
| Glance image member | `image.Members.InImage(ctx, imageRef)` |
| Octavia pool member | `loadBalancer.Pools.Members(ctx, pool)` |
| Octavia L7 rule | `loadBalancer.L7Policies.Rules(ctx, policy)` |
| Neutron port forwarding | `network.PortForwarding.InFloatingIP(ctx, floatingIP)` |
| Neutron bandwidth limit rule | `network.QoSRules.BandwidthLimitRules(ctx, policy)` |
| Neutron DSCP marking rule | `network.QoSRules.DSCPMarkingRules(ctx, policy)` |
| Neutron minimum bandwidth rule | `network.QoSRules.MinimumBandwidthRules(ctx, policy)` |
| Swift object | `swift.Objects.InContainer(ctx, container)` |

See the corresponding package guide for child identifiers, supported operations, versions, statuses, and missing-resource policies. Scope construction alone does not imply that CRUD, name finding, status waits, or parent mutation are available for every resource.
