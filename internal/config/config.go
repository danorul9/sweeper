package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Ignore    []string `mapstructure:"ignore"`
	SafeMode  bool     `mapstructure:"safe_mode"`
	ScanMode  ScanMode
}

type ScanMode int

const (
	ModeSafe ScanMode = iota
	ModeAggressive
	ModeReclaim
)

func (m ScanMode) String() string {
	switch m {
	case ModeSafe:
		return "safe"
	case ModeAggressive:
		return "aggressive"
	case ModeReclaim:
		return "reclaim"
	default:
		return "unknown"
	}
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	sweeperDir := filepath.Join(configDir, "sweeper")
	v.AddConfigPath(sweeperDir)

	v.SetDefault("safe_mode", false)
	v.SetDefault("ignore", []string{})

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	_ = os.MkdirAll(sweeperDir, 0755)

	return &cfg, nil
}

func CacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir, "sweeper")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func AppSupportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "Application Support", "Sweeper")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}
