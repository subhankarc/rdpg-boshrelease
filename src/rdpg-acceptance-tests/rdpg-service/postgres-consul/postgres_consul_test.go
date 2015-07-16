package postgres_consul_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/starkandwayne/rdpg-acceptance-tests/helpers"
)

func getNodesByClusterName(clusterName string) (allNodes []*consulapi.CatalogService) {

	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = helpers.TestConfig.ConsulIP
	consulClient, _ := consulapi.NewClient(consulConfig)

	nodes, _, _ := consulClient.Catalog().Service(clusterName, "", nil)

	return nodes

}

var _ = Describe("Postgres Consul Checks...", func() {

	It("Check Node Counts", func() {

		rdpgmcNodes := getNodesByClusterName("rdpgmc")
		rdpgsc1Nodes := getNodesByClusterName("rdpgsc1")
		rdpgsc2Nodes := getNodesByClusterName("rdpgsc2")

		//Check SC1
		expectedNodeCount := 2
		Expect(len(rdpgsc1Nodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Service Cluster 1 Nodes\n", len(rdpgsc1Nodes), expectedNodeCount)

		//Check SC2
		expectedNodeCount = 2
		Expect(len(rdpgsc2Nodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Service Cluster 2 Nodes\n", len(rdpgsc2Nodes), expectedNodeCount)

		//Check MC
		expectedNodeCount = 3
		Expect(len(rdpgmcNodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Managment Cluster Nodes\n", len(rdpgmcNodes), expectedNodeCount)
	})

})
