package nodes

import "github.com/JSYoo5B/go-openstacksdk/request"

func WithCreateClusterID(value string) CreateOption {
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		config.Options.ClusterID = request.Present(value)
		return nil
	})
}

func WithCreateRole(value string) CreateOption {
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		config.Options.Role = request.Present(value)
		return nil
	})
}

func WithUpdateName(value string) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Name = request.Present(value)
		return nil
	})
}

func WithUpdateProfileID(value string) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.ProfileID = request.Present(value)
		return nil
	})
}

func WithUpdateRole(value string) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Role = request.Present(value)
		return nil
	})
}

func WithUpdateTainted(value bool) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Tainted = request.Present(value)
		return nil
	})
}
