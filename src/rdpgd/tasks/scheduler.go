package tasks

import (
	"fmt"
	"os"
	"syscall"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/jmoiron/sqlx"

	"github.com/starkandwayne/rdpgd/log"
	"github.com/starkandwayne/rdpgd/pg"
)

var (
	scheduleLock   *consulapi.Lock
	scheduleLockCh <-chan struct{}
	scheduleDB     *sqlx.DB
)

type Schedule struct {
	ID        int64  `db:"id" json:"id"`
	ClusterID string `db:"cluster_id" json:"cluster_id"`
	Role      string `db:"role" json:"role"`
	Action    string `db:"action" json:"action"`
	Data      string `db:"data" json:"data"`
	TTL       int64  `db:"ttl" json:"ttl"`
}

/*
Task Scheduler TODO's
- Task TTL: "Task type X should take no more than this long"
- accounting history stored in database.
- TTL based cleanup of task Queue for workers that may have imploded.
*/
func Scheduler(role string) {
	p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
	p.Set(`database`, `rdpg`)

	err := p.WaitForRegClass("tasks.schedules")
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Scheduler() p.WaitForRegClass() ! %s`, err))
	}

	scheduleDB, err := p.Connect()
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Scheduler() p.Connect() Failed connecting to %s ! %s`, p.URI, err))
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(syscall.SIGTERM)
	}
	defer scheduleDB.Close()

	for {
		err = SchedulerLock()
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		schedules := []Schedule{}
		sq := fmt.Sprintf(`SELECT id,cluster_id, role, action, data, ttl FROM tasks.schedules WHERE enabled = true AND CURRENT_TIMESTAMP >= (last_scheduled_at + frequency::interval) AND role IN ('all','%s')`, role)
		log.Trace(fmt.Sprintf(`tasks#Scheduler() Selecting Schedules > %s`, sq))
		err = scheduleDB.Select(&schedules, sq)
		if err != nil {
			log.Error(fmt.Sprintf(`tasks.Scheduler() Selecting Schedules ! %s`, err))
			SchedulerUnlock()
			time.Sleep(10 * time.Second)
			continue
		}
		for index, _ := range schedules {
			sq = fmt.Sprintf(`UPDATE tasks.schedules SET last_scheduled_at = CURRENT_TIMESTAMP WHERE id=%d`, schedules[index].ID)
			log.Trace(fmt.Sprintf(`tasks#Scheduler() %+v > %s`, schedules[index], sq))
			_, err = scheduleDB.Exec(sq)
			if err != nil {
				log.Error(fmt.Sprintf(`tasks.Scheduler() Schedule: %+v ! %s`, schedules[index], err))
				continue
			}
			task := NewTask()
			task.ClusterID = schedules[index].ClusterID
			task.Role = schedules[index].Role
			task.Action = schedules[index].Action
			task.Data = schedules[index].Data
			task.TTL = schedules[index].TTL
			err = task.Enqueue()
			if err != nil {
				log.Error(fmt.Sprintf(`tasks.Scheduler() Task.Enqueue() %+v ! %s`, task, err))
			}
		}
		SchedulerUnlock()
		time.Sleep(10 * time.Second)
	}
}

func NewSchedule() (s *Schedule) {
	return &Schedule{}
}

func SchedulerLock() (err error) {
	// Acquire consul schedulerLock for cluster to aquire right to schedule tasks.
	key := fmt.Sprintf("rdpg/%s/tasks/scheduler/lock", ClusterID)
	client, _ := consulapi.NewClient(consulapi.DefaultConfig())
	scheduleLock, err = client.LockKey(key)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.SchedulerLock() Error Locking Scheduler Key %s ! %s", key, err))
		return
	}
	scheduleLockCh, err = scheduleLock.Lock(nil)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.SchedulerLock() Error Aquiring Scheduler Key lock %s ! %s", key, err))
		return
	}

	if scheduleLockCh == nil {
		err = fmt.Errorf(`tasks.SchedulerLock() Scheduler Lock not aquired.`)
	}

	return
}

func SchedulerUnlock() (err error) {
	if scheduleLock != nil {
		err = scheduleLock.Unlock()
		if err != nil {
			log.Error(fmt.Sprintf("tasks.SchedulerUnlock() Error Unlocking Scheduler ! %s", err))
		}
	}
	return
}
