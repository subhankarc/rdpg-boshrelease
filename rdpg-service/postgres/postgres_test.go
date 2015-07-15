package postgres_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wayneeseguin/rdpgd/pg"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/starkandwayne/rdpg-acceptance-tests/helpers"
)

func getRowCount(address string, sq string) (queryRowCount int, err error) {
	p := pg.NewPG(address, "7432", `rdpg`, `rdpg`, "admin")
	db, err := p.Connect()
	if err != nil {
		return 0, err
	}
	var rowCount []int
	err = db.Select(&rowCount, sq)
	if err != nil {
		return 0, err
	}
	return rowCount[0], nil
}

func getAllNodes() (allNodes []*consulapi.CatalogService) {

	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = helpers.TestConfig.ConsulIP
	consulClient, _ := consulapi.NewClient(consulConfig)

	rdpgsc1Nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
	rdpgsc2Nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
	nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
	for index := range rdpgsc1Nodes {
		nodes = append(nodes, rdpgsc1Nodes[index])
	}
	for index := range rdpgsc2Nodes {
		nodes = append(nodes, rdpgsc2Nodes[index])
	}

	return nodes
}

func getNodesByClusterName(clusterName string) (allNodes []*consulapi.CatalogService) {

	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = helpers.TestConfig.ConsulIP
	consulClient, _ := consulapi.NewClient(consulConfig)

	nodes, _, _ := consulClient.Catalog().Service(clusterName, "", nil)

	return nodes

}

func getServiceNodes() (allServiceNodes []*consulapi.CatalogService) {

	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = helpers.TestConfig.ConsulIP
	consulClient, _ := consulapi.NewClient(consulConfig)

	nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
	rdpgsc2Nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
	for index := range rdpgsc2Nodes {
		nodes = append(nodes, rdpgsc2Nodes[index])
	}

	return nodes
}

var _ = Describe("RDPG Postgres Testing...", func() {

	It("Check Schemas Exist", func() {

		allNodes := getAllNodes()

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(schema_name) as rowCount FROM information_schema.schemata WHERE schema_name IN ('bdr', 'rdpg', 'cfsb', 'tasks', 'backups', 'metrics', 'audit'); `
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d schemas in rdpg database...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		fmt.Printf("%#v\n", nodeRowCount)

		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(7))
	})

	It("Check cfsb Tables Exist", func() {

		allNodes := getAllNodes()

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(table_name) as rowCount FROM information_schema.tables WHERE table_schema = 'cfsb' and table_name IN ('services','plans','instances','bindings','credentials'); `
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema cfsb...\n", allNodes[i].Node, rowCount)
		}

		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(5))
	})

	It("Check rdpg Tables Exist", func() {

		allNodes := getAllNodes()

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(table_name) as rowCount FROM information_schema.tables WHERE table_schema = 'rdpg' and table_name IN ('config', 'consul_watch_notifications', 'events'); `
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema rdpg...\n", allNodes[i].Node, rowCount)
		}

		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(3))

	})

	It("Check tasks Tables Exist", func() {
		allNodes := getAllNodes()

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(table_name) as rowCount FROM information_schema.tables WHERE table_schema = 'tasks' and table_name IN ('tasks','schedules'); `
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema tasks...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(2))

	})

	It("Check Instance Counts", func() {

		rdpgmcNodes := getNodesByClusterName("rdpgmc")
		rdpgsc1Nodes := getNodesByClusterName("rdpgsc1")
		rdpgsc2Nodes := getNodesByClusterName("rdpgsc2")

		//Check SC1
		var sc1InstanceCount []int
		for i := 0; i < len(rdpgsc1Nodes); i++ {
			address := rdpgsc1Nodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL; `
			rowCount, err := getRowCount(address, sq)
			sc1InstanceCount = append(sc1InstanceCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgsc1Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1InstanceCount); i++ {
			Expect(sc1InstanceCount[0]).To(Equal(sc1InstanceCount[i]))
		}

		//Check SC2
		var sc2InstanceCount []int
		for i := 0; i < len(rdpgsc2Nodes); i++ {
			address := rdpgsc2Nodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL; `
			rowCount, err := getRowCount(address, sq)
			sc2InstanceCount = append(sc2InstanceCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgsc2Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2InstanceCount); i++ {
			Expect(sc2InstanceCount[0]).To(Equal(sc2InstanceCount[i]))
		}

		//Check MC
		var mcInstanceCount []int
		for i := 0; i < len(rdpgmcNodes); i++ {
			address := rdpgmcNodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL; `
			rowCount, err := getRowCount(address, sq)
			mcInstanceCount = append(mcInstanceCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgmcNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(mcInstanceCount); i++ {
			Expect(mcInstanceCount[0]).To(Equal(mcInstanceCount[i]))
		}

		//Verify that the number of instances seen in the Management Cluster is the
		//sum of the instances from the service ClusterIPs

		totalManagementClusterInstances := mcInstanceCount[0]
		totalServiceClusterInstances := sc1InstanceCount[0] + sc2InstanceCount[0]
		Expect(totalManagementClusterInstances).To(Equal(totalServiceClusterInstances))
		fmt.Printf("Total Management Cluster Instance Count: %d\n", totalManagementClusterInstances)
		fmt.Printf("Total Service Cluster Instance Count: %d\n", totalServiceClusterInstances)
	})

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

	It("Check Scheduled Tasks Exist", func() {

		rdpgmcNodes := getNodesByClusterName("rdpgmc")
		rdpgsc1Nodes := getNodesByClusterName("rdpgsc1")
		rdpgsc2Nodes := getNodesByClusterName("rdpgsc2")

		fmt.Println(rdpgsc1Nodes)

		//Check SC1
		var sc1RowCount []int
		for i := 0; i < len(rdpgsc1Nodes); i++ {
			address := rdpgsc1Nodes[i].Address
			sq := ` SELECT count(*) AS rowCount FROM tasks.schedules WHERE role IN ('all', 'service'); `
			rowCount, err := getRowCount(address, sq)
			sc1_rowCount = append(sc1_rowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgsc1Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1_rowCount); i++ {
			Expect(sc1_rowCount[0]).To(Equal(sc1_rowCount[i]))
		}

		Expect(sc1_rowCount[0]).To(BeNumerically(">=", 3))

		//Check SC2
		var sc2RowCount []int
		for i := 0; i < len(rdpgsc2Nodes); i++ {
			address := rdpgsc2Nodes[i].Address
			sq := ` SELECT count(*) AS rowCount FROM tasks.schedules WHERE role IN ('all', 'service'); `
			rowCount, err := getRowCount(address, sq)
			sc2RowCount = append(sc2RowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgsc2Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2RowCount); i++ {
			Expect(sc2RowCount[0]).To(Equal(sc2RowCount[i]))
		}

		Expect(sc2RowCount[0]).To(BeNumerically(">=", 3))

		//Check MC
		var mcRowCount []int
		for i := 0; i < len(rdpgmcNodes); i++ {
			address := rdpgmcNodes[i].Address
			sq := ` SELECT count(*) AS rowCount FROM tasks.schedules WHERE role IN ('all', 'manager'); `
			rowCount, err := getRowCount(address, sq)
			mc_rowCount = append(mc_rowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgmcNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(mc_rowCount); i++ {
			Expect(mc_rowCount[0]).To(Equal(mc_rowCount[i]))
		}

		Expect(mc_rowCount[0]).To(BeNumerically(">=", 4))

	})

	It("Check For Missed Scheduled Tasks", func() {

		//Looks for any active scheduled tasks which have not been scheduled in two
		//frequency cycles

		allNodes := getAllNodes()

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(*) as rowCount FROM tasks.schedules WHERE last_scheduled_at + (2 * frequency) < CURRENT_TIMESTAMP AND enabled=true; `
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d missed scheduled tasks...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		//There should be no rows which have missed their schedule twice
		Expect(nodeRowCount[0]).To(Equal(0))

	})

	It("Check for databases known to cfsb.instances but don't exist", func() {

		allNodes := getServiceNodes()

		//Check SC
		var scRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(name) AS rowCount FROM ( (SELECT dbname AS name FROM cfsb.instances) EXCEPT (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') ) AS instances_missing_databaes; `
			rowCount, err := getRowCount(address, sq)
			scRowCount = append(scRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases known to cfsb.instances but don't exist...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(scRowCount); i++ {
			Expect(scRowCount[0]).To(Equal(scRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		//There should be no rows of databases which are known to cfsb.instances but don't exist
		Expect(scRowCount[0]).To(Equal(0))

	})

	It("Check for databases which exist and aren't known to cfsb.instances", func() {

		allNodes := getServiceNodes()

		//Check SC
		var scRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(name) as rowCount FROM ( (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') EXCEPT (SELECT dbname AS name FROM cfsb.instances)) AS databases_missing_instances; `
			rowCount, err := getRowCount(address, sq)
			scRowCount = append(scRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases in pg not in cfsb.instances...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(scRowCount); i++ {
			Expect(scRowCount[0]).To(Equal(scRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		//There should be no rows of databases which are known to pg but aren't in cfsb.instances
		Expect(scRowCount[0]).To(Equal(0))

	})

})
