package rdpg

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

// Initialize the rdpg system database schemas.
func (r *RDPG) InitSchema(role string) (err error) {
	log.Trace(fmt.Sprintf(`rdpg.RDPG<%s>#InitSchema() Initializing Schema for Cluster...`, ClusterID))

	var name string
	p := pg.NewPG(`127.0.0.1`, pgPort, `rdpg`, `rdpg`, pgPass)

	p.BDRJoinWaitForReady()

	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf(`rdpg.RDPG#InitSchema(%s) Opening db connection ! %s`, role, err))
		return err
	}
	defer db.Close()

	_, err = db.Exec(`SELECT bdr.bdr_node_join_wait_for_ready();`)
	if err != nil {
		log.Error(fmt.Sprintf(`RDPG#initSchema() bdr.bdr_node_join_wait_for_ready ! %s`, err))
	}

	ddlLockRE := regexp.MustCompile(`cannot acquire DDL lock`)
	for { // Retry loop for acquiring DDL schema lock.
		log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", "rdpg_schemas"))
		_, err = db.Exec(SQL["rdpg_schemas"])
		if err != nil {
			if ddlLockRE.MatchString(err.Error()) {
				log.Trace("RDPG#initSchema() DDL Lock not available, waiting...")
				time.Sleep(1 * time.Second)
				continue
			}
			log.Error(fmt.Sprintf("RDPG#initSchema() ! %s", err))
		}
		break
	}

	keys := []string{
		"create_table_cfsb_services",
		"create_table_cfsb_plans",
		"create_table_cfsb_instances",
		"create_table_cfsb_bindings",
		"create_table_cfsb_credentials",
		"create_table_tasks_schedules",
		"create_table_tasks_tasks",
		"create_table_rdpg_consul_watch_notifications",
		"create_table_rdpg_events",
		"create_table_rdpg_config",
	}
	for _, key := range keys {
		k := strings.Split(strings.Replace(strings.Replace(key, "create_table_", "", 1), "_", ".", 1), ".")
		sq := fmt.Sprintf(`SELECT table_name FROM information_schema.tables where table_schema='%s' AND table_name='%s';`, k[0], k[1])

		log.Trace(fmt.Sprintf("RDPG#initSchema() %s", sq))
		if err := db.QueryRow(sq).Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", key))
				_, err = db.Exec(SQL[key])
				if err != nil {
					log.Error(fmt.Sprintf("RDPG#initSchema() ! %s", err))
				}
			} else {
				log.Error(fmt.Sprintf("rdpg.initSchema() ! %s", err))
			}
		}
	}

	err = insertDefaultSchedules(role, db)
	if err != nil {
		log.Error(fmt.Sprintf(`rdpg.initSchema() service task schedules ! %s`, err))
	}

	sq := fmt.Sprintf(`INSERT INTO rdpg.config (key,cluster_id,value) VALUES ('BackupsPath', '%s','/var/vcap/store/pgbdr/backups')`, ClusterID)
	log.Trace(fmt.Sprintf(`rdpg.InitSchema() > %s`, sq))
	_, err = db.Exec(sq)
	if err != nil {
		log.Error(fmt.Sprintf(`rdpg.initSchema() service task schedules ! %s`, err))
	}

	// TODO: Move initial population of services out of rdpg to Admin API.
	if err := db.QueryRow(`SELECT name FROM cfsb.services WHERE name='rdpg' LIMIT 1;`).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			if _, err = db.Exec(SQL["insert_default_cfsb_services"]); err != nil {
				log.Error(fmt.Sprintf("rdpg.initSchema(insert_default_cfsb_services) %s", err))
			}
		} else {
			log.Error(fmt.Sprintf("rdpg.initSchema() ! %s", err))
		}
	}

	// TODO: Move initial population of services out of rdpg to Admin API.
	if err = db.QueryRow(`SELECT name FROM cfsb.plans WHERE name='shared' LIMIT 1;`).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			if _, err = db.Exec(SQL["insert_default_cfsb_plans"]); err != nil {
				log.Error(fmt.Sprintf("rdpg.initSchema(insert_default_cfsb_plans) %s", err))
			}
		} else {
			log.Error(fmt.Sprintf("rdpg.initSchema() ! %s", err))
		}
	}
	db.Close()

	cluster, err := NewCluster(ClusterID, r.ConsulClient)
	for _, pg := range cluster.Nodes {
		pg.PG.Set(`database`, `postgres`)

		db, err := pg.PG.Connect()
		if err != nil {
			log.Error(fmt.Sprintf("RDPG#DropUser(%s) %s ! %s", name, pg.PG.IP, err))
		}
		log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", "postgres_schemas"))
		_, err = db.Exec(SQL["postgres_schemas"])
		if err != nil {
			log.Error(fmt.Sprintf("RDPG#initSchema() ! %s", err))
		}

		keys = []string{ // These are for the postgres database only
			"create_function_rdpg_disable_database",
		}
		for _, key := range keys {
			k := strings.Split(strings.Replace(strings.Replace(key, "create_function_", "", 1), "_", ".", 1), ".")
			// TODO: move this into a pg.PG#FunctionExists()
			sq := fmt.Sprintf(`SELECT routine_name FROM information_schema.routines WHERE routine_type='FUNCTION' AND routine_schema='%s' AND routine_name='%s';`, k[0], k[1])

			log.Trace(fmt.Sprintf("RDPG#initSchema() %s", sq))
			if err := db.QueryRow(sq).Scan(&name); err != nil {
				if err == sql.ErrNoRows {
					log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", key))
					_, err = db.Exec(SQL[key])
					if err != nil {
						log.Error(fmt.Sprintf("RDPG#initSchema() %s", err))
					}
				} else {
					log.Error(fmt.Sprintf("rdpg.initSchema() %s", err))
					db.Close()
					return err
				}
			}
		}

		db.Close()
	}
	log.Info(fmt.Sprintf(`rdpg.RDPG<%s>#InitSchema() Schema Initialized.`, ClusterID))
	return nil
}

/*
Default Schedules, general then by role.
*/
func insertDefaultSchedules(role string, db *sqlx.DB) (err error) {
	log.Trace(fmt.Sprintf(`rdpg.insertDefaultSchedules(%s)...`, role))

	// role == 'all':
	sq := fmt.Sprintf(`INSERT INTO tasks.schedules (cluster_id,role,action,data,frequency,enabled) VALUES ('%s','all','Vacuum','tasks.tasks','5 minutes'::interval, true) `, ClusterID)
	log.Trace(fmt.Sprintf(`rdpg.insertDefaultSchedules(%s) > %s`, role, sq))
	re := regexp.MustCompile(`global sequence.*not initialized yet`)
	for { // Ensure that we wait for global sequence initialization (post create)
		_, err = db.Exec(sq)
		if err != nil {
			if re.MatchString(err.Error()) {
				time.Sleep(1 * time.Second)
				continue
			} else {
				log.Error(fmt.Sprintf(`rdpg.insertDefaultSchedules() service task schedules ! %s`, err))
				return err
			}
		}
		break
	}

	if role == "manager" {
		sq := fmt.Sprintf(`INSERT INTO tasks.schedules (cluster_id,role,action,data,frequency,enabled) VALUES ('%s','manager','ScheduleBackups','','1 minute'::interval, true)`, ClusterID)
		log.Trace(fmt.Sprintf(`rdpg.insertDefaultSchedules() > %s`, sq))
		_, err = db.Exec(sq)
		if err != nil {
			log.Error(fmt.Sprintf(`rdpg.insertDefaultSchedules() service task schedules ! %s`, err))
		}

		sq = fmt.Sprintf(`INSERT INTO tasks.schedules (cluster_id,role,action,data,frequency,enabled) VALUES ('%s','manager','ReconcileAvailableDatabases','','1 minute'::interval, true), ('%s','manager','ReconcileAllDatabases','','5 minutes'::interval, true)`, ClusterID, ClusterID)
		log.Trace(fmt.Sprintf(`rdpg.insertDefaultSchedules() > %s`, sq))
		_, err = db.Exec(sq)
		if err != nil {
			log.Error(fmt.Sprintf(`rdpg.insertDefaultSchedules() service task schedules ! %s`, err))
		}
	}

	if role == "service" {
		// TODO: Move initial population of services out of rdpg to Admin API.
		sq := fmt.Sprintf(`INSERT INTO tasks.schedules (cluster_id,role,action,data,frequency,enabled) VALUES ('%s','service','PrecreateDatabases','','1 minute'::interval, true)`, ClusterID)
		log.Trace(fmt.Sprintf(`rdpg.insertDefaultSchedules() > %s`, sq))
		_, err = db.Exec(sq)
		if err != nil {
			log.Error(fmt.Sprintf(`rdpg.insertDefaultSchedules() service task schedules ! %s`, err))
		}

		sq = fmt.Sprintf(`INSERT INTO tasks.schedules (cluster_id,role,action,data,frequency,enabled) VALUES ('%s','service','DecommissionDatabases','','15 minutes'::interval, true)`, ClusterID)
		log.Trace(fmt.Sprintf(`rdpg.insertDefaultSchedules() > %s`, sq))
		_, err = db.Exec(sq)
		if err != nil {
			log.Error(fmt.Sprintf(`rdpg.insertDefaultSchedules() service task schedules ! %s`, err))
		}
	}

	return
}
