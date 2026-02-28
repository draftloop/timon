package configdaemon

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"net/url"
	"os"
	"path/filepath"
	"time"
	"timon/internal/log"
	"timon/internal/utils"
)

var (
	cfg     *Config
	cfgPath string
)

type Config struct {
	Daemon   DaemonConfig    `toml:"daemon"`
	Webhooks []WebhookConfig `toml:"webhook"`
}

type DaemonConfig struct {
	Hostname     string `toml:"hostname"`
	DataDir      string `toml:"data_dir"`
	LogDir       string `toml:"log_dir"`
	LogLevel     string `toml:"log_level"`
	PingInterval string `toml:"ping_interval"`

	PingIntervalDuration *time.Duration `toml:"-"`
}

type WebhookConfig struct {
	On      []string           `toml:"on"`
	URL     string             `toml:"url"`
	Cert    string             `toml:"cert"`
	Headers map[string]string  `toml:"headers"`
	Body    string             `toml:"body"`
	Retry   WebhookRetryConfig `toml:"retry"`
}

type WebhookRetryConfig struct {
	Attempts *int   `toml:"attempts"`
	Timeout  string `toml:"timeout"`
	Delay    string `toml:"delay"`

	TimeoutDuration time.Duration `toml:"-"`
	DelayDuration   time.Duration `toml:"-"`
}

var defaultConfig = Config{
	Daemon: DaemonConfig{
		Hostname:     defaultHostname(),
		DataDir:      "/etc/timon/",
		LogDir:       "/var/log/timon/",
		LogLevel:     "info",
		PingInterval: "5m",
	},
}

func defaultHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return hostname
}

func (config *Config) Validate() error {
	_, err := log.ParseLevel(config.Daemon.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid daemon.log_level — %v", err)
	}

	if err := os.MkdirAll(config.Daemon.DataDir, 0750); err != nil {
		return fmt.Errorf("failed to create data dir: %v", err)
	}
	if err := os.MkdirAll(config.Daemon.LogDir, 0750); err != nil {
		return fmt.Errorf("failed to create log dir: %v", err)
	}

	for i, wh := range config.Webhooks {
		if wh.URL == "" {
			return fmt.Errorf("webhook[%d]: url is required", i)
		}
		if _, err := url.Parse(wh.URL); err != nil {
			return fmt.Errorf("webhook[%d]: invalid url: %v", i, err)
		}

		if len(wh.On) == 0 {
			return fmt.Errorf("webhook[%d]: on is required", i)
		}

		if wh.Cert != "" {
			if _, err := os.Stat(wh.Cert); err != nil {
				return fmt.Errorf("webhook[%d]: cert file not found: %v", i, err)
			}
		}

		if wh.Retry.Attempts != nil && *wh.Retry.Attempts < 0 {
			return fmt.Errorf("webhook[%d]: retry.attempts must be >= 0", i)
		} else if wh.Retry.Attempts == nil {
			config.Webhooks[i].Retry.Attempts = utils.Ptr(5)
		} else if *wh.Retry.Attempts == 0 {
			config.Webhooks[i].Retry.Attempts = nil
		}

		if wh.Retry.Timeout == "" {
			config.Webhooks[i].Retry.Timeout = "10s"
			config.Webhooks[i].Retry.TimeoutDuration = 10 * time.Second
		} else if d, err := utils.ParseDuration(wh.Retry.Timeout); err != nil {
			return fmt.Errorf("webhook[%d]: invalid retry.timeout: %v", i, err)
		} else {
			config.Webhooks[i].Retry.TimeoutDuration = d
		}

		if wh.Retry.Delay == "" {
			config.Webhooks[i].Retry.Delay = "5s"
			config.Webhooks[i].Retry.DelayDuration = 5 * time.Second
		} else if d, err := utils.ParseDuration(wh.Retry.Delay); err != nil {
			return fmt.Errorf("webhook[%d]: invalid retry.delay: %v", i, err)
		} else {
			config.Webhooks[i].Retry.DelayDuration = d
		}
	}

	if config.Daemon.PingInterval != "" {
		if d, err := utils.ParseDuration(config.Daemon.PingInterval); err != nil {
			return fmt.Errorf("invalid daemon.ping_interval: %v", err)
		} else {
			config.Daemon.PingIntervalDuration = &d
		}
	}

	return nil
}

func LoadConfig() error {
	c := defaultConfig

	data, path, err := readFirstExistingConfig()
	if err != nil {
		return err
	} else if path == "" {
		cfg = &c
	} else {
		md, err := toml.Decode(string(data), &c)
		if err != nil {
			return fmt.Errorf("invalid configuration file %q: %w", path, err)
		}

		if undecoded := md.Undecoded(); len(undecoded) > 0 {
			return fmt.Errorf("unknown parameters in %q: %v", path, undecoded)
		}

		cfg = &c
		cfgPath = path
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration file %q: %v", path, err)
	}

	return nil
}

func GetConfig() *Config {
	return cfg
}

func GetConfigPath() string {
	return cfgPath
}

func readFirstExistingConfig() ([]byte, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	candidates := []string{
		"./timon.toml",
		filepath.Join(homeDir, ".config/timon/timon.toml"),
		"/etc/timon/timon.toml",
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, path, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("cannot read %s: %w", path, err)
		}
	}

	return nil, "", nil
}
