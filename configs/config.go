package configs

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/kennethpensopay/dbmservice/internal/modinfo"
	debug "github.com/kennethpensopay/go-debug"
	"log"
	"strings"
	"time"
)

type ServiceConfig struct {
	Name    string   `default:""`
	Address string   `default:"localhost"`
	Port    int      `default:"0"`
	Tags    []string `default:"go,service"`
}

type DiscoveryConfig struct {
	Address         string        `default:"127.0.0.1:8500"`
	Scheme          string        `default:"http"`
	HealthCheckTTL  time.Duration `envconfig:"CONSUL_HEALTH_CHECK_TTL" default:"10s"`
	EnablePlanWatch bool          `split_words:"true" default:"false"`
}

type config struct {
	Service ServiceConfig
	Consul  DiscoveryConfig
	Debug   bool `default:"false"`
}

func LoadConfigs() (service *ServiceConfig, discovery *DiscoveryConfig) {
	_ = godotenv.Load()
	var cfg config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("Could not load environment configuration.")
	}

	if cfg.Debug {
		debug.EnableDebug()
	}

	if strings.TrimSpace(cfg.Service.Name) == "" {
		cfg.Service.Name = strings.TrimSpace(strings.ToLower(modinfo.ModInfo().ModuleNameAsMD5Sum() + ".services.local"))
		debug.Debugf("No Service Name found in env. Set to: %s", cfg.Service.Name)
	}

	cfg.Service.Address = strings.TrimSpace(strings.ToLower(cfg.Service.Address))

	if cfg.Service.Port < 0 || cfg.Service.Port > 65535 {
		debug.Debugf("Service Port was set to %d which is out of the valid port range (0 -> 65535). Port has been set to 0, which will utilize a random available port.", cfg.Service.Port)
		cfg.Service.Port = 0
	}

	cfg.Consul.Address = strings.TrimSpace(strings.ToLower(cfg.Consul.Address))
	cfg.Consul.Scheme = strings.TrimSpace(cfg.Consul.Scheme)

	return &cfg.Service, &cfg.Consul
}
