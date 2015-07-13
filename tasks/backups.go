package tasks

import (
	"fmt"

	"github.com/starkandwayne/rdpgd/log"
)

// Insert start/stop/(status stuff) into history.backups:
//   kind {backup,restore,s3upload,...},
//   action {start,stop}
//   file location/status,
//   s3 bucket location
// Insert start/stop/(status stuff) into history.restores
// host role/type that Task applies to eg. write/read
func (t *Task) ScheduleBackups(workRole string) (err error) {
	log.Trace(fmt.Sprintf(`tasks.ScheduleBackups(%s)...`, t.Data))
	return
}

func (t *Task) BackupDatabase(workRole string) (err error) {
	//key := fmt.Sprintf("rdpg/%s/work/database/%s/backup", os.Getenv(`RDPGD_CLUSTER`),data)
	//client, _ := api.NewClient(api.DefaultConfig())
	//lock, err := client.LockKey()
	//if err != nil {
	//	log.Error(fmt.Sprintf("worker.BackupDatabase() Error aquiring lock ! %s", err))
	//	return
	//}
	//leaderCh, err := lock.Lock(nil)
	//if err != nil {
	//	log.Error(fmt.Sprintf("worker.BackupDatabase() Error aquiring lock ! %s", err))
	//	return
	//}
	//if leaderCh == nil {
	//	log.Trace(fmt.Sprintf("worker.BackupDatabase() > Not Leader."))
	//	return
	//}
	//log.Trace(fmt.Sprintf("worker.BackupDatabase() > Leader."))

	// Be sure to keep audit history in the rdpg database backups & audit schema.
	// func BackupDatabase(dbname, backup_location) {
	// 	fileName := backup_location + "/" + dbname + epochcalculation
	// 	exec "pg_dump -Fc ... connection info ... "
	// }
	//func BackupDatabase(dbname, backup_location) {
	//	start_at := now
	//	fileName := backup_location + "/" + dbname + epochcalculation
	//	host := somehow get the host or ip of the worker running this task
	//	exec "pg_dump -Fc ... connection info ... "
	//
	//	//Log Backup History
	//	sql := `INSERT INTO history.backup_restores (host, action, started_at, finished_at, file_location, dbname)
	//	        VALUES (` + host + ",'backup'," + start_at + "," + now() + ",'" + fileName + "','" + dbname + "'"
	//}
	//

	return
}

func (t *Task) BackupAllDatabases(workRole string) (err error) {
	//key := fmt.Sprintf("rdpg/%s/work/databases/backup", os.Getenv(`RDPGD_CLUSTER`),data)
	//client, _ := api.NewClient(api.DefaultConfig())
	//lock, err := client.LockKey()
	//if err != nil {
	//	log.Error(fmt.Sprintf("worker.BackupAllDatabases() Error aquiring lock ! %s", err))
	//	return
	//}
	//leaderCh, err := lock.Lock(nil)
	//if err != nil {
	//	log.Error(fmt.Sprintf("worker.BackupAllDatabases() Error aquiring lock ! %s", err))
	//	return
	//}
	//if leaderCh == nil {
	//	log.Trace(fmt.Sprintf("worker.BackupAllDatabases() > Not Leader."))
	//	return
	//}
	//log.Trace(fmt.Sprintf("worker.BackupAllDatabases() > Leader."))

	// Be sure to keep audit history in the rdpg database backups & audit schema.
	//
	// start_at := now
	// 	//Get list of databases
	// p := pg.NewPG(`127.0.0.1`, pgPort, `rdpg`, `rdpg`, pgbPass)
	// db, err := p.Connect()
	// if err != nil {
	// 	log.Error(fmt.Sprintf("cfsb#CreateBinding(%s) ! %s", bindingId, err))
	// 	return
	// }
	// defer db.Close()

	// 	dbList := rdpg.GetDBList()
	//
	// 	//Get backup location
	// 	backup_location := GetBackupLocation()
	//
	// 	//Perform backup for each DB
	// 	for each dbname in dbList
	// 		BackupDatabase(dbname, backup_location)
	// 	next
	// 	//Perform pg_dumpall for logins/users
	// 	BackupUsers(backup_location)

	return
}

// func BackupUsers(backup_location) {
// 	fileName := backup_location + "/" + users + epochcalculation
// 	exec "pg_dumpall --globals-only ... connection info ... "
// }
