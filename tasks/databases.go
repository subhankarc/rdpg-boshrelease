package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/starkandwayne/rdpgd/bdr"
	"github.com/starkandwayne/rdpgd/instances"
	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
	"github.com/starkandwayne/rdpgd/uuid"
)

// Scheduled task for precreating databaes.
func (t *Task) PrecreateDatabases(workRole string) (err error) {
	if workRole != "service" { // Safety valve...
		log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases() ! Not precreating databases as we are not running on a service node..."))
		return
	}
	t.ClusterID = os.Getenv(`RDPGD_CLUSTER`)

	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases() ! %s", err))
		return
	}
	// Lock Database Creation via Consul Lock
	key := fmt.Sprintf(`rdpg/%s/database/create/lock`, t.ClusterID)
	lo := &consulapi.LockOptions{
		Key:         key,
		SessionName: fmt.Sprintf(`rdpg/%s/databases/create`, t.ClusterID),
	}
	log.Trace(fmt.Sprintf(`tasks.Task<%s>#PrecreateDatabases() Attempting to acquire database creation lock %s...`, t.ClusterID, key))
	databaseCreateLock, err := client.LockOpts(lo)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Task<%s>#PrecreateDatabases() LockKey() Error locking database creation Key %s ! %s`, t.ClusterID, key, err))
		return
	}
	databaseCreateLockCh, err := databaseCreateLock.Lock(nil)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Task<%s>#PrecreateDatabases() Lock() Error Database Creation lock %s ! %s`, t.ClusterID, key, err))
		return
	}
	if databaseCreateLockCh == nil {
		log.Error(fmt.Sprintf(`tasks.Task<%s>#PrecreateDatabases() Database Creation Lock not aquired, halting bootstrap.`, t.ClusterID))
		return
	}

	// We have the database creation lock...
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases() p.Connect() ! %s", err))
		return err
	}
	total := 1
	if len(t.Data) == 0 {
		numAvailable := 0
		sq := `SELECT count(id) FROM cfsb.instances WHERE instance_id IS NULL AND effective_at IS NOT NULL`
		err = db.Get(&numAvailable, sq)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases() selecting numAvailable ! %s", err))
			db.Close()
			return err
		}
		log.Trace(fmt.Sprintf("tasks.Task#PrecreateDatabases() found %d available databases of required %d...", numAvailable, poolSize))
		db.Close()
		total = poolSize - numAvailable
	} else {
		// In this case we were called with a number to precreate, such as from
		// admin api "precreate 100":  for 1 .. N
		// TODO: admin API endpoint for this
		n, err := strconv.Atoi(t.Data)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases() strconv.Atoi(t.Data) ! %s", err))
		} else {
			total = n
		}
		db.Close()
	}

	log.Trace(fmt.Sprintf("tasks.Task#PrecreateDatabases() Creating %d databases...", total))
	for index := 0; index < total; index++ {
		err = t.precreateDatabase(workRole, client)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases() t.PrecreateDatabases() ! %s", err))
			databaseCreateLock.Unlock()
			return err
		}
	}
	databaseCreateLock.Unlock()

	return
}

// Precreate database functionality, note that the database creation lock is
// expected to be held when this is called as this must be done in sequence.
func (t *Task) precreateDatabase(workRole string, client *consulapi.Client) (err error) {
	log.Trace(fmt.Sprintf("tasks.Task#precreateDatabase()..."))

	b := bdr.NewBDR(t.ClusterID, client)
	re := regexp.MustCompile("[^A-Za-z0-9_]")
	u1 := uuid.NewUUID().String()
	u2 := uuid.NewUUID().String()
	identifier := strings.ToLower(string(re.ReplaceAll([]byte(u1), []byte(""))))
	dbpass := strings.ToLower(string(re.ReplaceAll([]byte(u2), []byte(""))))

	i := &instances.Instance{
		ClusterID: ClusterID,
		Database:  "d" + identifier,
		User:      "u" + identifier,
		Pass:      dbpass,
	}
	// TODO: Keep the databases under rdpg schema, link to them in the
	// cfsb.instances table so that we separate the concerns of CF and databases.
	err = b.CreateUser(i.User, i.Pass)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases(%s) CreateUser(%s) ! %s", i.Database, i.User, err))
		return err
	}

	err = b.CreateDatabase(i.Database, i.User)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases(%s) CreateDatabase(%s,%s) ! %s", i.Database, i.Database, i.User, err))
		return err
	}

	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Work() Failed connecting to %s err: %s", p.URI, err))
		return err
	}
	defer db.Close()

	sq := fmt.Sprintf(`INSERT INTO cfsb.instances (cluster_id,dbname, dbuser, dbpass) VALUES ('%s','%s','%s','%s')`, ClusterID, i.Database, i.User, i.Pass)
	log.Trace(fmt.Sprintf(`tasks.precreateDatabase(%s) > %s`, i.Database, sq))
	_, err = db.Query(sq)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.precreateDatabase(%s) ! %s`, i.Database, err))
		return err
	}

	err = b.CreateExtensions(i.Database, []string{`btree_gist`, `bdr`})
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases(%s) CreateExtensions(%s,%s) ! %s", i.Database, i.Database, i.User, err))
		return err
	}

	err = b.CreateReplicationGroup(i.Database)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#PrecreateDatabases(%s) CreateReplicationGroup() ! %s", i.Database, err))
		return err
	}

	sq = fmt.Sprintf(`UPDATE cfsb.instances SET effective_at=CURRENT_TIMESTAMP WHERE dbname='%s'`, i.Database)
	log.Trace(fmt.Sprintf(`tasks.precreateDatabase(%s) > %s`, i.Database, sq))
	_, err = db.Query(sq)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.precreateDatabase(%s) ! %s`, i.Database, err))
		return err
	}

	// Tell the management cluster about the newly available database.
	// TODO: This can be in a function.
	catalog := client.Catalog()
	svcs, _, err := catalog.Service("rdpgmc", "", nil)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#precreateDatabase(%s) catalog.Service() ! %s", i.Database, err))
		return err
	}
	if len(svcs) == 0 {
		log.Error(fmt.Sprintf("tasks.Task#precreateDatabase(%s) ! No services found, no known nodes?!", i.Database))
		return err
	}
	body, err := json.Marshal(i)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#precreateDatabase(%s) json.Marchal(i) ! %s", i.Database, err))
		return err
	}
	url := fmt.Sprintf("http://%s:%s/%s", svcs[0].Address, os.Getenv("RDPGD_ADMIN_PORT"), `databases/register`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	log.Trace(fmt.Sprintf(`tasks.Task#precreateDatabase(%s) POST %s`, i.Database, url))
	// req.Header.Set("Content-Type", "application/json")
	// TODO: Retrieve from configuration in database.
	req.SetBasicAuth(os.Getenv("RDPGD_ADMIN_USER"), os.Getenv("RDPGD_ADMIN_PASS"))
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Task#precreateDatabase(%s) httpClient.Do() %s ! %s`, i.Database, url, err))
		return err
	}
	resp.Body.Close()
	return
}

// TODO: This should be remove database
func (t *Task) RemoveDatabase(workRole string) (err error) {
	// For now we assume data is simply the database name.
	key := fmt.Sprintf("rdpg/%s/work/databases/remove", os.Getenv(`RDPGD_CLUSTER`))
	client, _ := consulapi.NewClient(consulapi.DefaultConfig())
	lock, err := client.LockKey(key)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() Error aquiring lock ! %s", err))
		return
	}
	leaderCh, err := lock.Lock(nil)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() Error aquiring lock ! %s", err))
		return
	}
	if leaderCh == nil {
		log.Trace(fmt.Sprintf("tasks.Task#RemoveDatabase() > Not Leader."))
		return
	}
	log.Trace(fmt.Sprintf("tasks.Task#RemoveDatabase() > Leader."))

	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() p.Connect() ! %s", err))
		return
	}
	ids := []string{}
	sq := fmt.Sprintf(`SELECT instance_id from cfsb.instances WHERE ineffective_at IS NOT NULL AND ineffective_at < CURRENT_TIMESTAMP AND decommissioned_at IS NULL`)
	err = db.Select(&ids, sq)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() Querying for Databases to Cleanup ! %s", err))
	}
	db.Close()
	for _, id := range ids {
		db, err := p.Connect()
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() p.Connect() ! %s", err))
			return err
		}

		uri := "postgres://"
		b := bdr.NewBDR(uri, client)

		i, err := instances.FindByInstanceID(id)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase(%s) FindingInstance(%s) ! %s", i.Database, i.InstanceID, err))
			db.Close()
			continue
		}

		err = b.DisableDatabase(i.Database)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() DisableDatabase(%s) for %s ! %s", i.Database, i.InstanceID, err))
			db.Close()
			continue
		}

		err = b.BackupDatabase(i.Database)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() BackupDatabase(%s) ! %s", i.Database, err))
			db.Close()
			continue
		}

		// Question, How do we "stop" the replication group before dropping the database?
		err = b.DropDatabase(i.Database)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() DropDatabase(%s) for %s ! %s", i.Database, i.InstanceID, err))
			db.Close()
			continue
		}

		err = b.DropUser(i.User)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() DropUser(%s) for %s ! %s", i.User, i.InstanceID, err))
			db.Close()
			continue
		}

		err = b.DropDatabase(i.Database)
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Task#RemoveDatabase() DropDatabase(%s) for %s ! %s", i.Database, i.InstanceID, err))
			db.Close()
			continue
		}
	}
	db.Close()

	return
}
