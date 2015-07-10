package rdpg

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

type Node struct {
	PG *pg.PG
}

type Cluster struct {
	Role         string `json:"role" db:"role"`
	ID           string `json:"id" db:"cluster_id"`
	Nodes        []Node
	ConsulClient *consulapi.Client
}

// Create a new RDPG Cluster.
func NewCluster(clusterID string, client *consulapi.Client) (c *Cluster, err error) {
	c = &Cluster{ID: clusterID, ConsulClient: client}
	catalog := client.Catalog()
	services, _, err := catalog.Service(clusterID, "", nil)
	if err != nil {
		log.Error(fmt.Sprintf("rdpg.NewCluster(%s) ! %s", clusterID, err))
		return
	}
	if len(services) == 0 {
		log.Error(fmt.Sprintf("rdpg.NewCluster(%s) ! No services found, no known nodes?!", clusterID))
		return
	}
	for index, _ := range services {
		p := pg.NewPG(services[index].Address, pgPort, `postgres`, `postgres`, ``)
		c.Nodes = append(c.Nodes, Node{PG: p})
	}
	return
}

// Sets the cluster's write master ndoe.
func (c *Cluster) SetWriteMaster(ip string) (err error) {
	log.Trace(fmt.Sprintf(`rdpg.Cluster<%s>#SetWriteMaster() > %s`, c.ID, ip))
	url := ""
	if c.ID == "rdpgmc" {
		url = fmt.Sprintf(`http://%s:%s@127.0.0.1:%s/health/pb`, rdpgdAdminUser, rdpgdAdminPass, rdpgdAdminPort)
	} else {
		url = fmt.Sprintf(`http://%s:%s@127.0.0.1:%s/health/ha_pb_pg`, rdpgdAdminUser, rdpgdAdminPass, rdpgdAdminPort)
	}
	agent := c.ConsulClient.Agent()
	registration := &consulapi.AgentServiceRegistration{
		ID:      fmt.Sprintf(`%s-write`, c.ID),
		Name:    fmt.Sprintf(`%s-master`, c.ID),
		Tags:    []string{},
		Address: ip,
		Port:    5432,
		Check: &consulapi.AgentServiceCheck{
			HTTP:     url,
			Interval: "10s",
			TTL:      "0s",
			Timeout:  "5s",
		},
	}
	agent.ServiceRegister(registration)
	return
}

// Gets the cluster's write master node.
func (c *Cluster) GetWriteMaster() (n *Node, err error) {
	log.Trace(fmt.Sprintf(`rdpg.Cluster<%s>#GetWriteMaster()`, c.ID))
	catalog := c.ConsulClient.Catalog()
	svcs, _, err := catalog.Service(fmt.Sprintf(`%s-master`, c.ID), "", nil)
	if err != nil {
		log.Error(fmt.Sprintf(`rdpg.Cluster<%s>#GetWriteMaster() ! %s`, c.ID, err))
		return
	}
	if len(svcs) == 0 {
		return nil, nil
	} else {
		n = &Node{PG: &pg.PG{IP: svcs[0].Address}}
	}
	return
}

func (c *Cluster) ClusterIPs() (ips []string, err error) {
	catalog := c.ConsulClient.Catalog()
	services, _, err := catalog.Service(ClusterID, "", nil)
	if err != nil {
		log.Error(fmt.Sprintf("rdpg.ClusterIPs() ! %s", err))
		return
	}
	if len(services) == 0 {
		log.Error(fmt.Sprintf("rdpg.ClusterIPs() ! No services found, no known nodes?!"))
		return
	}
	ips = []string{}
	for index, _ := range services {
		ips = append(ips, services[index].Address)
	}
	return
}
