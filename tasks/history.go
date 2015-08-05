package tasks

import (
	"fmt"

	"github.com/starkandwayne/rdpgd/log"
)

type fileHistory struct {
	backupFile        string
	backupPathAndFile string
	dbname            string
	node              string
	status            string
	duration          int
}

//DeleteBackupHistory - Responsible for deleting records from backups.file_history
//older than the value in rdpg.config.key = defaultDaysToKeepFileHistory
func (t *Task) DeleteBackupHistory(workRole string) (err error) {

	daysToKeep, err := getConfigKeyValue(`defaultDaysToKeepFileHistory`)
	log.Trace(fmt.Sprintf("tasks.DeleteBackupHistory() Keeping %s days of file history in backups.file_history", daysToKeep))

	address := `127.0.0.1`
	sq := fmt.Sprintf(`DELETE FROM backups.file_history WHERE created_at < NOW() - '%s days'::interval; `, daysToKeep)

	err = execQuery(address, sq)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.DeleteBackupHistory() Error when running query %s ! %s`, sq, err))
	}

	return

}

func insertHistory(f fileHistory) (err error) {
	address := `127.0.0.1`
	sq := fmt.Sprintf(`INSERT INTO backups.file_history(cluster_id, dbname, node, file_name, action, status, duration, params) VALUES ('%s','%s','%s','%s','%s','%s',%d,'{"location":"%s"}')`, ClusterID, f.dbname, f.node, f.backupFile, `CreateBackup`, f.status, f.duration, f.backupPathAndFile)
	err = execQuery(address, sq)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.insertHistory() Error inserting record into backups.file_history, running query: %s ! %s", sq, err))
	}
	return
}
