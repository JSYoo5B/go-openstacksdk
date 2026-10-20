package compute

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// BackupServerRequest contains the owned createBackup inputs. Nova validates
// the supplied backup name/type/rotation, including empty strings or zero.
type BackupServerRequest = servers.BackupActionRequest

// BackupServerOpts owns the optional Proxy wait controls. Timeout nil defaults
// to 120 seconds here, rather than Cloud wait_for_image's standalone 3600.
// ImageService is a concrete direct-client override; Connection binds it lazily.
type BackupServerOpts struct {
	Wait          bool
	Timeout       *time.Duration
	Unlimited     bool
	PollInterval  *time.Duration
	ActionOptions []servers.BackupActionOption
	ImageService  *image.Service
}
type BackupServerOption func(*BackupServerOpts) error

func WithBackupServerWait(enabled bool) BackupServerOption {
	return func(c *BackupServerOpts) error { c.Wait = enabled; return nil }
}
func WithBackupServerTimeout(value time.Duration) BackupServerOption {
	return func(c *BackupServerOpts) error { owned := value; c.Timeout = &owned; c.Unlimited = false; return nil }
}
func WithBackupServerUnlimitedWait() BackupServerOption {
	return func(c *BackupServerOpts) error { c.Timeout = nil; c.Unlimited = true; return nil }
}
func WithBackupServerPollInterval(value time.Duration) BackupServerOption {
	return func(c *BackupServerOpts) error { owned := value; c.PollInterval = &owned; return nil }
}
func WithBackupServerActionOptions(values ...servers.BackupActionOption) BackupServerOption {
	owned := slices.Clone(values)
	return func(c *BackupServerOpts) error { c.ActionOptions = append(c.ActionOptions, owned...); return nil }
}

// WithBackupImageService supplies a concrete Glance service for direct clients.
// Passing nil restores lazy Connection dependency selection.
func WithBackupImageService(value *image.Service) BackupServerOption {
	return func(c *BackupServerOpts) error { c.ImageService = value; return nil }
}

func copyBackupServerOptions(value BackupServerOpts) BackupServerOpts {
	if value.Timeout != nil {
		owned := *value.Timeout
		value.Timeout = &owned
	}
	if value.PollInterval != nil {
		owned := *value.PollInterval
		value.PollInterval = &owned
	}
	value.ActionOptions = slices.Clone(value.ActionOptions)
	return value
}

// BackupServerResult separates Nova's action acceptance from Glance lookup and
// active observation. A later failure never resubmits or deletes the backup.
type BackupServerResult struct {
	Action *servers.BackupActionResult
	Lookup *image.CloudImageRecordResult
	Wait   *image.ImageRecordCloudWaitResult
	Image  *image.ImageRecord
}

// captureBackupImage fixes concrete selected Glance bindings across both Get
// and Wait. Child APIs retain their own header/model/physical response guards.
func captureBackupImage(service *image.Service) (func(context.Context) error, error) {
	if service == nil || service.API == nil || service.API.Images == nil || service.RawClient() == nil || service.RawClient().ProviderClient == nil {
		return nil, invalid("backup image service is required")
	}
	api, collection, child, client := service.API, service.Images, service.API.Images, service.RawClient()
	provider, endpoint, base, kind, version := client.ProviderClient, client.Endpoint, client.ResourceBase, client.Type, client.Microversion
	var observed error
	var mu sync.Mutex
	return func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if observed == nil && (service.API != api || service.Images != collection || api.Images != child || service.RawClient() != client || api.RawClient() != client || child.RawClient() != client || client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base || client.Type != kind || client.Microversion != version) {
			observed = invalid("backup image source changed")
		}
		return cloudread.ContextError(ctx, observed)
	}, nil
}

// BackupServer implements Proxy backup_server: createBackup, Cloud get_image,
// then optional Cloud wait_for_image. Wait defaults false and timeout defaults
// 120 seconds; exact "active" succeeds and exact "error" fails in the Cloud
// helper. Image absence is nil for wait=false and an invalid seed for wait=true.
func (s *Service) BackupServer(ctx context.Context, serverID string, input BackupServerRequest, options ...BackupServerOption) (*BackupServerResult, error) {
	const op = "BackupServer"
	fail := func(result *BackupServerResult, err error) (*BackupServerResult, error) {
		return result, request.Wrap(op, "servers", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if s == nil || s.API == nil || s.API.Servers == nil || s.Servers == nil || s.client == nil || s.client.ProviderClient == nil {
		return fail(nil, invalid("backup compute service is required"))
	}
	computeGuard := s.captureAutomaticComputeSource()
	originalCtx := ctx
	outer := rest.OperationGuard(ctx)
	var imageGuard func(context.Context) error
	check := func(checkCtx context.Context) error {
		var ancestor, child error
		if outer != nil {
			ancestor = outer(originalCtx)
		}
		if imageGuard != nil {
			child = imageGuard(checkCtx)
		}
		return errors.Join(computeGuard(checkCtx), ancestor, child)
	}
	if err := check(ctx); err != nil {
		return fail(nil, err)
	}
	// Capture the lazy dependency before any caller option. It is resolved only
	// after the Nova action, matching the Source's cross-service request order.
	imageResolver := s.Servers.dependencies.BackupImages
	config := BackupServerOpts{}
	for _, apply := range slices.Clone(options) {
		if err := check(ctx); err != nil {
			return fail(nil, err)
		}
		if apply == nil {
			return fail(nil, invalid("nil backup server option"))
		}
		owned := copyBackupServerOptions(config)
		err := apply(&owned)
		config = copyBackupServerOptions(owned)
		if err = errors.Join(err, check(ctx)); err != nil {
			return fail(nil, err)
		}
	}
	if config.Wait && config.Unlimited && config.Timeout != nil {
		return fail(nil, invalid("backup wait cannot have both unlimited and finite timeout"))
	}
	if config.Wait && config.PollInterval != nil && *config.PollInterval <= 0 {
		return fail(nil, invalid("backup poll interval must be positive"))
	}
	selected := config.ImageService
	var err error
	if selected != nil {
		imageGuard, err = captureBackupImage(selected)
		if err != nil {
			return fail(nil, err)
		}
	}
	opctx := rest.WithOperationGuard(ctx, check)
	result := &BackupServerResult{}
	result.Action, err = s.API.Servers.CreateBackupAction(opctx, serverID, input, config.ActionOptions...)
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(result, err)
	}
	if selected == nil {
		if imageResolver == nil {
			return fail(result, invalid("backup image dependency is required; direct clients can use WithBackupImageService"))
		}
		selected, err = imageResolver(opctx)
		if err = errors.Join(err, check(opctx)); err != nil {
			return fail(result, err)
		}
		imageGuard, err = captureBackupImage(selected)
		if err = errors.Join(err, check(opctx)); err != nil {
			return fail(result, err)
		}
	}
	result.Lookup, err = selected.GetCloudImageRecord(opctx, result.Action.ImageID)
	if result.Lookup != nil {
		result.Image = result.Lookup.Image
	}
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(result, err)
	}
	if !config.Wait {
		return result, nil
	}
	timeout := 120 * time.Second
	if config.Timeout != nil {
		timeout = *config.Timeout
	}
	waitOptions := []image.ImageRecordCloudWaitOption{image.WithImageRecordCloudWaitTimeout(timeout)}
	if config.Unlimited {
		waitOptions = []image.ImageRecordCloudWaitOption{image.WithImageRecordCloudWaitUnlimited()}
	}
	if config.PollInterval != nil {
		waitOptions = append(waitOptions, image.WithImageRecordCloudWaitPollInterval(*config.PollInterval))
	}
	result.Wait, err = selected.WaitForCloudImageRecord(opctx, result.Image, waitOptions...)
	if result.Wait != nil && result.Wait.Last != nil {
		result.Image = result.Wait.Last
	}
	if result.Wait != nil && result.Wait.Image != nil {
		result.Image = result.Wait.Image
	}
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(result, err)
	}
	return result, nil
}
