package servers

import (
	"crypto/rsa"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type getPasswordOptions struct{ privateKey *rsa.PrivateKey }

type GetPasswordOption func(*getPasswordOptions) error

// WithGetPasswordPrivateKey decrypts the response using RSA PKCS#1 v1.5.
// Without this option, GetPassword returns the encrypted base64 value. A nil
// key restores that default when options are reused or composed.
func WithGetPasswordPrivateKey(key *rsa.PrivateKey) GetPasswordOption {
	return func(options *getPasswordOptions) error {
		if key != nil {
			if key.N == nil || key.D == nil || len(key.Primes) < 2 {
				return fmt.Errorf("%w: incomplete private key", resource.ErrInvalidOption)
			}
			for _, prime := range key.Primes {
				if prime == nil {
					return fmt.Errorf("%w: missing private key prime", resource.ErrInvalidOption)
				}
			}
			if err := key.Validate(); err != nil {
				return fmt.Errorf("%w: invalid private key: %v", resource.ErrInvalidOption, err)
			}
		}
		options.privateKey = key
		return nil
	}
}

func applyGetPasswordOptions(options ...GetPasswordOption) (getPasswordOptions, error) {
	var config getPasswordOptions
	for _, apply := range options {
		if apply == nil {
			return config, fmt.Errorf("%w: nil password option", resource.ErrInvalidOption)
		}
		if err := apply(&config); err != nil {
			return config, err
		}
	}
	return config, nil
}
