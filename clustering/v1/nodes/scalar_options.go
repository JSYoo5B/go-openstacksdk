package nodes

import "github.com/JSYoo5B/go-openstacksdk/request"

func WithCreateClusterID(value string) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.ClusterID = request.Present(value)
		return nil
	}
}
func WithCreateClusterIDNull() CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.ClusterID = request.Null[string]()
		return nil
	}
}
func WithCreateRole(value string) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.Role = request.Present(value)
		return nil
	}
}
func WithCreateRoleNull() CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.Role = request.Null[string]()
		return nil
	}
}
func WithUpdateName(value string) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Name = request.Present(value)
		return nil
	}
}
func WithUpdateNameNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Name = request.Null[string]()
		return nil
	}
}
func WithUpdateProfileID(value string) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.ProfileID = request.Present(value)
		return nil
	}
}
func WithUpdateProfileIDNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.ProfileID = request.Null[string]()
		return nil
	}
}
func WithUpdateRole(value string) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Role = request.Present(value)
		return nil
	}
}
func WithUpdateRoleNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Role = request.Null[string]()
		return nil
	}
}
func WithUpdateTainted(value bool) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Tainted = request.Present(value)
		return nil
	}
}
func WithUpdateTaintedNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Tainted = request.Null[bool]()
		return nil
	}
}
