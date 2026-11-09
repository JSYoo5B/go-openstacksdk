package clusters

import "github.com/JSYoo5B/go-openstacksdk/request"

func WithCreateTimeout(value int) CreateOption {
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		config.Options.Timeout = request.Present(value)
		return nil
	})
}

func WithUpdateTimeout(value int) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Timeout = request.Present(value)
		return nil
	})
}
