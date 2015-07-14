package postgres_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wayneeseguin/rdpgd/pg"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/starkandwayne/rdpg-acceptance-tests/helpers"
)

var (
	consulClient *consulapi.Client
)

func init() {
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = helpers.TestConfig.ConsulIP
	consulClient, _ = consulapi.NewClient(consulConfig)
	svcs, err := consulClient.Services()
	if err != nil {
		fmt.Sprintf(`Consul Services Error: %s`, err)
	} else {
		fmt.Sprintf(`Consul Services: %+v`, svcs)
	}
}

func getRowCount(address string, sq string) (count int, err error) {
	p := pg.NewPG(address, "7432", `rdpg`, `rdpg`, "admin")
	db, err := p.Connect()
	if err != nil {
		return 0, err
	}
	rowCount := []int{}
	if err = db.Select(&rowCount, sq); err != nil {
		return 0, err
	} else {
		count = rowCount[0]
	}
	return
}

func getAllNodes() (nodes []*consulapi.CatalogService) {
	nodes, _, _ = consulClient.Catalog().Service("rdpgmc", "", nil)

	rdpgSC1Nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
	for index, _ := range rdpgSC1Nodes {
		nodes = append(nodes, rdpgSC1Nodes[index])
	}

	rdpgSC2Nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
	for index, _ := range rdpgSC2Nodes {
		nodes = append(nodes, rdpgSC2Nodes[index])
	}

	return nodes
}

func getServiceNodes() (nodes []*consulapi.CatalogService) {
	nodes, _, _ = consulClient.Catalog().Service("rdpgsc1", "", nil)
	rdpgSC2Nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
	for index, _ := range rdpgSC2Nodes {
		nodes = append(nodes, rdpgSC2Nodes[index])
	}
	return nodes
}

var _ = Describe("RDPG Postgres Testing...", func() {
	BeforeEach(func() {
	})

	It("Check Schemas Exist", func() {
		allNodes := getAllNodes()

		//Check all nodes
		nodeRowCount := []int{}
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(schema_name) as row_count FROM information_schema.schemata WHERE schema_name IN ('bdr', 'rdpg', 'cfsb', 'tasks', 'backups', 'metrics', 'audit') LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d schemas in rdpg database...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(nodeRowCount[0]).To(Equal(7))
	})

	It("Check cfsb Tables Exist", func() {

		allNodes := getAllNodes()

		//Check all nodes
		nodeRowCount := []int{}
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(table_name) as row_count FROM information_schema.tables WHERE table_schema = 'cfsb' and table_name IN ('services','plans','instances','bindings','credentials') LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema cfsb...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(nodeRowCount[0]).To(Equal(5))

	})

	It("Check rdpg Tables Exist", func() {

		allNodes := getAllNodes()

		//Check all nodes
		nodeRowCount := []int{}
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(table_name) as row_count FROM information_schema.tables WHERE table_schema = 'rdpg' and table_name IN ('config', 'consul_watch_notifications', 'events') LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema rdpg...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(nodeRowCount[0]).To(Equal(3))
	})

	It("Check tasks Tables Exist", func() {
		allNodes := getAllNodes()

		//Check all nodes
		nodeRowCount := []int{}
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(table_name) as row_count FROM information_schema.tables WHERE table_schema = 'tasks' and table_name IN ('tasks','schedules') LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema tasks...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(nodeRowCount[0]).To(Equal(2))
	})

	It("Check Instance Counts", func() {
		rdpgMCNodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		rdpgSC1Nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgSC2Nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		fmt.Println(rdpgSC1Nodes)

		//Check SC1
		sc1InstanceCount := []int{}
		for i := 0; i < len(rdpgSC1Nodes); i++ {
			address := rdpgSC1Nodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			sc1InstanceCount = append(sc1InstanceCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgSC1Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1InstanceCount); i++ {
			Expect(sc1InstanceCount[0]).To(Equal(sc1InstanceCount[i]))
		}

		//Check SC2
		sc2InstanceCount := []int{}
		for i := 0; i < len(rdpgSC2Nodes); i++ {
			address := rdpgSC2Nodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			sc2InstanceCount = append(sc2InstanceCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgSC2Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2InstanceCount); i++ {
			Expect(sc2InstanceCount[0]).To(Equal(sc2InstanceCount[i]))
		}

		//Check MC
		mc_instance_count := []int{}
		for i := 0; i < len(rdpgMCNodes); i++ {
			address := rdpgMCNodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			mc_instance_count = append(mc_instance_count, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgMCNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(mc_instance_count); i++ {
			Expect(mc_instance_count[0]).To(Equal(mc_instance_count[i]))
		}

		//Verify that the number of instances seen in the Management Cluster is the
		//sum of the instances from the service ClusterIPs

		totalManagementClusterInstances := mc_instance_count[0]
		totalServiceClusterInstances := sc1InstanceCount[0] + sc2InstanceCount[0]
		Expect(totalManagementClusterInstances).To(Equal(totalServiceClusterInstances))
		fmt.Printf("Total Management Cluster Instance Count: %d\n", totalManagementClusterInstances)
		fmt.Printf("Total Service Cluster Instance Count: %d\n", totalServiceClusterInstances)
	})

	It("Check Node Counts", func() {
		rdpgMCNodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		rdpgSC1Nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgSC2Nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		//Check SC1
		expectedNodeCount := 2
		Expect(len(rdpgSC1Nodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Service Cluster 1 Nodes\n", len(rdpgSC1Nodes), expectedNodeCount)

		//Check SC2
		expectedNodeCount = 2
		Expect(len(rdpgSC2Nodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Service Cluster 2 Nodes\n", len(rdpgSC2Nodes), expectedNodeCount)

		//Check MC
		expectedNodeCount = 3
		Expect(len(rdpgMCNodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Managment Cluster Nodes\n", len(rdpgMCNodes), expectedNodeCount)
	})

	It("Check Scheduled Tasks Exist", func() {
		rdpgMCNodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		rdpgSC1Nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgSC2Nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		fmt.Println(rdpgSC1Nodes)

		//Check SC1
		sc1RowCount := []int{}
		for i := 0; i < len(rdpgSC1Nodes); i++ {
			address := rdpgSC1Nodes[i].Address
			sq := `SELECT count(*) AS row_count FROM tasks.schedules WHERE role IN ('all', 'service') LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			sc1RowCount = append(sc1RowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgSC1Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1RowCount); i++ {
			Expect(sc1RowCount[0]).To(Equal(sc1RowCount[i]))
		}

		Expect(sc1RowCount[0]).To(BeNumerically(">=", 3))

		//Check SC2
		sc2RowCount := []int{}
		for i := 0; i < len(rdpgSC2Nodes); i++ {
			address := rdpgSC2Nodes[i].Address
			sq := `SELECT count(*) AS row_count FROM tasks.schedules WHERE role IN ('all', 'service') LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			sc2RowCount = append(sc2RowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgSC2Nodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2RowCount); i++ {
			Expect(sc2RowCount[0]).To(Equal(sc2RowCount[i]))
		}

		Expect(sc2RowCount[0]).To(BeNumerically(">=", 3))

		//Check MC
		mcRowCount := []int{}
		for i := 0; i < len(rdpgMCNodes); i++ {
			address := rdpgMCNodes[i].Address
			sq := `SELECT count(*) AS row_count FROM tasks.schedules WHERE role IN ('all', 'manager') LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			mcRowCount = append(mcRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgMCNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(mcRowCount); i++ {
			Expect(mcRowCount[0]).To(Equal(mcRowCount[i]))
		}

		Expect(mcRowCount[0]).To(BeNumerically(">=", 4))
	})

	It("Check For Missed Scheduled Tasks", func() {
		//Looks for any active scheduled tasks which have not been scheduled in two
		//frequency cycles

		allNodes := getAllNodes()

		//Check all nodes
		nodeRowCount := []int{}
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(*) AS row_count FROM tasks.schedules WHERE last_scheduled_at + (2 * frequency) < CURRENT_TIMESTAMP AND enabled=true LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d missed scheduled tasks...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		//There should be no rows which have missed their schedule twice
		Expect(nodeRowCount[0]).To(Equal(0))

	})

	It("Check for databases known to cfsb.instances but don't exist", func() {
		allNodes := getServiceNodes()

		//Check SC
		scRowCount := []int{}
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(name) AS row_count FROM ( (SELECT dbname AS name FROM cfsb.instances) EXCEPT (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') ) AS instances_missing_databaes LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			scRowCount = append(scRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases known to cfsb.instances but don't exist...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(scRowCount); i++ {
			Expect(scRowCount[0]).To(Equal(scRowCount[i]))
		}
		//There should be no rows of databases which are known to cfsb.instances but don't exist
		Expect(scRowCount[0]).To(Equal(0))

	})

	It("Check for databases which exist and aren't known to cfsb.instances", func() {

		allNodes := getServiceNodes()

		//Check SC
		scRowCount := []int{}
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(name) as row_count FROM ( (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') EXCEPT (SELECT dbname AS name FROM cfsb.instances)) AS databases_missing_instances LIMIT 1`
			rowCount, err := getRowCount(address, sq)
			scRowCount = append(scRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases in pg not in cfsb.instances...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(scRowCount); i++ {
			Expect(scRowCount[0]).To(Equal(scRowCount[i]))
		}
		//There should be no rows of databases which are known to pg but aren't in cfsb.instances
		Expect(scRowCount[0]).To(Equal(0))

	})
})
