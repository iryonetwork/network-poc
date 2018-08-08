package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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

type uploadResponse struct {
	Err       []string `json:"error,omitempty"`
	FileID    string   `json:"fileID,omitempty"`
	EhrID     string   `json:"ehrID,omitempty"`
	CreatedAt string   `json:"createdAt,omitempty"`
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
				w.WriteHeader(500)
				w.Write([]byte("Key provided is not assigned to provided account"))
				return
			}
			w.WriteHeader(403)
			w.Write([]byte("Key provided is not assigned to provided account"))
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
		w.WriteHeader(500)
		w.Write([]byte("Error creating key"))
		return
	}
	// reconstruct signature
	sign, err := ecc.NewSignature(r.Form["sign"][0])
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error creating signature"))
		return
	}

	// verify signature
	if !sign.Verify([]byte(r.Form["hash"][0]), key) {
		w.WriteHeader(403)
		w.Write([]byte("Error creating signature"))
		return
	}

	token, err := h.token.NewToken(id, exists)
	h.log.Debugf("Token %s created", token)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error while generating token"))
		return
	}
	w.WriteHeader(201)
	w.Write([]byte(token))

}
func (h *handlers) uploadHandler(w http.ResponseWriter, r *http.Request) {
	response := uploadResponse{}
	// Authorize the user
	token := r.Header.Get("Authorization")
	if !h.validateToken(w, token) {
		return
	}
	if !h.token.IsAccount(token) {
		response.Err = append(response.Err, "You don't have an account")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(response)
		return
	}
	err := r.ParseMultipartForm(0)
	h.log.Debugf("API:: got upload request")
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	params := mux.Vars(r)
	owner := params["account"]
	account := h.token.GetAccount(token)
	keystr := r.PostForm["key"][0]

	if exists := h.eos.CheckAccountExists(owner); !exists {
		w.WriteHeader(404)
		w.Write([]byte("404 account not found"))
		return
	}
	// check if access is granted
	if account != owner {
		access, err := h.eos.AccessGranted(owner, account)

		if err != nil {
			response.Err = append(response.Err, err.Error())
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(response)
			return
		}
		if !access {
			response.Err = append(response.Err, "Account does not have access to owner")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// check if account and key match
	auth, err := h.eos.CheckAccountKey(account, keystr)
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	if !auth {
		response.Err = append(response.Err, "Provided key is not associated with account")
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if signature is correct
	key, err := ecc.NewPublicKey(keystr)
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	// reconstruct signature
	sign, err := ecc.NewSignature(r.Form["sign"][0])
	if err != nil {
		response.Err = append(response.Err, "Error getting signature")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	// get hash
	file, head, err := r.FormFile("data")
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	data := make([]byte, head.Size)
	_, err = file.Read(data)
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	hash := getHash(data)

	// verify signature
	if !sign.Verify(hash, key) {
		response.Err = append(response.Err, "Data could not be verified")
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle file saving
	// create dir
	os.MkdirAll("ehr/"+owner, os.ModePerm)
	fid, err := uuid.NewV1()
	// create file
	f, err := os.Create("ehr/" + owner + "/" + fid.String())
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer f.Close()
	// add data to file
	_, err = f.WriteString(string(data))
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}

	//Generate response
	response.FileID = fid.String()
	ts := time.Now().Format("2006-01-02T15:04:05.999Z")
	response.CreatedAt = ts
	w.WriteHeader(201)
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
	CreatedAt string `json:"createdAt,omitempty"`
}

func (h *handlers) lsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	account := params["account"]
	h.log.Debugf("API:: got ls(%v) request", account)
	response := lsResponse{}
	//Authorize
	token := r.Header.Get("Authorization")
	if !h.validateToken(w, token) {
		return
	}
	if !h.token.IsAccount(token) {
		response.Err = append(response.Err, "You don't have an account")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(response)
		return
	}
	// make sure connected has access to data
	access, err := h.eos.AccessGranted(account, h.token.GetAccount(token))
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	if !access {
		response.Err = append(response.Err, "403")
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if requested account exists
	if exists := h.eos.CheckAccountExists(account); !exists {
		w.WriteHeader(404)
		w.Write([]byte("404 account not found"))
		return
	}

	files, err := filepath.Glob("./ehr/" + account + "/*")
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
	} else {
		for _, f := range files {
			response.Files = append(response.Files, lsFile{filepath.Base(f), "Will be added soon"})
		}
		if len(response.Files) == 0 {
			w.WriteHeader(404)
			w.Write([]byte("404 files not found"))
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
		w.WriteHeader(400)
		w.Write([]byte("You don't have an account"))
		return
	}
	// make sure connected has access to data
	access, err := h.eos.AccessGranted(account, h.token.GetAccount(token))
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Internal server error"))
		return
	}
	if !access {
		w.WriteHeader(403)
		w.Write([]byte("You don't have access to requested data"))
		return
	}

	if exists := h.eos.CheckAccountExists(account); !exists {
		w.WriteHeader(404)
		w.Write([]byte("404 account not found"))
		return
	}
	// check if file exists
	if _, err := os.Stat("ehr/" + account); !os.IsNotExist(err) {
		_, err := os.Stat("ehr/" + account + "/" + fid)
		if err == nil {
			f, _ := ioutil.ReadFile("ehr/" + account + "/" + fid)
			w.WriteHeader(200)
			w.Write(f)
		} else {
			w.WriteHeader(404)
			w.Write([]byte("404 file not found"))
		}
	} else {
		w.WriteHeader(404)
		w.Write([]byte("404 account not found"))
	}
}

type createAccResponse struct {
	Err  []string `json:"error,omitempty"`
	Name string   `json:"account,omitempty"`
}

func (h *handlers) createaccHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	key := params["key"]
	response := createAccResponse{}
	// Authorize the user
	token := r.Header.Get("Authorization")
	if !h.validateToken(w, token) {
		return
	}
	if exists := h.token.IsAccount(token); exists {
		response.Err = append(response.Err, "You already have an account")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(response)
		return
	}
	accountname, err := h.getAccName()
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(response)
		return
	}
	h.log.Debugf("API:: Attempting to create account %s", accountname)

	err = h.eos.CreateAccount(accountname, key)
	if err != nil {
		response.Err = append(response.Err, err.Error())
		w.WriteHeader(500)
	} else {
		response.Name = accountname
		h.token.AccCreated(token, accountname, key)
		w.WriteHeader(201)
	}
	json.NewEncoder(w).Encode(response)
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
	w.WriteHeader(401)
	w.Write([]byte("unknown token"))
	return false
}
