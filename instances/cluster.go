package instances

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/starkandwayne/rdpgd/log"
)

func (i *Instance) ClusterIPs() (ips []string, err error) {
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf("instances.Instance<%s>#ClusterIPs() ! %s", i.Database, err))
		return
	}
	catalog := client.Catalog()
	svcs, _, err := catalog.Service(i.ClusterID, "", nil)
	if err != nil {
		log.Error(fmt.Sprintf("instances.Instance<%s>#ClusterIPs() ! %s", i.Database, err))
		return
	}
	if len(svcs) == 0 {
		log.Error(fmt.Sprintf("instances.Instance<%s>#ClusterIPs() ! No services found, no known nodes?!", i.Database))
		return
	}
	ips = []string{}
	for index, _ := range svcs {
		ips = append(ips, svcs[index].Address)
	}
	return
}
