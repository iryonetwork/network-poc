package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/eoscanada/eos-go/ecc"
	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/eos"
	"github.com/iryonetwork/network-poc/storage/token"
	"github.com/iryonetwork/network-poc/storage/ws"
	"github.com/lucasjones/reggen"
)

type handlers struct {
	config *config.Config
	log    *logger.Log
	eos    *eos.Storage
	hub    *ws.Hub
	token  *token.TokenList
}

func (h *handlers) loginHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Debugf("Got login request")
	r.ParseForm()
	keystr := r.Form["key"][0]
	var id string
	exists := false
	// if name is provided use it
	if name, ok := r.Form["account"]; ok {
		id = name[0]
		// check that key is part of account
		if ok, err := h.eos.CheckAccountKey(id, keystr); !ok {
			if err != nil {
				writeErrorBody(w, 500, "Key provided is not assigned to provided account")
				return
			}
			writeErrorBody(w, 403, "Key provided is not assigned to provided account")
			return
		}
		exists = true
		// if acc does not exists use key instead and make it usable only for account creation
	} else {
		id = r.Form["key"][0]
	}
	// Check if signature is correct
	key, err := ecc.NewPublicKey(keystr)
	if err != nil {
		writeErrorBody(w, 500, "Error creating key")
		return
	}
	// reconstruct signature
	sign, err := ecc.NewSignature(r.Form["sign"][0])
	if err != nil {
		writeErrorBody(w, 500, "Error creating signature")
		return
	}
	hash := getHash([]byte(r.Form["hash"][0]))
	// verify signature
	if !sign.Verify(hash, key) {
		writeErrorBody(w, 403, "Error verifying signature")
		return
	}

	token, validUntil, err := h.token.NewToken(id, exists)
	h.log.Debugf("Token %s created", token)
	if err != nil {
		writeErrorBody(w, 500, "Error while generating token")
		return
	}
	ret := make(map[string]string)
	ret["token"] = token
	ret["validUntil"] = strconv.FormatInt(validUntil.Unix(), 10)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(ret)
	// w.Write([]byte(token))

}

type uploadResponse struct {
	FileID    string `json:"fileID,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func (h *handlers) uploadHandler(w http.ResponseWriter, r *http.Request) {
	response := uploadResponse{}
	// Authorize the user
	token := r.Header.Get("Authorization")
	if !h.validateToken(w, token) {
		return
	}
	if !h.token.IsAccount(token) {
		writeErrorJson(w, 400, "You don't have an account")
		return
	}
	err := r.ParseMultipartForm(0)
	h.log.Debugf("API:: got upload request")
	if err != nil {
		writeErrorJson(w, 500, err.Error())
		return
	}
	params := mux.Vars(r)
	owner := params["account"]
	account := h.token.GetID(token)
	keystr := r.PostForm["key"][0]

	if exists := h.eos.CheckAccountExists(owner); !exists {
		writeErrorBody(w, 404, "404 account not found")
		return
	}
	// check if access is granted
	if account != owner {
		access, err := h.eos.AccessGranted(owner, account)

		if err != nil {
			writeErrorBody(w, 500, err.Error())
			return
		}
		if !access {
			writeErrorJson(w, 403, "Account does not have access to owner")
			return
		}
	}

	// check if account and key match
	auth, err := h.eos.CheckAccountKey(account, keystr)
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}
	if !auth {
		writeErrorJson(w, 403, "Provided key is not associated with account")
		return
	}

	// Check if signature is correct
	key, err := ecc.NewPublicKey(keystr)
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}
	// reconstruct signature
	sign, err := ecc.NewSignature(r.Form["sign"][0])
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}
	// get hash
	file, head, err := r.FormFile("data")
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}
	data := make([]byte, head.Size)
	_, err = file.Read(data)
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}
	hash := getHash(data)

	// verify signature
	if !sign.Verify(hash, key) {
		writeErrorJson(w, 403, "Data could not be verified")
		return
	}

	// Handle file saving
	// create dir
	os.MkdirAll(h.config.StoragePath+owner, os.ModePerm)
	var fid string
	if v, ok := r.Form["reencrypt"]; ok && v[0] == "true" {
		fid = head.Filename
	} else {
		uuid, err := uuid.NewV1()
		if err != nil {
			writeErrorBody(w, 500, err.Error())
			return
		}
		fid = uuid.String()
	}
	// create file
	f, err := os.Create(h.config.StoragePath + owner + "/" + fid)
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}
	defer f.Close()
	// add data to file
	_, err = f.WriteString(string(data))
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}

	//Generate response
	response.FileID = fid
	ts := time.Now().Format("2006-01-02T15:04:05.999Z")
	response.CreatedAt = ts
	h.log.Debugf("API:: File %s uploaded", fid)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(response)
}

func getHash(in []byte) []byte {
	sha := sha256.New()
	sha.Write(in)
	return sha.Sum(nil)
}

type lsResponse struct {
	Files []lsFile `json:"files,omitempty"`
}
type lsFile struct {
	FileID    string `json:"fileID,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func (h *handlers) lsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	account := params["account"]
	response := lsResponse{}
	//Authorize
	token := r.Header.Get("Authorization")
	if !h.validateToken(w, token) {
		return
	}
	if !h.token.IsAccount(token) {
		writeErrorJson(w, 400, "You don't have an account")
		return
	}
	// make sure connected has access to data
	access, err := h.eos.AccessGranted(account, h.token.GetID(token))
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	}
	if !access {
		writeErrorJson(w, 403, "403")
		return
	}

	// Check if requested account exists
	if exists := h.eos.CheckAccountExists(account); !exists {
		writeErrorBody(w, 404, "404 account not found")
		return
	}

	files, err := filepath.Glob(h.config.StoragePath + account + "/*")
	if err != nil {
		writeErrorBody(w, 500, err.Error())
		return
	} else {
		for _, f := range files {
			response.Files = append(response.Files, lsFile{filepath.Base(f), "Will be added soon"})
		}
		if len(response.Files) == 0 {
			writeErrorBody(w, 404, "404 files not found")

			return
		}
	}
	h.log.Debugf("API:: Sending ls")
	json.NewEncoder(w).Encode(response)
}

func (h *handlers) downloadHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	fid := params["fid"]
	account := params["account"]
	//Authorize
	token := r.Header.Get("Authorization")
	if !h.validateToken(w, token) {
		return
	}
	if !h.token.IsAccount(token) {
		writeErrorBody(w, 400, "You don't have an account")
		return
	}
	// make sure connected has access to data
	access, err := h.eos.AccessGranted(account, h.token.GetID(token))
	if err != nil {
		writeErrorBody(w, 500, "Internal server error")
		return
	}
	if !access {
		writeErrorBody(w, 403, "You don't have access to requested data")
		return
	}

	if exists := h.eos.CheckAccountExists(account); !exists {
		writeErrorBody(w, 404, "404 account not found")
		return
	}
	// check if file exists
	if _, err := os.Stat(h.config.StoragePath + account); !os.IsNotExist(err) {
		_, err := os.Stat(h.config.StoragePath + account + "/" + fid)
		if err == nil {
			f, _ := ioutil.ReadFile(h.config.StoragePath + account + "/" + fid)
			w.WriteHeader(200)
			w.Write(f)
		} else {
			writeErrorBody(w, 404, "404 file not found")
		}
	} else {
		writeErrorBody(w, 404, "404 account not found")
	}
}

func (h *handlers) createaccHandler(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]string)
	// Authorize the user
	token := r.Header.Get("Authorization")
	if !h.validateToken(w, token) {
		return
	}
	if exists := h.token.IsAccount(token); exists {
		writeErrorJson(w, 400, "You already have an account")
		return
	}
	key := h.token.GetID(token)
	accountname, err := h.getAccName()
	if err != nil {
		writeErrorJson(w, 500, err.Error())
		return
	}
	h.log.Debugf("API:: Attempting to create account %s", accountname)

	err = h.eos.CreateAccount(accountname, key)
	if err != nil {
		writeErrorJson(w, 400, err.Error())
		return
	}
	response["account"] = accountname
	h.token.AccCreated(token, accountname, key)
	w.WriteHeader(201)

	json.NewEncoder(w).Encode(response)
}

func writeErrorJson(w http.ResponseWriter, statuscode int, err string) {
	w.WriteHeader(statuscode)
	r := make(map[string]string)
	r["error"] = err
	json.NewEncoder(w).Encode(r)
}

func writeErrorBody(w http.ResponseWriter, statuscode int, err string) {
	w.WriteHeader(statuscode)
	w.Write([]byte(err))
}

// Generate random name that satisfies EOS
// regex: "iryo[a-z1-5]{8}"
func (h *handlers) getAccName() (string, error) {
	g, err := reggen.NewGenerator("[a-z1-5]{7}")
	if err != nil {
		return "", err
	}
	var accname string
	for {
		accname = fmt.Sprintf("%s.iryo", g.Generate(7))
		if !h.eos.CheckAccountExists(accname) {
			break
		}
	}
	return accname, nil
}

func (h *handlers) validateToken(w http.ResponseWriter, token string) bool {
	if h.token.IsValid(token) {
		return true
	}
	writeErrorBody(w, 401, "unknown token")
	return false
}
