package tasks

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/starkandwayne/rdpgd/log"
)

type backupParams struct {
	pgDumpPath   string `json:"pg_dump_path"`
	pgPort       string `json:"pg_port"`
	basePath     string `json:"base_path"`
	databaseName string `json:"database_name"`
	baseFileName string `json:"base_file_name"`
	node         string `json:"node"`
}

//ScheduleNewDatabaseBackups - Responsible for adding any databases which are in
//cfsb.instances and aren't already scheduled in tasks.schedules
func (t *Task) ScheduleNewDatabaseBackups(workRole string) (err error) {

	//SELECT active databases in cfsb.instances which aren't in tasks.schedules
	address := `127.0.0.1`
	sq := `SELECT name FROM ( (SELECT dbname AS name FROM cfsb.instances WHERE effective_at IS NOT NULL AND decommissioned_at IS NULL) EXCEPT (SELECT data AS name FROM tasks.schedules WHERE action = 'BackupDatabase' ) ) AS missing_databases; `
	listMissingDatabases, err := getList(address, sq)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.Task<%d>#ScheduleNewDatabaseBackups() Failed to load list of databases ! %s`, t.ID, err))
	}

	for _, databaseName := range listMissingDatabases {
		log.Trace(fmt.Sprintf("tasks.BackupDatabase() > Attempting to add %s", databaseName))
		sq = fmt.Sprintf(`INSERT INTO tasks.schedules (cluster_id,role,action,data,frequency,enabled,node_type) VALUES ('%s','service','BackupDatabase','%s','1 day'::interval, true, 'read')`, ClusterID, databaseName)
		log.Trace(fmt.Sprintf(`rdpg.insertDefaultSchedules() > %s`, sq))
		err = execQuery(address, sq)
		if err != nil {
			log.Error(fmt.Sprintf(`tasks.BackupDatabase() service task schedules ! %s`, err))
		}
	}
	return

}

//BackupDatabase - Perform a schema and database backup of a given database to local disk
func (t *Task) BackupDatabase(workRole string) (err error) {

	b := backupParams{}
	b.pgDumpPath, err = getConfigKeyValue(`pgDumpBinaryLocation`)
	if err != nil {
		return err
	}
	b.pgPort, err = getConfigKeyValue(`BackupPort`)
	if err != nil {
		return err
	}
	b.basePath, err = getConfigKeyValue(`BackupsPath`)
	if err != nil {
		return err
	}
	b.node, err = getNode()
	if err != nil {
		return err
	}
	b.databaseName = t.Data
	b.baseFileName = getBaseFileName() //Use this to keep schema and data file names the same

	err = createTargetFolder(b)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.BackupDatabase() Could not create target folder %s ! %s", b.basePath, err))
		return err
	}

	schemaFileHistory, err := createSchemaFile(b)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.BackupDatabase() Could not create schema file for database %s ! %s", b.databaseName, err))
		schemaFileHistory.status = `error`
	}
	err = insertHistory(schemaFileHistory)

	dataFileHistory, err := createDataFile(b)
	if err != nil {
		log.Error(fmt.Sprintf("tasks.BackupDatabase() Could not create data file for database %s ! %s", b.databaseName, err))
		dataFileHistory.status = `error`
	}
	err = insertHistory(dataFileHistory)

	return
}

// createTargetFolder - On the os, create the backup folder if it doesn't exist
func createTargetFolder(b backupParams) (err error) {
	err = os.MkdirAll(b.basePath+`/`+b.databaseName, 0777)
	return err
}

// createSchemaFile - Create a pg backup file which contains the schema to recreate
// the user database.
func createSchemaFile(b backupParams) (f fileHistory, err error) {

	start := time.Now()
	f.duration = 0
	f.status = `ok`
	f.backupFile = b.baseFileName + ".schema"
	f.backupPathAndFile = b.basePath + "/" + b.databaseName + "/" + f.backupFile
	f.dbname = b.databaseName
	f.node = b.node

	out, err := exec.Command(b.pgDumpPath, "-p", b.pgPort, "-U", "vcap", "-c", "-s", "-N", `"bdr"`, b.databaseName).CombinedOutput()
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.createSchemaFile() Error running pg_dump command for: %s out: %s ! %s`, b.databaseName, out, err))
		return
	}
	err = ioutil.WriteFile(f.backupPathAndFile, out, 0644)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.createSchemaFile() Error running output to file: %s ! %s`, f.backupPathAndFile, err))
		return
	}

	f.duration = int(time.Since(start).Seconds())
	return
}

// createDataFile - Create a pg backup file which contains only data which can be
// copied back to an existing schema
func createDataFile(b backupParams) (f fileHistory, err error) {

	start := time.Now()
	f.duration = 0
	f.status = `ok`
	f.backupFile = b.baseFileName + ".data"
	f.backupPathAndFile = b.basePath + "/" + b.databaseName + "/" + f.backupFile
	f.dbname = b.databaseName
	f.node = b.node

	out, err := exec.Command(b.pgDumpPath, "-p", b.pgPort, "-U", "vcap", "-a", "-N", `"bdr"`, b.databaseName).CombinedOutput()
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.createDataFile() Error running pg_dump command for: %s out: %s ! %s`, b.databaseName, out, err))
		return
	}

	err = ioutil.WriteFile(f.backupPathAndFile, out, 0644)
	if err != nil {
		log.Error(fmt.Sprintf(`tasks.createDataFile() Error running output to file: %s ! %s`, f.backupPathAndFile, err))
		return
	}

	f.duration = int(time.Since(start).Seconds())
	return

}

func getBaseFileName() (baseFileName string) {
	baseFileName = time.Now().Format("20060102150405")
	return
}

// func BackupUsers(backup_location) {
// 	fileName := backup_location + "/" + users + epochcalculation
// 	exec "pg_dumpall --globals-only ... connection info ... "
// }
