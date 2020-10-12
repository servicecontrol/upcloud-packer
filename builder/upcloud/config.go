package upcloud

import (
	"errors"
	"fmt"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/client"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/service"
	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/communicator"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
	"os"
	"time"
)

// Config represents the configuration for this builder
type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`

	// Required configuration values
	Username       string `mapstructure:"username"`
	Password       string `mapstructure:"password"`
	Zone           string `mapstructure:"zone"`
	StorageUUID    string `mapstructure:"storage_uuid"`
	TemplatePrefix string `mapstructure:"template_prefix"`

	// Optional configuration values
	StorageSize             int    `mapstructure:"storage_size"`
	RawStateTimeoutDuration string `mapstructure:"state_timeout_duration"`
	SSHPrivateKeyFile       string `mapstructure:"ssh_private_keyfile"`
	SSHPublicKeyFile        string `mapstructure:"ssh_public_keyfile"`

	StateTimeoutDuration time.Duration
	ctx                  interpolate.Context
}

// GetService returns a service object using the credentials specified in the configuration
func (c *Config) GetService() *service.Service {
	t := client.New(c.Username, c.Password)
	return service.New(t)
}

// NewConfig creates a new configuration, setting default values and validating it along the way
func NewConfig(raws ...interface{}) (*Config, error) {
	c := new(Config)

	err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
	}, raws...)

	if err != nil {
		return nil, err
	}

	// Assign default values if possible
	c.Comm.SSHUsername = "root"

	if c.StorageSize == 0 {
		c.StorageSize = 30
	}

	if c.RawStateTimeoutDuration == "" {
		c.RawStateTimeoutDuration = "5m"
	}

	// Validation
	var errs *packer.MultiError
	errs = packer.MultiErrorAppend(errs, c.Comm.Prepare(&c.ctx)...)

	if c.SSHPrivateKeyFile != "" && c.SSHPublicKeyFile != "" {
		if _, err := os.Stat(c.SSHPrivateKeyFile); os.IsNotExist(err) {
			errs = packer.MultiErrorAppend(
				errs, errors.New("ssh_private_keyfile does not exist"))
		}
		if _, err := os.Stat(c.SSHPublicKeyFile); os.IsNotExist(err) {
			errs = packer.MultiErrorAppend(
				errs, errors.New("ssh_public_keyfile does not exist"))
		}
	}

	// Check for required configurations that will display errors if not set
	if c.Username == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("\"username\" must be specified"))
	}

	if c.Password == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("\"password\" must be specified"))
	}

	if c.Zone == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("\"zone\" must be specified"))
	}

	if c.StorageUUID == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("\"storage_uuid\" must be specified"))
	}

	stateTimeout, err := time.ParseDuration(c.RawStateTimeoutDuration)
	if err != nil {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Failed to parse state_timeout_duration: %s", err))
	}
	c.StateTimeoutDuration = stateTimeout

	if len(errs.Errors) > 0 {
		return nil, errs
	}

	return c, nil
}
