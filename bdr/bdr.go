package bdr

import (
	"fmt"
	"os"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

var (
	pgPort string
	SQL    map[string]string = map[string]string{
		"bdr_nodes":     `SELECT * FROM bdr.bdr_nodes;`,
		"bdr_nodes_dsn": `SELECT node_local_dsn FROM bdr.bdr_nodes;`,
	}
)

type BDR struct {
	ClusterID    string `db:"cluster_id" json:"cluster_id"`
	DB           *sqlx.DB
	ConsulClient *consulapi.Client
}

func init() {
	pgPort = os.Getenv("RDPGD_PG_PORT")
	if pgPort == "" {
		pgPort = "5432"
	}
}

func NewBDR(cluster_id string, client *consulapi.Client) (r *BDR) {
	r = &BDR{ClusterID: cluster_id, ConsulClient: client}
	return
}

// Return a list of PG Nodes for the BDR cluster with the IP addresses filled in.
func (b *BDR) PGNodes() (nodes []pg.PG, err error) {
	catalog := b.ConsulClient.Catalog()
	services, _, err := catalog.Service(b.ClusterID, "", nil)
	if err != nil {
		log.Error(fmt.Sprintf("bdr.BDR<%s>#PGNodes() ! %s", b.ClusterID, err))
		return
	}
	if len(services) == 0 {
		log.Error(fmt.Sprintf("bdr.BDR<%s>#PGNodes() ! No services found, no known nodes?!", b.ClusterID))
		return
	}
	for index, _ := range services {
		p := pg.NewPG(services[index].Address, pgPort, `postgres`, `postgres`, ``)
		nodes = append(nodes, *p)
	}
	return
}

// Question: Should we extract the BDR related functionality into a bd* package?
func (b *BDR) CreateUser(dbuser, dbpass string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#CreateUser(%s) ! %s`, dbuser, err))
	}

	for _, pg := range nodes {
		pg.Set(`database`, `postgres`)
		err = pg.CreateUser(dbuser, dbpass)
		if err != nil {
			log.Error(fmt.Sprintf(`bdr.BDR<%s>#CreateUser(%s) ! %s`, pg.IP, dbuser, err))
			return err
		}
	}
	return nil
}

func (b *BDR) CreateDatabase(dbname, owner string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#CreateDatabase(%s) ! %s`, dbname, err))
	}
	for _, pg := range nodes {
		err = pg.CreateDatabase(dbname, owner)
		if err != nil {
			log.Error(fmt.Sprintf(`bdr.BDR<%s>#CreateDatabase(%s,%s) ! %s`, pg.IP, dbname, owner, err))
			break
		}
	}
	if err != nil {
		// Cleanup in BDR currently requires droping the database and trying again...
		err = b.DropDatabase(dbname)
		if err != nil {
			log.Error(fmt.Sprintf(`bdr.BDR#CreateDatabase(%s,%s) Dropping Database due to Create Error ! %s`, dbname, owner, err))
		}
	}
	return
}

func (b *BDR) CreateExtensions(dbname string, exts []string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#CreateExtensions(%s) %+v ! %s`, dbname, exts, err))
	}
	for _, pg := range nodes {
		err = pg.CreateExtensions(dbname, exts)
		if err != nil {
			log.Error(fmt.Sprintf(`bdr.BDR<%s>#CreateExtensions(%s) %+v ! %s`, pg.IP, dbname, exts, err))
			break
		}
	}
	return
}

func (b *BDR) CreateReplicationGroup(dbname string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#CreateReplicationGroup(%s) ! %s`, dbname, err))
	}

	// TODO: Drop Database on all nodes if err != nil for any operation below
	for index, pg := range nodes {
		localNodeName := fmt.Sprintf(`%s_%d`, dbname, index)
		if index == 0 {
			err = pg.BDRGroupCreate(localNodeName, dbname)
		} else {
			err = pg.BDRGroupJoin(localNodeName, dbname, nodes[0])
		}
		if err != nil {
			log.Error(fmt.Sprintf(`bdr.BDR<%s>#CreateReplicationGroup(%s) ! %s`, pg.IP, dbname, err))
			break
		}
	}
	if err != nil {
		// Cleanup in BDR currently requires droping the database and trying again...
		err = b.DropDatabase(dbname)
		if err != nil {
			log.Error(fmt.Sprintf(`bdr.BDR#CreateReplicationGroup(%s) Dropping Database due to Create Error ! %s`, dbname, err))
		}
	}
	return err
}

// Disable all usage of database.
func (b *BDR) DisableDatabase(dbname string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#DisableDatabase(%s) ! %s`, dbname, err))
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		pg := nodes[i]
		err := pg.DisableDatabase(dbname)
		if err != nil {
			log.Error(fmt.Sprintf("bdr.BDR<%s>#DisableDatabase(%s) ! %s", pg.IP, dbname, err))
			return err
		}
	}
	return nil
}

func (b *BDR) BackupDatabase(dbname string) (err error) {
	log.Error(fmt.Sprintf("bdr.BDR#BackupDatabase(%s) TODO: IMPLEMENT", dbname))
	return nil
}

func (b *BDR) DropDatabase(dbname string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#DropDatabase(%s) ! %s`, dbname, err))
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		pg := nodes[i]
		pg.Set(`database`, `postgres`)
		err = pg.DropDatabase(dbname)
		if err != nil {
			log.Error(fmt.Sprintf("bdr.BDR<%s>#DropDatabase(%s) ! %s", pg.IP, dbname, err))
		}
	}
	return nil
}

func (b *BDR) DropUser(dbuser string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#DropUser(%s) ! %s`, dbuser, err))
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		pg := nodes[i]
		err = pg.DropUser(dbuser)
		if err != nil {
			log.Error(fmt.Sprintf("bdr.BDR<%s>#DropUser(%s) ! %s", pg.IP, dbuser, err))
		}
	}
	return nil
}

// Stop replication for given database (bdr replication group) and delete the grop on each node.
func (b *BDR) DeleteReplicationGroup(dbname string) (err error) {
	nodes, err := b.PGNodes()
	if err != nil {
		log.Error(fmt.Sprintf(`bdr.BDR#DeleteReplicationGroup(%s) ! %s`, dbname, err))
	}

	for i := len(nodes) - 1; i >= 0; i-- {
		//pg := nodes[i]
		//pg.Set(`database`, `postgres`)
		//db, err := pg.Connect()
		//if err != nil {
		//	log.Error(fmt.Sprintf("bdr.BDR<%s>#DeleteReplicationGroup(%s) ! %s", pg.IP, dbname, err))
		//	return err
		//}

		// TODO: Diable Replication for node...
		// Stop the replication
		//db.Close()
	}
	return nil
}
