package rdpg

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/jmoiron/sqlx"
	"github.com/starkandwayne/rdpgd/log"
)

var (
	ClusterID      string
	myIP           string
	pgPort         string
	pgPass         string
	rdpgdAdminPort string
	rdpgdAdminUser string
	rdpgdAdminPass string
)

type RDPG struct {
	IP            string
	URI           string
	DB            *sqlx.DB
	ConsulClient  *consulapi.Client
	ConsulAgent   *consulapi.Agent
	ConsulCatalog *consulapi.Catalog
}

func init() {
	pgPort = os.Getenv(`RDPGD_PG_PORT`)
	if pgPort == `` {
		pgPort = `5432`
	}
	pgPass = os.Getenv(`RDPGD_PG_PASS`)
	ClusterID = os.Getenv(`RDPGD_CLUSTER`)
	rdpgdAdminPort = os.Getenv(`RDPGD_ADMIN_PORT`)
	if rdpgdAdminPort == "" {
		rdpgdAdminPort = "58888"
	}
	rdpgdAdminUser = os.Getenv(`RDPGD_ADMIN_USER`)
	if rdpgdAdminUser == "" {
		rdpgdAdminUser = "admin"
	}
	rdpgdAdminPass = os.Getenv(`RDPGD_ADMIN_PASS`)
	if rdpgdAdminPass == "" {
		rdpgdAdminPass = "admin"
	}
}

func newRDPG() (r *RDPG) {
	r = &RDPG{}
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf("rdpg.newRDPG() consulapi.NewClient()! %s", err))
		return
	}
	r.ConsulClient = client
	r.ConsulAgent = r.ConsulClient.Agent()
	r.ConsulCatalog = r.ConsulClient.Catalog()
	info, err := r.ConsulAgent.Self()
	r.IP = info["Config"]["AdvertiseAddr"].(string)
	myIP = r.IP
	return
}

// Return a list of the rdpg management and service clusters registered with Consul.
func (r *RDPG) Clusters() (clusters []string, err error) {
	log.Trace(`rdpg.Clusters(%s) Retrieving list of registered RDPG clusters from Consul...`)
	services, _, err := r.ConsulCatalog.Services(nil)
	if err != nil {
		log.Error(fmt.Sprintf("rdpg.Clusters() ! %s", err))
		return
	}
	if len(services) == 0 {
		log.Error(fmt.Sprintf("rdpg.Clusters() ! No services found, no known clusters?!"))
		return
	}
	re := regexp.MustCompile(`^rdpg(mc|sc[0-9]+)$`)
	for key, _ := range services {
		if re.MatchString(key) {
			clusters = append(clusters, key)
		}
	}
	return
}

// Call RDPG Admin API for given IP
func CallAdminAPI(ip, method, path string) (err error) {
	url := fmt.Sprintf("http://%s:%s/%s", ip, os.Getenv("RDPGD_ADMIN_PORT"), path)
	log.Trace(fmt.Sprintf(`rdpg.CallAdminAPI(%s,%s,%s) %s`, ip, method, path, url))
	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(`{}`)))
	// req.Header.Set("Content-Type", "application/json")
	// TODO: Retrieve from configuration in database.
	req.SetBasicAuth(os.Getenv("RDPGD_ADMIN_USER"), os.Getenv("RDPGD_ADMIN_PASS"))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf(`pg.Host<%s>#AdminAPI(%s,%s) ! %s`, ip, method, url, err))
		return
	}
	resp.Body.Close()
	return
}

// Register the rdpgd node with the cluster service in Consul.
func (r *RDPG) Register() (err error) {
	log.Trace(fmt.Sprintf(`rdpg.RDPG<%s>#Register() Registering Cluster Service...`, ClusterID))

	port, err := strconv.Atoi(pgPort)
	if err != nil {
		log.Error(fmt.Sprintf("rdpg.RDPG<%s>#Register() ! %s", ClusterID, err))
		return
	}

	registration := &consulapi.AgentServiceRegistration{
		ID:   "rdpg",
		Name: ClusterID,
		Tags: []string{},
		Port: port,
	}
	err = r.ConsulAgent.ServiceRegister(registration)
	if err != nil {
		log.Error(fmt.Sprintf("rdpg.RDPG<%s>#Register() consulapi.Agent.ServiceRegister() ! %s", ClusterID, err))
		return
	}
	return
}
