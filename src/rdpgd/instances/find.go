package instances

import (
	"database/sql"
	"fmt"

	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

func FindByInstanceID(instanceID string) (i *Instance, err error) {
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("instances.FindByInstanceID(%s) ! %s", instanceID, err))
		return
	}
	defer db.Close()
	in := Instance{}
	sq := fmt.Sprintf(`SELECT id, cluster_id,instance_id, service_id, plan_id, organization_id, space_id, dbname, dbuser, dbpass FROM cfsb.instances WHERE instance_id=lower('%s') LIMIT 1`, instanceID)
	log.Trace(fmt.Sprintf(`instances.FindByInstanceID(%s) > %s`, instanceID, sq))
	err = db.Get(&in, sq)
	if err != nil {
		// TODO: Change messaging if err is sql.NoRows then say couldn't find instance with instanceID
		log.Error(fmt.Sprintf("instances.FindByInstanceID(%s) ! %s", instanceID, err))
	}
	i = &in
	return
}

func FindByDatabase(dbname string) (i *Instance, err error) {
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("instances.FindByDatabase(%s) ! %s", dbname, err))
		return
	}
	defer db.Close()
	in := Instance{}
	sq := fmt.Sprintf(`SELECT id,cluster_id, dbname, dbuser, dbpass FROM cfsb.instances WHERE dbname='%s' LIMIT 1`, dbname)
	log.Trace(fmt.Sprintf(`instances.FindByInstanceID(%s) > %s`, dbname, sq))
	err = db.Get(&in, sq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			log.Error(fmt.Sprintf("instances.FindByDatabase(%s) ! %s", dbname, err))
		}
	}
	i = &in
	return
}

func Active() (si []Instance, err error) {
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("instances.Active() p.Connect(%s) ! %s", p.URI, err))
		return
	}
	defer db.Close()

	si = []Instance{}
	// TODO: Move this into a versioned SQL Function.
	sq := `SELECT instance_id, service_id, plan_id, organization_id, space_id, dbname, dbuser, 'md5'||md5(cfsb.instances.dbpass||dbuser) as dbpass FROM cfsb.instances WHERE instance_id IS NOT NULL AND ineffective_at IS NULL `
	err = db.Select(&si, sq)
	if err != nil {
		log.Error(fmt.Sprintf("instances.Active() ! %s", err))
	}
	return
}

func All() (is []Instance, err error) {
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("instances.All() p.Connect(%s) ! %s", p.URI, err))
		return
	}
	defer db.Close()
	// TODO: Move this into a versioned SQL Function.
	// TODO: return all fields.
	sq := fmt.Sprintf(`SELECT id, cluster_id,instance_id, service_id, plan_id, organization_id, space_id, dbname, dbuser, dbpass FROM cfsb.instances`)
	log.Trace(fmt.Sprintf(`instances.All() > %s`, sq))
	err = db.Select(&is, sq)
	if err != nil {
		if err == sql.ErrNoRows {
			is = []Instance{}
		} else {
			log.Error(fmt.Sprintf("instances.All() ! %s", err))
		}
	}
	return
}

func Available() (is []Instance, err error) {
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("instances.Available() p.Connect(%s) ! %s", p.URI, err))
		return
	}
	defer db.Close()
	// TODO: Move this into a versioned SQL Function.
	sq := `SELECT cluster_id,dbname, dbuser, dbpass FROM cfsb.instances WHERE instance_id IS NULL AND effective_at IS NOT NULL AND ineffective_at IS NULL `
	log.Trace(fmt.Sprintf(`instances.Available() > %s`, sq))
	err = db.Select(&is, sq)
	if err != nil {
		if err == sql.ErrNoRows {
			is = []Instance{}
		} else {
			// TODO: Change messaging if err is sql.NoRows then say couldn't find instance with instanceId
			log.Error(fmt.Sprintf("instances.Available() ! %s", err))
		}
	}
	return
}
