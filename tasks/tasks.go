package tasks

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/starkandwayne/rdpgd/log"
)

var (
	myIP      string
	ClusterID string
	pbPort    string
	pgPass    string
	poolSize  int
)

type Task struct {
	ID        int64  `db:"id" json:"id"`
	ClusterID string `db:"cluster_id" json:"cluster_id"`
	Node      string `db:"node" json:"node"`
	Role      string `db:"role" json:"role"`
	Action    string `db:"action" json:"action"`
	Data      string `db:"data" json:"data"`
	TTL       int64  `db:"ttl" json:"ttl"`
}

func init() {
	ClusterID = os.Getenv(`RDPGD_CLUSTER`)
	if ClusterID == "" {
		log.Error(`tasks.Scheduler() RDPGD_CLUSTER not found in environment!!!`)
	}
	pbPort = os.Getenv(`RDPGD_PB_PORT`)
	if pbPort == `` {
		pbPort = `6432`
	}
	pgPass = os.Getenv(`RDPGD_PG_PASS`)
	ps := os.Getenv(`RDPGD_POOL_SIZE`)
	if ps == "" {
		poolSize = 10
	} else {
		p, err := strconv.Atoi(ps)
		if err != nil {
			poolSize = 10
		} else {
			poolSize = p
		}
	}
}

func NewTask() *Task {
	return &Task{Node: "*"}
}

func setMyIP() (err error) {
	client, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		log.Error(fmt.Sprintf("tasks.setMyIP() consulapi.NewClient() ! %s", err))
		return
	}
	agent := client.Agent()
	info, err := agent.Self()
	myIP = info["Config"]["AdvertiseAddr"].(string)
	return
}

func (t *Task) Enqueue() (err error) {
	sq := fmt.Sprintf(`INSERT INTO tasks.tasks (cluster_id,node,role,action,data,ttl) VALUES ('%s','%s','%s','%s','%s',%d)`, t.ClusterID, t.Node, t.Role, t.Action, t.Data, t.TTL)
	log.Trace(fmt.Sprintf(`tasks.Task#Enqueue() > %s`, sq))
	for {
		OpenWorkDB()
		_, err = workDB.Exec(sq)
		if err != nil {
			re := regexp.MustCompile(`tasks_pkey`)
			if re.MatchString(err.Error()) {
				continue
			} else {
				log.Error(fmt.Sprintf(`tasks.Task#Enqueue() Insert Task %+v ! %s`, t, err))
				return
			}
		}
		break
	}
	log.Trace(fmt.Sprintf(`tasks.Task#Enqueue() Task Enqueued > %+v`, t))
	return
}

func (t *Task) Dequeue() (err error) {
	tasks := []Task{}
	sq := fmt.Sprintf(`SELECT id,node,cluster_id,role,action,data,ttl FROM tasks.tasks WHERE id=%d LIMIT 1`, t.ID)
	log.Trace(fmt.Sprintf(`tasks.Task<%d>#Dequeue() > %s`, t.ID, sq))
	OpenWorkDB()
	err = workDB.Select(&tasks, sq)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Task<%d>#Dequeue() Selecting Task %+v ! %s`, t.ID, t, err))
		return
	}
	if tasks == nil {
		log.Error(fmt.Sprintf(`tasks.Task<%d>#Dequeue() No rows returned for task %+v`, t.ID, t))
		return
	}
	t = &tasks[0]
	// TODO: Add the information for who has this task locked using IP
	sq = fmt.Sprintf(`UPDATE tasks.tasks SET locked_by='%s', processing_at=CURRENT_TIMESTAMP WHERE id=%d`, myIP, t.ID)
	log.Trace(fmt.Sprintf(`tasks.Task<%d>#Dequeue() > %s`, t.ID, sq))
	_, err = workDB.Exec(sq)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Task<%d>#Dequeue() Updating Task processing_at ! %s`, t.ID, err))
		return
	}
	log.Trace(fmt.Sprintf(`tasks.Task<%d>#Dequeue() Task Dequeued > %+v`, t.ID, t))
	return
}
