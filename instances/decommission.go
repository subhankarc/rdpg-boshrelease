package instances

import (
	"fmt"

	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

func (i *Instance) Decommission() (err error) {
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("instances.Decommission() p.Connect(%s) ! %s", p.URI, err))
		return
	}
	defer db.Close()

	// TODO: i.SetIneffective()
	sq := fmt.Sprintf(`UPDATE cfsb.instances SET ineffective_at=CURRENT_TIMESTAMP WHERE dbname='%s'`, i.Database)
	log.Trace(fmt.Sprintf(`instances.Instance<%s>#Decommission() > %s`, i.InstanceID, sq))
	_, err = db.Exec(sq)
	if err != nil {
		log.Error(fmt.Sprintf("Instance#Decommission(%s) ! %s", i.InstanceID, err))
	}

	// TODO: tasks.Task{ClusterID: ,Node: ,Role: ,Action:, Data: }.Enqueue()
	// Question is how to do this without an import cycle? Some tasks require instances.
	sq = fmt.Sprintf(`INSERT INTO tasks.tasks (cluster_id,role,action,data) VALUES ('%s','all','DecommissionDatabase','%s')`, ClusterID, i.Database)
	log.Trace(fmt.Sprintf(`instances.Instance#Decommission(%s) Scheduling Instance Removal > %s`, i.InstanceID, sq))
	_, err = db.Exec(sq)
	if err != nil {
		log.Error(fmt.Sprintf(`instances.Instance#Decommission(%s) ! %s`, i.InstanceID, err))
	}
	return
}
