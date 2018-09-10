package ehrdata

import (
	"encoding/json"
	"fmt"
	"strings"
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

func addUnit(value, unit string) string {
	return fmt.Sprintf("%s,%s", value, unit)
}

type Entry struct {
	Timestamp []string
	Value     []string
}

func ExtractEhrData(owner string, ehr *ehr.Storage, config *config.Config) (*map[string]Entry, error) {
	listOfData := []*openEHR.All{}
	for _, id := range ehr.ListIds(config.EosAccount) {
		datajson, err := ehr.Decrypt(owner, id, config.EncryptionKeys[owner])
		if err != nil {
			return nil, err
		}
		data := ReadFromJSON(datajson)
		listOfData = append(listOfData, data)
	}

	return setDataByTime(listOfData), nil
}

func setDataByTime(data []*openEHR.All) *map[string]Entry {
	out := make(map[string]Entry)
	for _, v := range data {
		if v.Category == "433|event" {
			if v.Weight.Measure != "" {
				entry := out["weight"]
				entry.Value = append(entry.Value, removeUnit(v.Weight.Measure))
				entry.Timestamp = append(entry.Timestamp, v.Weight.Ts)
				out["weight"] = entry
			}
			if v.Glucose.Measure != "" {
				entry := out["glucose"]
				entry.Value = append(entry.Value, removeUnit(v.Glucose.Measure))
				entry.Timestamp = append(entry.Timestamp, v.Glucose.Ts)
				out["glucose"] = entry
			}
			if v.BloodPressure.Systolic != "" {
				entry := out["bpSys"]
				entry.Value = append(entry.Value, removeUnit(v.BloodPressure.Systolic))
				entry.Timestamp = append(entry.Timestamp, v.BloodPressure.Ts)
				out["bpSys"] = entry
			}
			if v.BloodPressure.Diastolyc != "" {
				entry := out["bpDia"]
				entry.Value = append(entry.Value, removeUnit(v.BloodPressure.Diastolyc))
				entry.Timestamp = append(entry.Timestamp, v.BloodPressure.Ts)
				out["bpDia"] = entry
			}
		}
	}

	return orderDataByTime(&out)
}

func orderDataByTime(data *map[string]Entry) *map[string]Entry {
	out := make(map[string]Entry)
	for k, entry := range *data {
		out[k] = sortEntry(entry)
	}

	return &out
}

func sortEntry(e Entry) Entry {
	indexOrder := []int{}
	times := []time.Time{}

	// Get the order
	for j, v := range e.Timestamp {
		// j = index in entry
		// i = index in time order
		i := j
		ts, err := time.Parse("2006-01-02T15:04:05.999Z", v)
		if err != nil {
			panic(err)
		}

		for {
			if i <= 0 {
				times = append([]time.Time{ts}, times...)
				indexOrder = append([]int{j}, indexOrder...)
				break
			}

			if ts.After(times[i-1]) {
				indexOrder = append(indexOrder[:i], append([]int{j}, indexOrder[i:]...)...)
				times = append(times[:i], append([]time.Time{ts}, times[i:]...)...)
				break
			}
			i--
		}
	}

	// Order entries
	values := []string{}
	for _, i := range indexOrder {
		values = append(values, e.Value[i])
	}
	timestamps := []string{}
	for _, i := range indexOrder {
		timestamps = append(timestamps, e.Timestamp[i])
	}

	return Entry{Value: values, Timestamp: timestamps}
}

func removeUnit(in string) string {
	return strings.Split(in, ",")[0]
}
