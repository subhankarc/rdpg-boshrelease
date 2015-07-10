package cfsb

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

type Credentials struct {
	ID         int    `db:"id"`
	BindingID  string `db:"binding_id" json:"binding_id"`
	InstanceID string `db:"instance_id" json:"instance_id"`
	URI        string `json:"uri"`
	DSN        string `json:"dsn"`
	JDBCURI    string `json:"jdbc_uri"`
	Host       string `db:"host" json:"host"`
	Port       string `db:"port" json:"port"`
	UserName   string `db:"username" json:"username"`
	Password   string `db:"password" json:"password"`
	Database   string `db:"database" json:"database"`
}

// Create Credentials in the data store
func (c *Credentials) Create() (err error) {
	log.Trace(fmt.Sprintf(`cfsb.Credentials#Create(%s,%s) ... `, c.InstanceID, c.BindingID))

	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("cfsb.Credentials#Create(%s) ! %s", c.BindingID, err))
		return
	}
	defer db.Close()

	err = c.Find()
	if err != nil { // Does not yet exist, insert the credentials.
		sq := fmt.Sprintf(`INSERT INTO cfsb.credentials (instance_id,binding_id,host,port,dbuser,dbpass,dbname) VALUES (lower('%s'),lower('%s'),'%s','%s','%s','%s','%s');`, c.InstanceID, c.BindingID, c.Host, c.Port, c.UserName, c.Password, c.Database)
		log.Trace(fmt.Sprintf(`cfsb.Credentials#Create() > %s`, sq))
		_, err = db.Exec(sq)
		if err != nil {
			log.Error(fmt.Sprintf(`cfsb.Credentials#Create(%s) ! %s`, c.BindingID, err))
		}
	} else { // Credentials already exists, return.
		return
	}
	return
}

func (c *Credentials) Find() (err error) {
	log.Trace(fmt.Sprintf(`cfsb.Find(%s) ... `, c.BindingID))

	if c.BindingID == "" {
		return errors.New("Credentials ID is empty, can not Find()")
	}
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("cfsb.Find(%s) ! %s", c.BindingID, err))
		return
	}
	defer db.Close()

	sq := fmt.Sprintf(`SELECT id,instance_id FROM cfsb.bindings WHERE binding_id=lower('%s') LIMIT 1`, c.BindingID)
	log.Trace(fmt.Sprintf(`cfsb.Find(%s) > %s`, c.BindingID, sq))
	err = db.Get(c, sq)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error(fmt.Sprintf("cfsb.Find(%s) ! Could not find binding with given Credentials ID", c.BindingID))
		} else {
			log.Error(fmt.Sprintf("cfsb.Find(%s) ! %s", c.BindingID, err))
		}
	}
	return
}

func (c *Credentials) Remove() (err error) {
	log.Trace(fmt.Sprintf(`cfsb.Credentials#Remove(%s) ... `, c.BindingID))
	err = c.Find()
	if err != nil {
		log.Error(fmt.Sprintf(`cfsb.Credentials#Remove(%s) ! %s`, c.BindingID, err))
		return
	}
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	db, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf("cfsb.Credentials#Remove(%s) ! %s", c.BindingID, err))
		return
	}
	defer db.Close()

	// TODO: Scheduled background task that does any cleanup necessary for an
	// unbinding (remove credentials?)
	sq := fmt.Sprintf(`UPDATE cfsb.bindings SET ineffective_at=CURRENT_TIMESTAMP WHERE binding_id=lower('%s')`, c.BindingID)
	log.Trace(fmt.Sprintf(`cfsb.Credentials#Remove(%s) > %s`, c.BindingID, sq))
	_, err = db.Exec(sq)
	if err != nil {
		log.Error(fmt.Sprintf(`cfsb.Credentials#Remove(%s) ! %s`, c.BindingID, err))
	}
	return
}
