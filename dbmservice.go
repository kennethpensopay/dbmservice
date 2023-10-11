package dbmservice

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"github.com/kennethpensopay/dbmservice/configs"
	"github.com/kennethpensopay/dbmservice/internal/discovery"
	debug "github.com/kennethpensopay/go-debug"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Service struct {
	listener  net.Listener
	config    *configs.ServiceConfig
	discovery *discovery.ServiceDiscovery
}

func NewService() *Service {
	svcCfg, consulCfg := configs.LoadConfigs()
	debug.Debugf("Creating service: %s", svcCfg.Name)

	return &Service{
		config:    svcCfg,
		discovery: discovery.NewServiceDiscovery(consulCfg),
	}
}

func (s *Service) RegisterService() {
	var err error

	if s.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Address, s.config.Port)); err != nil {
		log.Fatalf("Could not create server listener: %v", err)
	}

	_, portStr, portStrErr := net.SplitHostPort(s.listener.Addr().String())
	if portStrErr != nil || strings.TrimSpace(portStr) == "" {
		log.Fatalf("Could not resolve Service Port from listener: %v", portStrErr)
	}

	s.config.Port, portStrErr = strconv.Atoi(strings.TrimSpace(portStr))
	if portStrErr != nil || strconv.Itoa(s.config.Port) != strings.TrimSpace(portStr) {
		log.Fatalf("Service port '%s' could not be parsed to an int: %v", portStr, portStrErr)
	}

	serviceID := strings.TrimSpace(fmt.Sprintf("%s-%d", strings.ReplaceAll(s.config.Name, ".", "-"), s.config.Port))
	svcReg := &consul.AgentServiceRegistration{
		ID:      serviceID,
		Name:    s.config.Name,
		Tags:    s.config.Tags,
		Address: s.config.Address,
		Port:    s.config.Port,
		Check: &consul.AgentServiceCheck{
			CheckID:                        serviceID + "_health",
			FailuresBeforeCritical:         3,
			DeregisterCriticalServiceAfter: s.discovery.HealthCheckTTL.String(),
			TTL:                            s.discovery.HealthCheckTTL.String(),
			TLSSkipVerify:                  true,
		},
	}

	debug.Debugf("Service configured to register on address %s:%d, with ID '%s' and Health CheckID '%s'",
		svcReg.Address,
		svcReg.Port,
		svcReg.ID,
		svcReg.Check.CheckID)

	if err = s.discovery.RegisterService(svcReg); err != nil {
		log.Fatalf("An error occurred while trying to register service with Consul: %v", err)
	}

	debug.Debugf("Service '%s' was registered with address/port '%s' to Consul instance: %s",
		svcReg.Name,
		fmt.Sprintf("%s:%d", svcReg.Address, svcReg.Port),
		fmt.Sprintf("%s://%s", s.discovery.Config.Scheme, s.discovery.Config.Address),
	)
}

func (s *Service) Start(handler http.Handler) {
	if s.listener == nil {
		log.Fatalln("Can't start server - no listener available.")
	}

	log.Printf("Service '%s' is now running on: %s:%d ...\n", s.config.Name, s.config.Address, s.config.Port)
	if err := http.Serve(s.listener, handler); err != nil {
		log.Fatalf("Could not start server on address '%s:%d': %v", s.config.Address, s.config.Port, err)
	}
}
