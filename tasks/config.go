package tasks

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/starkandwayne/rdpgd/log"
)

// getConfigKeyValue - Returns the key value from rdpg.config
func getConfigKeyValue(keyName string) (defaultBasePath string, err error) {
	address := `127.0.0.1`
	sq := fmt.Sprintf(`SELECT value AS keyvalue FROM rdpg.config WHERE key = '%s' AND cluster_id = '%s' ; `, keyName, ClusterID)
	keyValue, err := getList(address, sq)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error(fmt.Sprintf("tasks.BackupDatabase() No default value found for %s ! %s", keyName, err))
		} else {
			log.Error(fmt.Sprintf("tasks.BackupDatabase() Error when retrieving key value %s ! %s", keyName, err))
		}
		return ``, err
	}
	if len(keyValue) == 0 {
		log.Error(fmt.Sprintf("tasks.BackupDatabase() No value found for %s ! %s", keyName, err))
		return ``, errors.New(fmt.Sprintf("Key name %s not found", keyName))
	}
	return keyValue[0], nil
}
