package migrations_test

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

func execQuery(address string, sq string) (err error) {
	p := pg.NewPG(address, "7432", `rdpg`, `rdpg`, "admin")
	db, err := p.Connect()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(sq)
	return err
}

var _ = Describe("RDPG Database Migrations...", func() {

	It("Check backups.file_history table exists, otherwise create", func() {

		allNodes := getAllNodes()

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := ` SELECT count(table_name) as rowCount FROM information_schema.tables WHERE table_schema = 'backups' and table_name IN ('file_history'); `
			tableCount, err := getRowCount(address, sq)

			if tableCount == 0 {
				//Table doesn't exist, create it
				sq = `CREATE TABLE IF NOT EXISTS backups.file_history (
				  id               BIGSERIAL PRIMARY KEY NOT NULL,
					cluster_id        TEXT NOT NULL,
				  dbname            TEXT NOT NULL,
					node							TEXT NOT NULL,
					file_name					TEXT NOT NULL,
					action						TEXT NOT NULL,
					status						TEXT NOT NULL,
					params            json DEFAULT '{}'::json,
					created_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					duration          INT,
					removed_at        TIMESTAMP
				);`
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to create backups.file_history table...\n", allNodes[i].Node)
				Expect(err).NotTo(HaveOccurred())
			}

			//Now rerun and verify the table was created
			sq = ` SELECT count(table_name) as rowCount FROM information_schema.tables WHERE table_schema = 'backups' and table_name IN ('file_history'); `
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

	It("Check node_type column in tasks.tasks table exists, otherwise create", func() {

		allNodes := getAllNodes()
		tableSchema := `tasks`
		tableName := `tasks`
		columnName := `node_type`
		defaultValue := `any`

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := fmt.Sprintf(` SELECT count(table_name) as rowCount FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' AND column_name = '%s' `, tableSchema, tableName, columnName)
			columnCount, err := getRowCount(address, sq)

			if columnCount == 0 {
				//Table doesn't exist, create it

				sq := fmt.Sprintf(`ALTER TABLE %s.%s ADD COLUMN %s text;`, tableSchema, tableName, columnName)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to create '%s' column in %s.%s...\n", allNodes[i].Node, columnName, tableSchema, tableName)
				Expect(err).NotTo(HaveOccurred())

				sq = fmt.Sprintf(`ALTER TABLE %s.%s ALTER COLUMN %s SET DEFAULT '%s';`, tableSchema, tableName, columnName, defaultValue)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to create '%s' column in %s.%s, setting default value to '%s'...\n", allNodes[i].Node, columnName, tableSchema, tableName, defaultValue)
				Expect(err).NotTo(HaveOccurred())

			}
			//Now rerun and verify the column was created
			sq = fmt.Sprintf(` SELECT count(table_name) as rowCount FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' AND column_name = '%s' `, tableSchema, tableName, columnName)
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d '%s' columns in table '%s.%s'...\n", allNodes[i].Node, rowCount, columnName, tableSchema, tableName)
		}

		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(1))
	})

	It("Check node_type column in tasks.schedules table exists, otherwise create", func() {

		allNodes := getAllNodes()
		tableSchema := `tasks`
		tableName := `schedules`
		columnName := `node_type`
		defaultValue := `any`

		//Check all nodes
		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address
			sq := fmt.Sprintf(` SELECT count(table_name) as rowCount FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' AND column_name = '%s' `, tableSchema, tableName, columnName)
			columnCount, err := getRowCount(address, sq)

			if columnCount == 0 {
				//Table doesn't exist, create it

				sq := fmt.Sprintf(`ALTER TABLE %s.%s ADD COLUMN %s text;`, tableSchema, tableName, columnName)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to create '%s' column in %s.%s...\n", allNodes[i].Node, columnName, tableSchema, tableName)
				Expect(err).NotTo(HaveOccurred())

				sq = fmt.Sprintf(`ALTER TABLE %s.%s ALTER COLUMN %s SET DEFAULT '%s';`, tableSchema, tableName, columnName, defaultValue)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to create '%s' column in %s.%s, setting default value to '%s'...\n", allNodes[i].Node, columnName, tableSchema, tableName, defaultValue)
				Expect(err).NotTo(HaveOccurred())

			}
			//Now rerun and verify the column was created
			sq = fmt.Sprintf(` SELECT count(table_name) as rowCount FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' AND column_name = '%s' `, tableSchema, tableName, columnName)
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d '%s' columns in table '%s.%s'...\n", allNodes[i].Node, rowCount, columnName, tableSchema, tableName)
		}

		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}

		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(1))
	})

	It("Check default for defaultDaysToKeepFileHistory added rdpg.config", func() {

		allNodes := getAllNodes()
		configKey := `defaultDaysToKeepFileHistory`
		configValue := `180`

		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address

			sq := fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			configCount, err := getRowCount(address, sq)

			if configCount == 0 {
				//Table entry doesn't exist, create it
				sq = fmt.Sprintf(`INSERT INTO rdpg.config (key,cluster_id,value) VALUES ('%s', '%s', '%s')`, configKey, allNodes[i].ServiceName, configValue)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to insert key %s with value %s into 'rdpg.config'...\n", allNodes[i].Node, configKey, configValue)
				Expect(err).NotTo(HaveOccurred())
			}

			sq = fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d default values for key %s in rdpg.config...\n", allNodes[i].Node, rowCount, configKey)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(1))

	})

	It("Check default for BackupPort added rdpg.config", func() {

		allNodes := getAllNodes()
		configKey := `BackupPort`
		configValue := `7432`

		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address

			sq := fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			configCount, err := getRowCount(address, sq)

			if configCount == 0 {
				//Table entry doesn't exist, create it
				sq = fmt.Sprintf(`INSERT INTO rdpg.config (key,cluster_id,value) VALUES ('%s', '%s', '%s')`, configKey, allNodes[i].ServiceName, configValue)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to insert key %s with value %s into 'rdpg.config'...\n", allNodes[i].Node, configKey, configValue)
				Expect(err).NotTo(HaveOccurred())
			}

			sq = fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d default values for key %s in rdpg.config...\n", allNodes[i].Node, rowCount, configKey)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(1))

	})

	It("Check default for BackupsPath added rdpg.config", func() {

		allNodes := getAllNodes()
		configKey := `BackupsPath`
		configValue := `/var/vcap/store/pgbdr/backups`

		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address

			sq := fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			configCount, err := getRowCount(address, sq)

			if configCount == 0 {
				//Table entry doesn't exist, create it
				sq = fmt.Sprintf(`INSERT INTO rdpg.config (key,cluster_id,value) VALUES ('%s', '%s', '%s')`, configKey, allNodes[i].ServiceName, configValue)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to insert key %s with value %s into 'rdpg.config'...\n", allNodes[i].Node, configKey, configValue)
				Expect(err).NotTo(HaveOccurred())
			}

			sq = fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d default values for key %s in rdpg.config...\n", allNodes[i].Node, rowCount, configKey)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(1))

	})

	It("Check default for pgDumpBinaryLocation added rdpg.config", func() {

		allNodes := getAllNodes()
		configKey := `pgDumpBinaryLocation`
		configValue := `/var/vcap/packages/pgbdr/bin/pg_dump`

		var nodeRowCount []int
		for i := 0; i < len(allNodes); i++ {
			address := allNodes[i].Address

			sq := fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			configCount, err := getRowCount(address, sq)

			if configCount == 0 {
				//Table entry doesn't exist, create it
				sq = fmt.Sprintf(`INSERT INTO rdpg.config (key,cluster_id,value) VALUES ('%s', '%s', '%s')`, configKey, allNodes[i].ServiceName, configValue)
				err = execQuery(address, sq)
				fmt.Printf("%s: Had to insert key %s with value %s into 'rdpg.config'...\n", allNodes[i].Node, configKey, configValue)
				Expect(err).NotTo(HaveOccurred())
			}

			sq = fmt.Sprintf(`SELECT count(key) as rowCount FROM rdpg.config WHERE key IN ('%s') ;  `, configKey)
			rowCount, err := getRowCount(address, sq)
			nodeRowCount = append(nodeRowCount, rowCount)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("%s: Found %d default values for key %s in rdpg.config...\n", allNodes[i].Node, rowCount, configKey)
		}
		//Verify each database also sees the same number of records (bdr sanity check)
		for i := 1; i < len(nodeRowCount); i++ {
			Expect(nodeRowCount[0]).To(Equal(nodeRowCount[i]))
		}
		Expect(len(nodeRowCount)).NotTo(Equal(0))
		Expect(nodeRowCount[0]).To(Equal(1))

	})

})
