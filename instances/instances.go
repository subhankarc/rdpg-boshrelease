package instances

import (
	"fmt"
	"os"
	"strings"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/starkandwayne/rdpgd/log"
)

var (
	pbPort    string
	pgPass    string
	ClusterID string
)

type Instance struct {
	ID             string `db:"id"`
	ClusterID      string `db:"cluster_id" json:"cluster_id"`
	InstanceID     string `db:"instance_id" json:"instance_id"`
	ServiceID      string `db:"service_id" json:"service_id"`
	PlanID         string `db:"plan_id" json:"plan_id"`
	OrganizationID string `db:"organization_id" json:"organization_id"`
	SpaceID        string `db:"space_id" json:"space_id"`
	Database       string `db:"dbname" json:"dbname"`
	User           string `db:"dbuser" json:"uname"`
	Pass           string `db:"dbpass" json:"pass"`
	lock           *consulapi.Lock
	lockCh         <-chan struct{}
}

func init() {
	ClusterID = os.Getenv(`RDPGD_CLUSTER`)
	if ClusterID == "" {
		log.Error(`tasks.Scheduler() RDPGD_CLUSTER not found in environment!!!`)
	}
	pbPort = os.Getenv(`RDPGD_PB_PORT`)
	if pbPort == `` {
		pbPort = `6432`
	}
	pgPass = os.Getenv(`RDPGD_PG_PASS`)
}

func (i *Instance) ExternalDNS() (dns string, err error) {
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf(`instances.Instance#ExternalDNS(%s) i.ClusterIPs() ! %s`, i.InstanceID, err))
		return
	}
	catalog := client.Catalog()

	services, _, err := catalog.Service(fmt.Sprintf(`%s-master`, i.ClusterID), "", nil)
	if err != nil {
		log.Error(fmt.Sprintf("instances.Instance#ExternalDNS(%s) consulapi.Catalog().Service() ! %s", i.ClusterID, err))
		return
	}
	if len(services) == 0 {
		// Master is missing, use the first service node available...
		log.Error(fmt.Sprintf("instances.Instance#ExternalDNS(%s) ! Master service node not found via Consul...?!", i.ClusterID))
		return
	}
	masterIP := services[0].Address
	// TODO: Figure out where we'll store and retrieve the external DNS information
	// instead of IP Should be settable per service cluster via BOSH properties
	// and we read here via os.Getenv(``).
	dns = fmt.Sprintf(`%s:5432`, masterIP)
	return
}

func (i *Instance) URI() (uri string, err error) {
	dns, err := i.ExternalDNS()
	if err != nil {
		log.Error(fmt.Sprintf("instances.Instance#URI(%s) ! %s", i.ClusterID))
		return
	}
	d := `postgres://%s:%s@%s/%s?sslmode=%s`
	uri = fmt.Sprintf(d, i.User, i.Pass, dns, i.Database, `disable`)
	return
}

func (i *Instance) DSN() (uri string, err error) {
	dns, err := i.ExternalDNS()
	if err != nil {
		log.Error(fmt.Sprintf("instances.Instance#DSN(%s) ! %s", i.ClusterID))
		return
	}
	s := strings.Split(dns, ":")
	d := `host=%s port=%s user=%s password=%s dbname=%s connect_timeout=%s sslmode=%s`
	uri = fmt.Sprintf(d, s[0], s[1], i.User, i.Pass, i.Database, `5`, `disable`)
	return
}

func (i *Instance) JDBCURI() (uri string, err error) {
	dns, err := i.ExternalDNS()
	if err != nil {
		log.Error(fmt.Sprintf("instances.Instance#JDBCURI(%s) ! %s", i.ClusterID))
		return
	}
	d := `jdbc:postgres://%s:%s@%s/%s?sslmode=%s`
	uri = fmt.Sprintf(d, i.User, i.Pass, dns, i.Database, `disable`)
	return
}

// Lock the instance within the current cluster via Consul.
func (i *Instance) Lock() (err error) {
	key := fmt.Sprintf("rdpg/%s/database/%s/lock", i.ClusterID, i.Database)
	client, _ := consulapi.NewClient(consulapi.DefaultConfig())
	i.lock, err = client.LockKey(key)
	if err != nil {
		log.Error(fmt.Sprintf("scheduler.Schedule() Error Locking Scheduler Key %s ! %s", key, err))
		return
	}
	i.lockCh, err = i.lock.Lock(nil)
	if err != nil {
		log.Error(fmt.Sprintf("scheduler.Lock() Error aquiring instance Key lock %s ! %s", key, err))
		return
	}
	if i.lockCh == nil {
		err = fmt.Errorf(`Scheduler Lock not aquired.`)
	}
	return
}

func (i *Instance) Unlock() (err error) {
	if i.lock != nil {
		err = i.lock.Unlock()
	}
	return
}
