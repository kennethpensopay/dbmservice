package discovery

import (
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/kennethpensopay/dbmservice/configs"
	debug "github.com/kennethpensopay/go-debug"
	"log"
	"strings"
	"time"
)

type ServiceDiscovery struct {
	HealthCheckTTL     time.Duration
	IsPlanWatchEnabled bool
	Config             *consul.Config
	client             *consul.Client
}

func NewServiceDiscovery(cfg *configs.DiscoveryConfig) *ServiceDiscovery {
	serviceDiscovery := &ServiceDiscovery{
		HealthCheckTTL:     cfg.HealthCheckTTL,
		IsPlanWatchEnabled: cfg.EnablePlanWatch,
		Config:             createConsulConfig(cfg),
	}

	client, err := consul.NewClient(serviceDiscovery.Config)
	if err != nil {
		log.Fatalf("Could not create new Consul Service Client: %v", err)
	}

	serviceDiscovery.client = client
	return serviceDiscovery
}

func (d *ServiceDiscovery) RegisterService(service *consul.AgentServiceRegistration) error {
	if d.IsPlanWatchEnabled {
		debug.Debug("Consul Plan Watch has been enabled.")

		plan, err := watch.Parse(map[string]any{
			"type":        "service",
			"service":     service.Name,
			"passingonly": true,
		})
		if err != nil {
			log.Fatal(err)
		}

		plan.HybridHandler = func(index watch.BlockingParamVal, result any) {
			switch msg := result.(type) {
			case []*consul.ServiceEntry:
				if msgLength := len(msg); msgLength > 0 {
					if svc := msg[msgLength-1].Service; svc.ID != service.ID {
						log.Printf("Service instance joined: %v", svc)
					}
				}
			}
		}

		debug.Debugf("Plan: %v", plan)

		go func() {
			if runErr := plan.RunWithConfig("", d.Config); runErr != nil {
				log.Fatal(runErr)
			}
		}()
	}

	if err := d.client.Agent().ServiceRegister(service); err != nil {
		return err
	}

	go d.probeHealthCheck(service.Check.CheckID)

	return nil
}

func createConsulConfig(cfg *configs.DiscoveryConfig) *consul.Config {
	consulCfg := consul.DefaultConfig()

	if cfg.Address != "" && cfg.Address != strings.TrimSpace(strings.ToLower(consulCfg.Address)) {
		consulCfg.Address = cfg.Address
	}
	if cfg.Scheme != "" && strings.ToLower(cfg.Scheme) != strings.ToLower(consulCfg.Scheme) {
		consulCfg.Scheme = cfg.Scheme
	}

	return consulCfg
}

func (d *ServiceDiscovery) probeHealthCheck(checkID string) {
	debug.Debugf("Health Check Enabled with ticker duration set to %s, and CheckID set to '%s'.", (d.HealthCheckTTL / 2).String(), checkID)
	ticker := time.NewTicker(d.HealthCheckTTL / 2)
	for {
		debug.Debugf("Health Check status updated at %s.", time.Now().Format(time.DateTime))
		if err := d.client.Agent().UpdateTTL(checkID, "healthy", consul.HealthPassing); err != nil {
			log.Fatalf("An error occurred while running health check ticker: %v", err)
		}
		<-ticker.C
	}
}
