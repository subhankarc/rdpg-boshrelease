package tasks

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/starkandwayne/rdpgd/instances"
	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

func (t *Task) DecommissionDatabase(workRole string) (err error) {
	log.Trace(fmt.Sprintf(`tasks.DecommissionDatabase(%s)...`, t.Data))

	i, err := instances.FindByDatabase(t.Data)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.DecommissionDatabase(%s) instances.FindByDatabase() ! %s", i.Database, err))
		return err
	}

	ips, err := i.ClusterIPs()
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Task#DecommissionDatabase(%s) i.ClusterIPs() ! %s`, i.Database, err))
		return err
	}
	if len(ips) == 0 {
		log.Error(fmt.Sprintf("tasks.Task#DecommissionDatabase(%s) ! No service cluster nodes found in Consul?!", i.Database))
		return
	}
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Task#DecommissionDatabase(%s) p.Connect(%s) ! %s", t.Data, p.URI, err))
		return err
	}
	defer db.Close()

	switch workRole {
	case "manager":
		path := fmt.Sprintf(`databases/decommission/%s`, t.Data)
		url := fmt.Sprintf("http://%s:%s/%s", ips[0], os.Getenv("RDPGD_ADMIN_PORT"), path)
		req, err := http.NewRequest("DELETE", url, bytes.NewBuffer([]byte("{}")))
		log.Trace(fmt.Sprintf(`tasks.Task#Decommission() > DELETE %s`, url))
		//req.Header.Set("Content-Type", "application/json")
		// TODO: Retrieve from configuration in database.
		req.SetBasicAuth(os.Getenv("RDPGD_ADMIN_USER"), os.Getenv("RDPGD_ADMIN_PASS"))
		httpClient := &http.Client{}
		_, err = httpClient.Do(req)
		if err != nil {
			log.Error(fmt.Sprintf(`tasks.Task#DecommissionDatabase(%s) httpClient.Do() %s ! %s`, i.Database, url, err))
			return err
		}
		// TODO: Is there anything we want to do on successful request?
	case "service":
		for _, ip := range ips {
			newTask := Task{ClusterID: ClusterID, Node: ip, Role: "all", Action: "Reconfigure", Data: "pgbouncer"}
			err = newTask.Enqueue()
			if err != nil {
				log.Error(fmt.Sprintf(`tasks.Task#DecommissionDatabase(%s) ! %s`, i.Database, err))
			}
		}
		log.Trace(fmt.Sprintf(`tasks.DecommissionDatabase(%s) TODO: Here is where we finally decommission on the service cluster...`, i.Database))
		return nil
	default:
		log.Error(fmt.Sprintf(`tasks.Task#DecommissionDatabase(%s) ! Unknown work role: '%s' -> BUG!!!`, i.Database, workRole))
		return nil
	}
	return
}

// Scheduled task to find and decommission databases that require decommissioning
func (t *Task) DecommissionDatabases(workRole string) (err error) {
	log.Trace(fmt.Sprintf(`tasks.DecommissionDatabases(%s) TODO: Regularly scheduled maintenance removal of databases...`, t.Data))
	return
}
