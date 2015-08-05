package backups_test

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

var _ = Describe("RDPG Backups Testing...", func() {

	It("Check backups Tables Exist", func() {

		allNodes := getAllNodes()

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(table_name) as rowCount FROM information_schema.tables WHERE table_schema = 'backups' and table_name IN ('file_history'); `
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d tables in schema 'backups'...\n", allNodes[i].Node, rowCount)
		}

		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(1))
	})

	It("Check all user databases are scheduled for backups", func() {

		//Note: this one may need a sleep command in order for it to report correctly on a freshly created deployment
		allNodes := getServiceNodes()

		//Check SC
		var scRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(name) as rowCount FROM ( (SELECT dbname AS name FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL) EXCEPT (SELECT data AS name FROM tasks.schedules WHERE action = 'BackupDatabase' ) ) AS missing_databases;  `
			rowCount, err := getRowCount(address, sq)
			scRowCount = append(scRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d databases in cfsb.instances not scheduled for backups in tasks.schedules...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(scRowCount); i++ {
			Expect(scRowCount[0]).To(Equal(scRowCount[i]))
		}

		Expect(len(allNodes)).NotTo(Equal(0))

		//There should be no rows of databases which are in cfsb.instances but not in tasks.schedules
		Expect(scRowCount[0]).To(Equal(0))

	})

	It("Check all configuration defaults have been configured", func() {

		allNodes := getAllNodes()

		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('pgDumpBinaryLocation','BackupPort','BackupsPath','defaultDaysToKeepFileHistory') ;  `
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d default values configured in rdpg.config...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(4))

	})

	It("Check task DeleteBackupHistory is scheduled", func() {

		allNodes := getServiceNodes()

		//Check SC
		var scRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(action) as rowCount FROM tasks.schedules WHERE action = 'DeleteBackupHistory' AND enabled=true;  `
			rowCount, err := getRowCount(address, sq)
			scRowCount = append(scRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d entry for 'DeleteBackupHistory' in tasks.schedules...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(scRowCount); i++ {
			Expect(scRowCount[0]).To(Equal(scRowCount[i]))
		}
		Expect(len(allNodes)).NotTo(Equal(0))
		Expect(scRowCount[0]).To(Equal(1))

	})

	It("Check task ScheduleNewDatabaseBackups is scheduled", func() {

		allNodes := getServiceNodes()

		//Check SC
		var scRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := `SELECT count(action) as rowCount FROM tasks.schedules WHERE action = 'ScheduleNewDatabaseBackups' AND enabled=true;  `
			rowCount, err := getRowCount(address, sq)
			scRowCount = append(scRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d entry for 'ScheduleNewDatabaseBackups' in tasks.schedules...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(scRowCount); i++ {
			Expect(scRowCount[0]).To(Equal(scRowCount[i]))
		}
		Expect(len(allNodes)).NotTo(Equal(0))
		Expect(scRowCount[0]).To(Equal(1))

	})

	It("Check backups.file_history truncation is working", func() {

		//daysToKeep, err := getConfigKeyValue(`defaultDaysToKeepFileHistory`)
		daysToKeep := `181` //Default is 180, use defaultDaysToKeepFileHistory + 1 day
		allNodes := getAllNodes()

		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := fmt.Sprintf(`SELECT count(*) as rowCount FROM backups.file_history WHERE created_at < NOW() - '%s days'::interval; `, daysToKeep)
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d rows in backups.file_history which should have been removed via a scheduled task...\n", allNodes[i].Node, rowCount)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(0))

	})

})
