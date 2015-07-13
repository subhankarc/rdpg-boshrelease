package services

import (
	"errors"
	"fmt"
	"os"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/starkandwayne/rdpgd/log"
)

type Service struct {
	Name string `db:"name" json:"name"`
}

var (
	pgPort string
	pbPort string
)

func init() {
	pgPort = os.Getenv("RDPGD_PG_PORT")
	if pgPort == "" {
		pgPort = "5432"
	}
	pbPort = os.Getenv(`RDPGD_PB_PORT`)
	if pbPort == `` {
		pbPort = `6432`
	}
}

func NewService(name string) (s Service, err error) {
	switch name {
	case "haproxy", "pgbouncer", "pgbdr", "consul":
		s = Service{Name: name}
	default:
	}
	return
}

func (s *Service) Configure() (err error) {
	log.Trace(fmt.Sprintf(`services.Service<%s>#Configure()`, s.Name))
	// TODO: Protect each service configuration with a consul lock for the host
	// so that only one may be done at a time and we don't encounter write conflicts.
	switch s.Name {
	case "consul":
		err = s.ConfigureConsul()
	case "haproxy":
		err = s.ConfigureHAProxy()
	case "pgbouncer":
		err = s.ConfigurePGBouncer()
	case "pgbdr":
		err = s.ConfigurePGBDR()
	default:
		return errors.New(fmt.Sprintf(`services.Service<%s>#Configure() is unknown.`, s.Name))
	}
	if err != nil {
		log.Error(fmt.Sprintf("services.Service<%s>#Configure() ! %s", s.Name, err))
		return
	}
	return
}

func (s *Service) GetWriteMasterIP() (ip string, err error) {
	log.Trace(fmt.Sprintf(`services.Service<%s>#GetWriteMaster()`, s.Name))
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf("services.Service<%s>#GetWriteMaster() ! %s", s.Name, err))
		return
	}
	catalog := client.Catalog()
	svcs, _, err := catalog.Service(fmt.Sprintf(`%s-master`, os.Getenv(`RDPGD_CLUSTER`)), "", nil)
	if err != nil {
		log.Error(fmt.Sprintf(`services.Service<%s>#GetWriteMaster() ! %s`, s.Name, err))
		return
	}
	if len(svcs) == 0 {
		return "", nil
	} else {
		ip = svcs[0].Address
	}
	return
}

func (s *Service) ClusterIPs(clusterID string) (ips []string, err error) {
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf("services.Service<%s>#ClusterIPs() ! %s", s.Name, err))
		return
	}
	catalog := client.Catalog()
	services, _, err := catalog.Service(clusterID, "", nil)
	if err != nil {
		log.Error(fmt.Sprintf("services.Service<%s>#ClusterIPs() ! %s", s.Name, err))
		return
	}
	if len(services) == 0 {
		log.Error(fmt.Sprintf("services.Service<%s>#ClusterIPs() ! No services found, no known nodes?!", s.Name))
		return
	}
	ips = []string{}
	for index, _ := range services {
		ips = append(ips, services[index].Address)
	}
	return
}
