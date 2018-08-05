package main

import (
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/eos"
	"github.com/lucasjones/reggen"
	"github.com/segmentio/ksuid"

	"github.com/eoscanada/eos-go/ecc"
	"github.com/iryonetwork/network-poc/config"
)

type handlers struct {
	config *config.Config
	log    *logger.Log
	eos    *eos.Storage
	hub    *ws.Hub
}

type uploadResponse struct {
	Err       []string `json:"error,omitempty"`
	FileID    string   `json:"fileID,omitempty"`
	EhrID     string   `json:"ehrID,omitempty"`
	CreatedAt string   `json:"createdAt,omitempty"`
}

func (h *handlers) UploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	h.log.Debugf("API got request \nFrom: %s \n For:%s", r.Form["owner"][0], r.Form["account"][0])

	// create response
	response := uploadResponse{}

	// check if access is granted
	if r.Form["account"][0] != r.Form["owner"][0] {
		access, err := h.eos.AccessGranted(r.Form["owner"][0], r.Form["account"][0])

		if err != nil {
			response.Err = append(response.Err, err.Error())
			json.NewEncoder(w).Encode(response)
			return
		}
		if !access {
			response.Err = append(response.Err, "Account does not have access to owner")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// check if account and key match
	auth, err := h.eos.CheckAccountKey(r.Form["account"][0], r.Form["key"][0])
	if err != nil {
		response.Err = append(response.Err, err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}
	if !auth {
		response.Err = append(response.Err, "Account and key does not match")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if signature is correct
	key, err := ecc.NewPublicKey(r.Form["key"][0])
	if err != nil {
		response.Err = append(response.Err, err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}
	// reconstruct signature
	sign, err := ecc.NewSignature(r.Form["sign"][0])
	if err != nil {
		response.Err = append(response.Err, err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}
	// get hash
	data := []byte(r.Form["data"][0])
	hash := getHash(data)

	// verify signature
	if !sign.Verify(hash, key) {
		response.Err = append(response.Err, "Data could not be verified")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle file saving
	// create dir
	owner := r.Form["owner"][0]
	os.MkdirAll("ehr/"+owner, os.ModePerm)
	fid := ksuid.New().String()
	// create file
	f, err := os.Create("ehr/" + owner + "/" + fid)
	if err != nil {
		response.Err = append(response.Err, err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}
	defer f.Close()
	// add data to file
	_, err = f.WriteString(string(data))
	if err != nil {
		response.Err = append(response.Err, err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}

	//Generate response and save it to md
	// create file
	f, err = os.Create("ehr/" + owner + "/md_" + fid)
	defer f.Close()

	if err != nil {
		response.Err = append(response.Err, err.Error())
		json.NewEncoder(w).Encode(response)
		return
	}
	// write md
	response.FileID = fid
	f.WriteString("f=" + fid + "\n")
	response.EhrID = r.Form["ehrID"][0]
	f.WriteString("e=" + r.Form["ehrID"][0] + "\n")
	ts := time.Now().Format("2006-01-02T15:04:05.999Z")
	response.CreatedAt = ts
	f.WriteString("t=" + ts + "\n")

	json.NewEncoder(w).Encode(response)
}

func getHash(in []byte) []byte {
	sha := sha256.New()
	sha.Write(in)
	return sha.Sum(nil)
}

type lsResponse struct {
	Err   []string `json:"error,omitempty"`
	Files []lsFile `json:"files,omitempty"`
}
type lsFile struct {
	FileID    string `json:"fileID,omitempty"`
	EhrID     string `json:"ehrID,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func (h *handlers) lsHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	h.log.Debugf("Got request: ls(%v)", r.Form["account"][0])
	response := lsResponse{}
	files, err := filepath.Glob("./ehr/" + r.Form["account"][0] + "/*")
	if err != nil {
		response.Err = append(response.Err, err.Error())
	} else {
		for _, f := range files {
			if filepath.Base(f)[:2] == "md" {
				response.Files = append(response.Files, parseLs(f))
			}
		}
	}
	h.log.Debugf("Sending: %v", response)
	json.NewEncoder(w).Encode(response)
}

func parseLs(name string) lsFile {
	// read
	f, _ := ioutil.ReadFile(name)
	c := strings.Split(string(f), "\n")
	// parse
	fid := strings.TrimLeft(strings.TrimRight(c[0], "\n"), "f=")
	eid := strings.TrimLeft(strings.TrimRight(c[1], "\n"), "e=")
	ts := strings.TrimLeft(strings.TrimRight(c[2], "\n"), "t=")
	return lsFile{fid, eid, ts}
}

func (h *handlers) downloadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	req := r.Form["fileID"][0]
	account := r.Form["account"][0]
	// check if file exists
	if _, err := os.Stat("ehr/" + account); !os.IsNotExist(err) {
		_, err := os.Stat("ehr/" + account + "/" + req)
		if err == nil {
			// get its metadata
			if parseLs("ehr/"+account+"/md_"+req).EhrID == r.Form["ehrID"][0] {
				// return file
				f, _ := ioutil.ReadFile("ehr/" + account + "/" + req)
				json.NewEncoder(w).Encode(f)
			} else {
				json.NewEncoder(w).Encode("ERROR: ehrID error")
			}
		} else {
			json.NewEncoder(w).Encode("ERROR:" + err.Error())
		}
	} else {
		json.NewEncoder(w).Encode("ERROR: Account does not exists")
	}
}

type createAccResponse struct {
	Err  []string `json:"error,omitempty"`
	Name string   `json:"account,omitempty"`
}

func (h *handlers) createaccHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	response := createAccResponse{}
	key := r.Form["key"][0]
	accountname, err := h.getAccName()
	if err != nil {
		response.Err = append(response.Err, err.Error())
		json.NewEncoder(w).Encode(response)
	}
	h.log.Debugf("Attempting to create account %s", accountname)

	err = h.eos.CreateAccount(accountname, key)
	if err != nil {
		response.Err = append(response.Err, err.Error())
	} else {
		response.Name = accountname
	}
	json.NewEncoder(w).Encode(response)
}

// Generate random name that satisfies EOS
// regex: "iryo[a-z1-5]{8}"
func (h *handlers) getAccName() (string, error) {
	g, err := reggen.NewGenerator("[a-z1-5]{8}")
	if err != nil {
		return "", err
	}
	accname := "iryo" + g.Generate(8)

	for h.eos.CheckExists(accname) {
		accname = "iryo" + g.Generate(8)
	}
	return accname, nil
}
