package clusters

import "github.com/JSYoo5B/go-openstacksdk/request"

func WithCreateMinSize(value int) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		copy := value
		config.Options.MinSize = &copy
		return nil
	}
}

func WithCreateMaxSize(value int) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		copy := value
		config.Options.MaxSize = &copy
		return nil
	}
}

func WithCreateDesiredCapacity(value int) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		copy := value
		config.Options.DesiredCapacity = &copy
		return nil
	}
}

func WithCreateTimeout(value int) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.Timeout = request.Present(value)
		return nil
	}
}

func WithCreateTimeoutNull() CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.Timeout = request.Null[int]()
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

func WithUpdateTimeout(value int) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Timeout = request.Present(value)
		return nil
	}
}

func WithUpdateTimeoutNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Timeout = request.Null[int]()
		return nil
	}
}
