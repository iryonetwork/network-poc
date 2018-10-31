package personaldata

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/openEHR"
	"github.com/iryonetwork/network-poc/state"
	"github.com/iryonetwork/network-poc/storage/ehr"
)

func New(state *state.State) {
	fname := newName()
	sname := newSurname()
	name := fmt.Sprintf("%s %s", fname, sname)
	state.PersonalData = &openEHR.PersonalData{
		Shared: openEHR.Shared{
			Repeating: openEHR.Repeating{
				Composer: openEHR.Composer{
					ID:   state.EosAccount,
					Name: name,
				},
				Language: "en",
			},
			Category:  "openehr::431|persistent|",
			Timestamp: time.Now().Format("2006-01-02T15:04:05.999Z"),
		},
		PersonalDataFields: openEHR.PersonalDataFields{
			BirthDate:  randomDate(),
			Gender:     getGender(),
			FirstName:  fname,
			FamilyName: sname,
		},
	}
}

func randomDate() string {
	min := time.Date(1930, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2018, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec, err := rand.Int(rand.Reader, big.NewInt(int64(delta)))
	if err != nil {
		panic(err)
	}
	return time.Unix(sec.Int64()+min, 0).Format("2006-01-02")
}

func getGender() string {
	if time.Now().Unix()%2 == 0 {
		return "local::at0310|Male|"
	}
	return "local::at0311|Female|"
}

func Upload(state *state.State, ehr *ehr.Storage, client *client.Client) error {
	data, err := json.Marshal(state.PersonalData)
	if err != nil {
		return err
	}
	id, err := ehr.Encrypt(state.EosAccount, data, state.EncryptionKeys[state.EosAccount])
	if err != nil {
		return err
	}
	return client.Upload(state.EosAccount, id, false)
}
