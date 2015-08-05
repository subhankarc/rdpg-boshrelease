package tasks

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/starkandwayne/rdpgd/log"
)

func getNode() (node string, err error) {
	node = ``
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf("tasks.getNode() consulapi.NewClient()! %s", err))
		return
	}

	consulAgent := client.Agent()
	info, err := consulAgent.Self()
	node = info["Config"]["AdvertiseAddr"].(string)
	return
}

func isWriteNode(currentIP string) (isWriter bool) {
	isWriter = false

	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.consul_helpers isWriteMode() ! %s`, err))
		return
	}
	catalog := client.Catalog()

	svcs, _, err := catalog.Service(fmt.Sprintf(`%s-master`, ClusterID), "", nil)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.consul_helpers isWriteMode() Retrieving Service Catalog ! %s`, err))
		return
	}
	if len(svcs) == 0 {
		return
	}
	if svcs[0].Address == currentIP {
		isWriter = true
		return
	}

	return
}
