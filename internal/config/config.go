package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	DefaultServeAddress   = ":8080"
	DefaultDatabaseDriver = "sqlite"
	DefaultDatabaseDSN    = "go-template.db"

	envPrefix                 = "GO_TEMPLATE"
	serveAddressConfigKey     = "serve.address"
	databaseDriverConfigKey   = "database.driver"
	databaseDSNConfigKey      = "database.dsn"
	instanceAdminKeyConfigKey = "instance.admin_key"
	serveAddressFlagName      = "address"
	databaseDriverFlagName    = "database-driver"
	databaseDSNFlagName       = "database-dsn"
)

type Config struct {
	Serve    Serve    `mapstructure:"serve"`
	Database Database `mapstructure:"database"`
	Instance Instance `mapstructure:"instance"`
}

type Serve struct {
	Address string `mapstructure:"address"`
}

type Database struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type Instance struct {
	AdminKey string `mapstructure:"admin_key"`
}

type Options struct {
	ConfigFile string
	FlagSet    *pflag.FlagSet
	KV         map[string]any
	Overrides  map[string]any
}

func (c Config) Validate() error {
	if err := c.Serve.Validate(); err != nil {
		return err
	}
	if err := c.Database.Validate(); err != nil {
		return err
	}
	return c.Instance.Validate()
}

func (c Serve) Validate() error {
	_, port, err := net.SplitHostPort(c.Address)
	if err != nil {
		return fmt.Errorf("%s must be a TCP host:port: %w", serveAddressConfigKey, err)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return fmt.Errorf("%s port must be a number from 0 to 65535: %w", serveAddressConfigKey, err)
	}
	return nil
}

func (c Database) Validate() error {
	if c.Driver != DefaultDatabaseDriver {
		return fmt.Errorf("%s must be %q", databaseDriverConfigKey, DefaultDatabaseDriver)
	}
	if strings.TrimSpace(c.DSN) == "" {
		return fmt.Errorf("%s is required", databaseDSNConfigKey)
	}
	return nil
}

func (c Instance) Validate() error {
	if strings.TrimSpace(c.AdminKey) == "" {
		return fmt.Errorf("%s is required", instanceAdminKeyConfigKey)
	}
	return nil
}

func Load(opts Options) (Config, error) {
	v := viper.New()
	v.SetDefault(serveAddressConfigKey, DefaultServeAddress)
	v.SetDefault(databaseDriverConfigKey, DefaultDatabaseDriver)
	v.SetDefault(databaseDSNConfigKey, DefaultDatabaseDSN)
	if err := v.BindEnv(instanceAdminKeyConfigKey); err != nil {
		return Config{}, fmt.Errorf("bind instance admin key environment: %w", err)
	}

	if err := v.MergeConfigMap(opts.KV); err != nil {
		return Config{}, fmt.Errorf("merge kv configuration: %w", err)
	}
	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
		if err := v.MergeInConfig(); err != nil {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	if opts.FlagSet != nil {
		if flag := opts.FlagSet.Lookup(serveAddressFlagName); flag != nil {
			if err := v.BindPFlag(serveAddressConfigKey, flag); err != nil {
				return Config{}, fmt.Errorf("bind serve address flag: %w", err)
			}
		}
		if flag := opts.FlagSet.Lookup(databaseDriverFlagName); flag != nil {
			if err := v.BindPFlag(databaseDriverConfigKey, flag); err != nil {
				return Config{}, fmt.Errorf("bind database driver flag: %w", err)
			}
		}
		if flag := opts.FlagSet.Lookup(databaseDSNFlagName); flag != nil {
			if err := v.BindPFlag(databaseDSNConfigKey, flag); err != nil {
				return Config{}, fmt.Errorf("bind database dsn flag: %w", err)
			}
		}
	}

	for key, value := range opts.Overrides {
		v.Set(key, value)
	}

	var cfg Config
	if err := v.UnmarshalExact(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode configuration: %w", err)
	}
	return cfg, nil
}
