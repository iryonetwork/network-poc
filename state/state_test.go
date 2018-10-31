package state

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/openEHR"
)

func TestPersistence(t *testing.T) {
	// retrieve a temporary path
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Couldn't create temporary persistence db file")
	}
	path := file.Name()
	file.Close()
	defer os.Remove(path)

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		t.Fatalf("Couldn't generate encryption key for temporary persistence db")
	}

	cfg := config.Config{
		PersistentState:              true,
		PersistentStateStoragePath:   path,
		PersistentStateEncryptionKey: base64.StdEncoding.EncodeToString(key),
	}

	// initialize state
	s1, err := New(&cfg, logger.New(&cfg))
	if err != nil {
		t.Fatalf("Couldn't initialize state with persistance")
	}

	// add test data to state
	fillStateWithTestData(s1)

	// close state to dump data to persistence db
	s1.Close()

	// initialize state once again with the same path
	s2, err := New(&cfg, logger.New(&cfg))
	if err != nil {
		t.Fatalf("Couldn't initialize state with persistance")
	}
	defer s2.Close()

	// compare state values
	if !reflect.DeepEqual(s1.IsDoctor, s2.IsDoctor) {
		t.Errorf("Expected IsDoctor to equal\n%+v\ngot\n%+v", s1.IsDoctor, s2.IsDoctor)
	}
	if !reflect.DeepEqual(s1.EosPrivate, s2.EosPrivate) {
		t.Errorf("expected EosPrivate to equal\n%+v\ngot\n%+v", s1.EosPrivate, s2.EosPrivate)
	}
	if !reflect.DeepEqual(s1.EosAccount, s2.EosAccount) {
		t.Errorf("expected EosAccount to equal\n%+v\ngot\n%+v", s1.EosAccount, s2.EosAccount)
	}
	if !reflect.DeepEqual(s1.PersonalData, s2.PersonalData) {
		fmt.Println("Expected")
		printJson(s1.PersonalData)
		fmt.Println("Got")
		printJson(s2.PersonalData)
		t.Errorf("expected PersonalData to equal\n%+v\ngot\n%+v", s1.PersonalData, s2.PersonalData)
	}
	if !reflect.DeepEqual(s1.EncryptionKeys, s2.EncryptionKeys) {
		fmt.Println("Expected")
		printJson(s1.EncryptionKeys)
		fmt.Println("Got")
		printJson(s2.EncryptionKeys)
		t.Errorf("expected EncryptionKeys to equal\n%+v\ngot\n%+v", s1.EncryptionKeys, s2.EncryptionKeys)
	}
	if !reflect.DeepEqual(s1.RSAKey, s2.RSAKey) {
		fmt.Println("Expected")
		printJson(s1.RSAKey)
		fmt.Println("Got")
		printJson(s2.RSAKey)
		t.Errorf("expected RSAKey to equal\n%+v\ngot\n%+v", s1.RSAKey, s2.RSAKey)
	}
	if !reflect.DeepEqual(s1.Connections, s2.Connections) {
		fmt.Println("Expected")
		printJson(s1.Connections)
		fmt.Println("Got")
		printJson(s2.Connections)
		t.Errorf("expected Connections to equal\n%+v\ngot\n%+v", s1.Connections, s2.Connections)
	}
	if !reflect.DeepEqual(s1.Directory, s2.Directory) {
		fmt.Println("Expected")
		printJson(s1.Directory)
		fmt.Println("Got")
		printJson(s2.Directory)
		t.Errorf("expected Directory to equal\n%+v\ngot\n%+v", s1.Directory, s2.Directory)
	}
}

func fillStateWithTestData(s *State) {
	s.IsDoctor = true
	s.EosPrivate = "test_eos_private"
	s.EosAccount = "test_eos_account"
	s.PersonalData = &openEHR.PersonalData{
		Shared: openEHR.Shared{
			Repeating: openEHR.Repeating{
				Composer: openEHR.Composer{
					ID:   "test ID",
					Name: "test name",
				},
				Language: "en",
			},
			Category:  "openehr::431|persistent|",
			Timestamp: time.Now().Format("2006-01-02T15:04:05.999Z"),
		},
		PersonalDataFields: openEHR.PersonalDataFields{
			BirthDate:  time.Now().Format("2006-01-02"),
			Gender:     "local::at0311|Female|",
			FirstName:  "test first name",
			FamilyName: "test family name",
		},
	}
	s.EncryptionKeys["some_account2"] = []byte("some_key")

	var err error
	s.RSAKey, err = rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	s.Connections.GrantedTo = append(s.Connections.GrantedTo, "some_account1")
	s.Connections.WithKey = append(s.Connections.WithKey, "some_account2")
	s.Connections.WithoutKey = append(s.Connections.WithoutKey, "some_account3")

	rsaKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}
	pubKey, ok := rsaKey.Public().(*rsa.PublicKey)
	if !ok {
		panic("failed to get public key")
	}
	s.Connections.Requested["some_acount"] = Request{Key: pubKey, CustomData: "some_custom_data"}

	s.Directory["some_account"] = "some_name"
}

func printJson(item interface{}) {
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(item)
}
