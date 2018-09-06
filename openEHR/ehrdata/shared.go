package ehrdata

import (
	"encoding/json"
	"time"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/openEHR"
	"github.com/iryonetwork/network-poc/storage/ehr"
)

func SaveAndUpload(user string, config *config.Config, ehr *ehr.Storage, client *client.Client, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	id, err := ehr.Encrypt(user, jsonData, config.EncryptionKeys[user])
	if err != nil {
		return err
	}

	return client.Upload(user, id, false)
}

func timestamp() string {
	return time.Now().Format("2006-01-02T15:04:05.999Z")
}

func ReadFromJSON(data []byte) *openEHR.All {
	out := &openEHR.All{}
	json.Unmarshal(data, &out)
	return out
}
