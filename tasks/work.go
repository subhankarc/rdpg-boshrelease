package tasks

import (
	"database/sql"
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
	workLock   *consulapi.Lock
	workLockCh <-chan struct{}
	workDB     *sqlx.DB
	workRole   string
)

func Work(role string) {
	workRole = role
	err := setMyIP()
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Work() OpenWorkDB() %s", err))
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(syscall.SIGTERM)
	}
	err = OpenWorkDB()
	if err != nil {
		log.Error(fmt.Sprintf("tasks.Work() OpenWorkDB() %s", err))
		proc, _ := os.FindProcess(os.Getpid())
		proc.Signal(syscall.SIGTERM)
	}
	defer CloseWorkDB()

	for { // TODO: only work for my role type: write vs read eg. WHERE role = 'read'
		tasks := []Task{}
		err = WorkLock()
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		sq := fmt.Sprintf(`SELECT id,cluster_id,node,role,action,data,ttl FROM tasks.tasks WHERE locked_by IS NULL AND role IN ('all','%s') AND node IN ('*','%s') ORDER BY created_at DESC LIMIT 1`, workRole, myIP)
		log.Trace(fmt.Sprintf(`tasks.Work() > %s`, sq))
		err = workDB.Select(&tasks, sq)
		if err != nil {
			WorkUnlock()
			if err == sql.ErrNoRows {
				log.Trace(`tasks.Work() No tasks found.`)
			} else {
				log.Error(fmt.Sprintf(`tasks.Work() Selecting Task ! %s`, err))
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if len(tasks) == 0 {
			WorkUnlock()
			time.Sleep(5 * time.Second)
			continue
		}
		task := tasks[0]
		err = task.Dequeue()
		if err != nil {
			log.Error(fmt.Sprintf(`tasks.Work() Task<%d>#Dequeue() ! %s`, task.ID, err))
			continue
		}
		WorkUnlock()

		// TODO: Come back and have a cleanup routine for tasks that were locked
		// but never finished past the TTL, perhaps a health check or such.
		err = task.Work()
		if err != nil {
			log.Error(fmt.Sprintf(`tasks.Task<%d>#Work() ! %s`, task.ID, err))

			sq = fmt.Sprintf(`UPDATE tasks.tasks SET locked_by=NULL, processing_at=NULL WHERE id=%d`, task.ID)
			log.Trace(fmt.Sprintf(`tasks#Work() > %s`, sq))
			_, err = workDB.Exec(sq)
			if err != nil {
				log.Error(fmt.Sprintf(`tasks.Work() Updating Task %d processing_at ! %s`, task.ID, err))
			}
			continue
		} else {
			// TODO: (t *Task) Delete()
			sq = fmt.Sprintf(`DELETE FROM tasks.tasks WHERE id=%d`, task.ID)
			log.Trace(fmt.Sprintf(`tasks#Work() > %s`, sq))
			_, err = workDB.Exec(sq)
			if err != nil {
				log.Error(fmt.Sprintf(`tasks.Work() Deleting Task %d ! %s`, task.ID, err))
				continue
			}
			log.Trace(fmt.Sprintf(`tasks.Work() Task Completed! > %+v`, task))
		}
	}
}

func WorkLock() (err error) {
	// Acquire consul for cluster to aquire right to schedule tasks.
	key := fmt.Sprintf("rdpg/%s/tasks/work/lock", os.Getenv(`RDPGD_CLUSTER`))
	client, _ := consulapi.NewClient(consulapi.DefaultConfig())
	workLock, err = client.LockKey(key)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.WorkLock() Error Locking Work Key %s ! %s", key, err))
		return
	}

	workLockCh, err = workLock.Lock(nil) // Acquire Consul K/V Lock
	if err != nil {
		log.Error(fmt.Sprintf("tasks.WorkLock() Error Aquiring Work Key lock %s ! %s", key, err))
		return
	}

	if workLockCh == nil {
		err = fmt.Errorf(`tasks.WorkLock() Work Lock not aquired.`)
	}

	return
}

func WorkUnlock() (err error) {
	if workLock != nil {
		err = workLock.Unlock()
		if err != nil {
			log.Error(fmt.Sprintf("tasks.WorkUnlock() Error Unlocking Work ! %s", err))
		}
	}
	return
}

func (t *Task) Work() (err error) {
	// TODO: Add in TTL Logic with error logging.
	switch t.Action {
	case "ScheduleBackups":
		go t.ScheduleBackups(workRole)
	case "Vacuum":
		go t.Vacuum(workRole)
	case "PrecreateDatabases":
		go t.PrecreateDatabases(workRole)
	case "ReconcileAvailableDatabases":
		go t.ReconcileAvailableDatabases(workRole)
	case "ReconcileAllDatabases":
		go t.ReconcileAllDatabases(workRole)
	case "DecommissionDatabase":
		go t.DecommissionDatabase(workRole)
	case "DecommissionDatabases":
		go t.DecommissionDatabases(workRole)
	case "Reconfigure":
		go t.Reconfigure(workRole)
	case "RemoveDatabase": // Role: all
		go t.RemoveDatabase(workRole)
	case "BackupDatabase": // Role: read
		go t.BackupDatabase(workRole)
	case "BackupAllDatabases":
		// Role: read
		go t.BackupAllDatabases(workRole)
	default:
		err = fmt.Errorf(`tasks.Work() BUG!!! Unknown Task Action %s`, t.Action)
		log.Error(fmt.Sprintf(`tasks.Work() Task %+v ! %s`, t, err))
	}
	sq := fmt.Sprintf(`DELETE FROM tasks.tasks WHERE id=%d`, t.ID)
	_, err = workDB.Exec(sq)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Work() Error deleting completed task %d ! %s`, t.ID, err))
	}
	return
}

func OpenWorkDB() (err error) {
	if workDB == nil {
		p := pg.NewPG(`127.0.0.1`, pbPort, `rdpg`, `rdpg`, pgPass)
		err := p.WaitForRegClass("tasks.tasks")
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Work() Failed connecting to %s err: %s", p.URI, err))
			return err
		}

		workDB, err = p.Connect()
		if err != nil {
			log.Error(fmt.Sprintf("tasks.Work() Failed connecting to %s err: %s", p.URI, err))
			return err
		}
	}
	return
}

func CloseWorkDB() (err error) {
	if workDB != nil {
		workDB.Close()
	}
	return
}
