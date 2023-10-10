package dbmservice

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/kennethpensopay/dbmservice/internal/modinfo"
	debug "github.com/kennethpensopay/go-debug"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	name           string
	port           int
	tags           []string
	listener       net.Listener
	consulClient   *consul.Client
	consulConfig   *consul.Config
	healthCheckTTL time.Duration
}

func NewService() *Service {
	serviceName := getServiceName()
	debug.Debugf("Creating service: %s", serviceName)

	consulCfg := createConsulConfig()
	consulClient, err := consul.NewClient(consulCfg)
	if err != nil {
		log.Fatalf("Could not create new Consul Service Client: %v", err)
	}

	return &Service{
		name:           serviceName,
		port:           getServicePort(),
		tags:           getServiceTags(),
		listener:       nil,
		consulClient:   consulClient,
		consulConfig:   consulCfg,
		healthCheckTTL: getHealthCheckTTL(),
	}
}

// StartService This function is deprecated. Will be removed in future version.
//
// Deprecated: As of v1.1.0, this function simply calls [Start()].
func (s *Service) StartService() {
	s.Start()
}

func (s *Service) Start() {
	serviceAddress := strings.TrimSpace(os.Getenv("SERVICE_ADDRESS"))
	if serviceAddress == "" {
		serviceAddress = "localhost"
		debug.Debug("Service Address not defined. Set to 'localhost'.")
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", serviceAddress, s.port))
	if err != nil {
		log.Fatalf("Could not listen to server: %v", err)
	}
	s.listener = ln
	s.registerService()

	for {
		if _, lnaErr := s.listener.Accept(); lnaErr != nil {
			log.Fatalf("Service Listener could not accept next connection: %v", lnaErr)
		}
	}
}

func (s *Service) ServiceAddr() string {
	return s.listener.Addr().String()
}

func (s *Service) registerService() {
	var err error

	svcReg := &consul.AgentServiceRegistration{
		Name: s.name,
		Tags: s.tags,
		Check: &consul.AgentServiceCheck{
			FailuresBeforeCritical:         3,
			DeregisterCriticalServiceAfter: s.healthCheckTTL.String(),
			TTL:                            s.healthCheckTTL.String(),
			TLSSkipVerify:                  true,
		},
	}

	var servicePortStr string
	if svcReg.Address, servicePortStr, err = net.SplitHostPort(s.listener.Addr().String()); err != nil || svcReg.Address == "" || servicePortStr == "" {
		log.Fatalln("Could not resolve Service Address and Service Port.")
	}

	if svcReg.Port, err = strconv.Atoi(strings.TrimSpace(servicePortStr)); err != nil {
		log.Fatalln("Could not parse service port to integer.")
	}
	s.port = svcReg.Port

	svcReg.ID = s.serviceID()
	svcReg.Check.CheckID = svcReg.ID + "_health"

	debug.Debugf("Service configured to register on address %s:%d, with ID '%s' and Health check CheckID '%s'",
		svcReg.Address,
		svcReg.Port,
		svcReg.ID,
		svcReg.Check.CheckID)

	if enablePlanWatch, bErr := strconv.ParseBool(strings.TrimSpace(os.Getenv("ENABLE_PLAN_WATCH"))); bErr == nil && enablePlanWatch {
		go s.enablePlanWatch(svcReg.ID)
	}

	if err = s.consulClient.Agent().ServiceRegister(svcReg); err != nil {
		log.Fatalf("An error occurred while trying to register service: %v", err)
	}
	go s.updateHealthCheck()

	debug.Debugf("Service '%s' was registered with address/port '%s' to Consul instance: %s",
		s.name,
		s.listener.Addr().String(),
		fmt.Sprintf("%s://%s", s.consulConfig.Scheme, s.consulConfig.Address),
	)
	log.Printf("Service '%s' is now running on port %d ...\n", s.name, s.port)
}

func (s *Service) enablePlanWatch(serviceID string) {
	debug.Debug("Consul Plan Watch has been enabled.")
	plan, err := watch.Parse(map[string]any{
		"type":        "service",
		"service":     s.name,
		"passingonly": true,
	})
	if err != nil {
		log.Fatal(err)
	}

	plan.HybridHandler = func(index watch.BlockingParamVal, result any) {
		switch msg := result.(type) {
		case []*consul.ServiceEntry:
			msgLength := len(msg)
			if msgLength > 0 {
				if service := msg[msgLength-1].Service; service.ID != serviceID {
					log.Printf("Service instance joined: %v", service)
				}
			}
		}
	}

	go func() {
		if runErr := plan.RunWithConfig("", s.consulConfig); runErr != nil {
			log.Fatal(runErr)
		}
	}()
}

func (s *Service) updateHealthCheck() {
	checkId := s.serviceID() + "_health"

	debug.Debugf("Health Check Enabled with ticker duration set to %s, and CheckID set to '%s'.", (s.healthCheckTTL / 2).String(), checkId)
	ticker := time.NewTicker(s.healthCheckTTL / 2)
	for {
		debug.Debugf("Health Check status updated at %s.", time.Now().Format(time.DateTime))
		if err := s.consulClient.Agent().UpdateTTL(checkId, "healthy", consul.HealthPassing); err != nil {
			log.Fatalf("An error occurred while running health check ticker: %v", err)
		}
		<-ticker.C
	}
}

func (s *Service) serviceID() string {
	return fmt.Sprintf("%s-%d", strings.ReplaceAll(s.name, ".", "-"), s.port)
}

func createConsulConfig() *consul.Config {
	cfg := consul.DefaultConfig()
	if address := strings.TrimSpace(os.Getenv("CONSUL_ADDRESS")); address != "" {
		cfg.Address = address
	}
	if scheme := strings.TrimSpace(os.Getenv("CONSUL_SCHEME")); scheme != "" {
		cfg.Scheme = scheme
	}
	return cfg
}

func getServiceName() string {
	var serviceName string
	if serviceName = strings.TrimSpace(strings.ReplaceAll(os.Getenv("SERVICE_NAME"), " ", "")); serviceName != "" {
		debug.Debugf("Service Name set from env: %s", serviceName)
	} else {
		serviceName = fmt.Sprintf("%s.services.local", modinfo.ModInfo().ModuleNameAsMD5Sum())
		debug.Debugf("No Service Name found in env. Set to: %s", serviceName)
	}
	return serviceName
}

func getServicePort() int {
	if portInt, err := strconv.Atoi(strings.TrimSpace(os.Getenv("SERVICE_PORT"))); err != nil || (portInt < 0 || portInt > 65535) {
		log.Println("The 'SERVICE_PORT' environment variable is not defined or not valid. Reverting to default Port 0, resulting in a random available port.")
		return 0
	} else {
		debug.Debugf("Service Port set from env: %d", portInt)
		return portInt
	}
}

func getServiceTags() []string {
	if tagsStr := strings.TrimSpace(strings.ReplaceAll(os.Getenv("SERVICE_TAGS"), " ", "")); tagsStr != "" {
		if tags := strings.Split(tagsStr, ","); len(tags) > 0 {
			debug.Debugf("Service Tags set from env: %v", tags)
			return tags
		}
	}
	return []string{"go", "service"}
}

func getHealthCheckTTL() time.Duration {
	if ttl, err := strconv.Atoi(strings.TrimSpace(os.Getenv("CONSUL_TTL_IN_SECONDS"))); err == nil && ttl >= 5 {
		debug.Debugf("Health Check TTL set from env: %d", ttl)
		return time.Duration(ttl) * time.Second
	}

	debug.Debug("The 'CONSUL_TTL_IN_SECONDS' environment variable is not defined or not valid (minimum 5 seconds). Reverting to default TTL (8s / 4s).")
	return 8 * time.Second
}
