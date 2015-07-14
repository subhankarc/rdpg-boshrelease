package postgres_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wayneeseguin/rdpgd/pg"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/starkandwayne/rdpg-acceptance-tests/helpers"
)

func getRowCount(address string, sq string) (query_row_count int, err error) {
	p := pg.NewPG(address, "7432", `rdpg`, `rdpg`, "admin")
	db, err := p.Connect()
	if err != nil {
		return 0, err
	}
	row_count := make([]int, 0)
	err = db.Select(&row_count, sq)
	if err != nil {
		return 0, err
	}
	return row_count[0], nil
}

var _ = Describe("RDPG Service broker", func() {
	var (
		consulClient *consulapi.Client
	)

	BeforeEach(func() {
		consulConfig := consulapi.DefaultConfig()
		consulConfig.Address = helpers.TestConfig.ConsulIP
		consulClient, _ = consulapi.NewClient(consulConfig)

	})

	It("Check Schemas Exist", func() {
		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
		all_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		for index, _ := range rdpgsc1_nodes {
			all_nodes = append(all_nodes, rdpgsc1_nodes[index])
		}
		for index, _ := range rdpgsc2_nodes {
			all_nodes = append(all_nodes, rdpgsc2_nodes[index])
		}

		//Check all nodes
		node_row_count := make([]int, 0)
		for i := 0; i < len(all_nodes); i++ {
			address := all_nodes[i].Address
			sq := ` SELECT count(schema_name) as row_count FROM information_schema.schemata WHERE schema_name IN ('bdr', 'rdpg', 'cfsb', 'tasks', 'backups', 'metrics', 'audit'); `
			row_count, err := getRowCount(address, sq)
			node_row_count = append(node_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d schemas in rdpg database...\n", all_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(node_row_count); i++ {
			Expect(node_row_count[0]).To(Equal(node_row_count[i]))
		}

		Expect(node_row_count[0]).To(Equal(7))

	})

	It("Check cfsb Tables Exist", func() {
		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
		all_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		for index, _ := range rdpgsc1_nodes {
			all_nodes = append(all_nodes, rdpgsc1_nodes[index])
		}
		for index, _ := range rdpgsc2_nodes {
			all_nodes = append(all_nodes, rdpgsc2_nodes[index])
		}

		//Check all nodes
		node_row_count := make([]int, 0)
		for i := 0; i < len(all_nodes); i++ {
			address := all_nodes[i].Address
			sq := ` SELECT count(table_name) as row_count FROM information_schema.tables WHERE table_schema = 'cfsb' and table_name IN ('services','plans','instances','bindings','credentials'); `
			row_count, err := getRowCount(address, sq)
			node_row_count = append(node_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema cfsb...\n", all_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(node_row_count); i++ {
			Expect(node_row_count[0]).To(Equal(node_row_count[i]))
		}

		Expect(node_row_count[0]).To(Equal(5))

	})

	It("Check rdpg Tables Exist", func() {

		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
		all_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		for index, _ := range rdpgsc1_nodes {
			all_nodes = append(all_nodes, rdpgsc1_nodes[index])
		}
		for index, _ := range rdpgsc2_nodes {
			all_nodes = append(all_nodes, rdpgsc2_nodes[index])
		}

		//Check all nodes
		node_row_count := make([]int, 0)
		for i := 0; i < len(all_nodes); i++ {
			address := all_nodes[i].Address
			sq := ` SELECT count(table_name) as row_count FROM information_schema.tables WHERE table_schema = 'rdpg' and table_name IN ('config', 'consul_watch_notifications', 'events'); `
			row_count, err := getRowCount(address, sq)
			node_row_count = append(node_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema rdpg...\n", all_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(node_row_count); i++ {
			Expect(node_row_count[0]).To(Equal(node_row_count[i]))
		}

		Expect(node_row_count[0]).To(Equal(3))

	})

	It("Check tasks Tables Exist", func() {
		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
		all_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		for index, _ := range rdpgsc1_nodes {
			all_nodes = append(all_nodes, rdpgsc1_nodes[index])
		}
		for index, _ := range rdpgsc2_nodes {
			all_nodes = append(all_nodes, rdpgsc2_nodes[index])
		}

		//Check all nodes
		node_row_count := make([]int, 0)
		for i := 0; i < len(all_nodes); i++ {
			address := all_nodes[i].Address
			sq := ` SELECT count(table_name) as row_count FROM information_schema.tables WHERE table_schema = 'tasks' and table_name IN ('tasks','schedules'); `
			row_count, err := getRowCount(address, sq)
			node_row_count = append(node_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema tasks...\n", all_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(node_row_count); i++ {
			Expect(node_row_count[0]).To(Equal(node_row_count[i]))
		}

		Expect(node_row_count[0]).To(Equal(2))

	})

	It("Check Instance Counts", func() {

		rdpgmc_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		fmt.Println(rdpgsc1_nodes)

		//Check SC1
		sc1_instance_count := make([]int, 0)
		for i := 0; i < len(rdpgsc1_nodes); i++ {
			address := rdpgsc1_nodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL; `
			row_count, err := getRowCount(address, sq)
			sc1_instance_count = append(sc1_instance_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgsc1_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1_instance_count); i++ {
			Expect(sc1_instance_count[0]).To(Equal(sc1_instance_count[i]))
		}

		//Check SC2
		sc2_instance_count := make([]int, 0)
		for i := 0; i < len(rdpgsc2_nodes); i++ {
			address := rdpgsc2_nodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL; `
			row_count, err := getRowCount(address, sq)
			sc2_instance_count = append(sc2_instance_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgsc2_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2_instance_count); i++ {
			Expect(sc2_instance_count[0]).To(Equal(sc2_instance_count[i]))
		}

		//Check MC
		mc_instance_count := make([]int, 0)
		for i := 0; i < len(rdpgmc_nodes); i++ {
			address := rdpgmc_nodes[i].Address
			sq := ` SELECT count(*) as instance_count FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL; `
			row_count, err := getRowCount(address, sq)
			mc_instance_count = append(mc_instance_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d instances...\n", rdpgmc_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(mc_instance_count); i++ {
			Expect(mc_instance_count[0]).To(Equal(mc_instance_count[i]))
		}

		//Verify that the number of instances seen in the Management Cluster is the
		//sum of the instances from the service ClusterIPs

		totalManagementClusterInstances := mc_instance_count[0]
		totalServiceClusterInstances := sc1_instance_count[0] + sc2_instance_count[0]
		Expect(totalManagementClusterInstances).To(Equal(totalServiceClusterInstances))
		fmt.Printf("Total Management Cluster Instance Count: %d\n", totalManagementClusterInstances)
		fmt.Printf("Total Service Cluster Instance Count: %d\n", totalServiceClusterInstances)
	})

	It("Check Node Counts", func() {

		rdpgmc_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		//Check SC1
		expectedNodeCount := 2
		Expect(len(rdpgsc1_nodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Service Cluster 1 Nodes\n", len(rdpgsc1_nodes), expectedNodeCount)

		//Check SC2
		expectedNodeCount = 2
		Expect(len(rdpgsc2_nodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Service Cluster 2 Nodes\n", len(rdpgsc2_nodes), expectedNodeCount)

		//Check MC
		expectedNodeCount = 3
		Expect(len(rdpgmc_nodes)).To(Equal(expectedNodeCount))
		fmt.Printf("Found %d of %d Managment Cluster Nodes\n", len(rdpgmc_nodes), expectedNodeCount)
	})

	It("Check Scheduled Tasks Exist", func() {

		rdpgmc_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		fmt.Println(rdpgsc1_nodes)

		//Check SC1
		sc1_row_count := make([]int, 0)
		for i := 0; i < len(rdpgsc1_nodes); i++ {
			address := rdpgsc1_nodes[i].Address
			sq := ` SELECT count(*) AS row_count FROM tasks.schedules WHERE role IN ('all', 'service'); `
			row_count, err := getRowCount(address, sq)
			sc1_row_count = append(sc1_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgsc1_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1_row_count); i++ {
			Expect(sc1_row_count[0]).To(Equal(sc1_row_count[i]))
		}

		Expect(sc1_row_count[0]).To(BeNumerically(">=", 3))

		//Check SC2
		sc2_row_count := make([]int, 0)
		for i := 0; i < len(rdpgsc2_nodes); i++ {
			address := rdpgsc2_nodes[i].Address
			sq := ` SELECT count(*) AS row_count FROM tasks.schedules WHERE role IN ('all', 'service'); `
			row_count, err := getRowCount(address, sq)
			sc2_row_count = append(sc2_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgsc2_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2_row_count); i++ {
			Expect(sc2_row_count[0]).To(Equal(sc2_row_count[i]))
		}

		Expect(sc2_row_count[0]).To(BeNumerically(">=", 3))

		//Check MC
		mc_row_count := make([]int, 0)
		for i := 0; i < len(rdpgmc_nodes); i++ {
			address := rdpgmc_nodes[i].Address
			sq := ` SELECT count(*) AS row_count FROM tasks.schedules WHERE role IN ('all', 'manager'); `
			row_count, err := getRowCount(address, sq)
			mc_row_count = append(mc_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d scheduled tasks...\n", rdpgmc_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(mc_row_count); i++ {
			Expect(mc_row_count[0]).To(Equal(mc_row_count[i]))
		}

		Expect(mc_row_count[0]).To(BeNumerically(">=", 4))

	})

	It("Check For Missed Scheduled Tasks", func() {

		//Looks for any active scheduled tasks which have not been scheduled in two
		//frequency cycles

		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)
		all_nodes, _, _ := consulClient.Catalog().Service("rdpgmc", "", nil)
		for index, _ := range rdpgsc1_nodes {
			all_nodes = append(all_nodes, rdpgsc1_nodes[index])
		}
		for index, _ := range rdpgsc2_nodes {
			all_nodes = append(all_nodes, rdpgsc2_nodes[index])
		}

		//Check all nodes
		node_row_count := make([]int, 0)
		for i := 0; i < len(all_nodes); i++ {
			address := all_nodes[i].Address
			sq := ` SELECT count(*) as row_count FROM tasks.schedules WHERE last_scheduled_at + (2 * frequency) < CURRENT_TIMESTAMP AND enabled=true; `
			row_count, err := getRowCount(address, sq)
			node_row_count = append(node_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d missed scheduled tasks...\n", all_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(node_row_count); i++ {
			Expect(node_row_count[0]).To(Equal(node_row_count[i]))
		}
		//There should be no rows which have missed their schedule twice
		Expect(node_row_count[0]).To(Equal(0))

	})

	It("Check for databases known to cfsb.instances but don't exist", func() {

		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		fmt.Println(rdpgsc1_nodes)

		//Check SC1
		sc1_row_count := make([]int, 0)
		for i := 0; i < len(rdpgsc1_nodes); i++ {
			address := rdpgsc1_nodes[i].Address
			sq := `SELECT count(name) AS row_count FROM ( (SELECT dbname AS name FROM cfsb.instances) EXCEPT (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') ) AS instances_missing_databaes; `
			row_count, err := getRowCount(address, sq)
			sc1_row_count = append(sc1_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases known to cfsb.instances but don't exist...\n", rdpgsc1_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1_row_count); i++ {
			Expect(sc1_row_count[0]).To(Equal(sc1_row_count[i]))
		}
		//There should be no rows of databases which are known to cfsb.instances but don't exist
		Expect(sc1_row_count[0]).To(Equal(0))

		//Check SC2
		sc2_row_count := make([]int, 0)
		for i := 0; i < len(rdpgsc2_nodes); i++ {
			address := rdpgsc2_nodes[i].Address
			sq := `SELECT count(name) AS row_count FROM ( (SELECT dbname AS name FROM cfsb.instances) EXCEPT (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') ) AS instances_missing_databaes; `
			row_count, err := getRowCount(address, sq)
			sc2_row_count = append(sc2_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases known to cfsb.instances but don't exist...\n", rdpgsc2_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2_row_count); i++ {
			Expect(sc2_row_count[0]).To(Equal(sc2_row_count[i]))
		}
		//There should be no rows of databases which are known to cfsb.instances but don't exist
		Expect(sc2_row_count[0]).To(Equal(0))

	})

	It("Check for databases which exist and aren't known to cfsb.instances", func() {

		rdpgsc1_nodes, _, _ := consulClient.Catalog().Service("rdpgsc1", "", nil)
		rdpgsc2_nodes, _, _ := consulClient.Catalog().Service("rdpgsc2", "", nil)

		fmt.Println(rdpgsc1_nodes)

		//Check SC1
		sc1_row_count := make([]int, 0)
		for i := 0; i < len(rdpgsc1_nodes); i++ {
			address := rdpgsc1_nodes[i].Address
			sq := `SELECT count(name) as row_count FROM ( (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') EXCEPT (SELECT dbname AS name FROM cfsb.instances)) AS databases_missing_instances; `
			row_count, err := getRowCount(address, sq)
			sc1_row_count = append(sc1_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases in pg not in cfsb.instances...\n", rdpgsc1_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc1_row_count); i++ {
			Expect(sc1_row_count[0]).To(Equal(sc1_row_count[i]))
		}
		//There should be no rows of databases which are known to pg but aren't in cfsb.instances
		Expect(sc1_row_count[0]).To(Equal(0))

		//Check SC2
		sc2_row_count := make([]int, 0)
		for i := 0; i < len(rdpgsc2_nodes); i++ {
			address := rdpgsc2_nodes[i].Address
			sq := `SELECT count(name) as row_count FROM ( (SELECT datname AS name FROM pg_database WHERE datname LIKE 'd%') EXCEPT (SELECT dbname AS name FROM cfsb.instances)) AS databases_missing_instances; `
			row_count, err := getRowCount(address, sq)
			sc2_row_count = append(sc2_row_count, row_count)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases in pg not in cfsb.instances...\n", rdpgsc2_nodes[i].Node, row_count)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(sc2_row_count); i++ {
			Expect(sc2_row_count[0]).To(Equal(sc2_row_count[i]))
		}
		//There should be no rows of databases which are known to pg but aren't in cfsb.instances
		Expect(sc2_row_count[0]).To(Equal(0))

	})

})
