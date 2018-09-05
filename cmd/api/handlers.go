package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/eos"
	"github.com/iryonetwork/network-poc/storage/token"
	"github.com/iryonetwork/network-poc/storage/ws"
)

type handlers struct {
	config *config.Config
	log    *logger.Log
	f      *storage
}

type storage struct {
	eos    *eos.Storage
	hub    *ws.Hub
	token  *token.TokenList
	config *config.Config
	log    *logger.Log
}

func (h *handlers) loginHandler(w http.ResponseWriter, r *http.Request) {
	h.log.Debugf("Got login request")

	// Get data from form
	r.ParseForm()
	key := r.Form["key"][0]
	id := key

	// If user has an account use it as id
	exists := false
	if name, ok := r.Form["account"]; ok {
		id = name[0]

		// Are key and acc connected?
		code, err := h.f.checkKeyAndID(id, key)
		if err != nil {
			h.writeErrorJson(w, code, err.Error())
			return
		}

		exists = true
	}

	// Verify Signature
	if code, err := h.f.checkSignature(key, r.Form["sign"][0], []byte(r.Form["hash"][0])); err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	// Create new token
	token, validUntil, code, err := h.f.newToken(id, exists)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	ret := make(map[string]string)
	ret["token"] = token
	ret["validUntil"] = strconv.FormatInt(validUntil.Unix(), 10)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(ret)
}

type uploadResponse struct {
	FileID    string `json:"fileID,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

func (h *handlers) uploadHandler(w http.ResponseWriter, r *http.Request) {
	response := uploadResponse{}
	if !isMultipart(r) {
		h.writeErrorJson(w, 400, "Request is not multipart/form-data")
		return
	}

	h.log.Debugf("API:: got upload request")

	// Authorize the user
	token := r.Header.Get("Authorization")
	account, code, err := h.f.tokenValidateGetName(token)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	// Parse the form
	if err = r.ParseMultipartForm(0); err != nil {
		h.log.Debugf("Error parsing multipart form; %v", err)
		h.writeErrorJson(w, 500, err.Error())
		return
	}

	params := mux.Vars(r)
	owner := params["account"]
	key := r.PostForm["key"][0]

	file, header, err := r.FormFile("data")
	if err != nil {
		h.writeErrorJson(w, 500, err.Error())
		return
	}

	reupload := isReupload(r)
	// Save the file
	fid, ts, code, err := h.f.saveFileWithChecks(owner, account, key, r.FormValue("sign"), file, header, reupload)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	//Generate response
	response.FileID = fid
	response.CreatedAt = ts
	h.log.Debugf("API:: File %s uploaded", fid)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(response)
}

func (h *handlers) lsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	owner := params["account"]

	//Authorize
	token := r.Header.Get("Authorization")
	account, code, err := h.f.tokenValidateGetName(token)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	if code, err = h.f.checkAccessGranted(owner, account); err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}
	response, code, err := h.f.listFiles(owner)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}
	h.log.Debugf("API:: Sending ls")
	json.NewEncoder(w).Encode(response)
}

func (h *handlers) downloadHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	fid := params["fid"]
	owner := params["account"]
	//Authorize
	token := r.Header.Get("Authorization")
	account, code, err := h.f.tokenValidateGetName(token)
	// make sure connected has access to data
	if code, err := h.f.checkAccessGranted(owner, account); err != nil {
		h.writeErrorBody(w, code, err.Error())
		return
	}
	f, code, err := h.f.readFileData(owner, fid)
	if err != nil {
		h.writeErrorBody(w, code, err.Error())
		return
	}
	w.WriteHeader(200)
	w.Write(f)

}

func (h *handlers) createaccHandler(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]string)
	// Authorize the user
	token := r.Header.Get("Authorization")
	h.log.Debugf("Checking token")
	key, code, err := h.f.tokenAccountCreation(token)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}
	h.log.Debugf("Creating new account")
	accountname, code, err := h.f.newAccount(key)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	response["account"] = accountname
	h.f.token.AccCreated(token, accountname, key)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(response)
}

func (h *handlers) writeErrorJson(w http.ResponseWriter, statuscode int, err string) {
	h.log.Debugf("API handlers ERR = %s", err)
	w.WriteHeader(statuscode)
	r := make(map[string]string)
	r["error"] = err
	json.NewEncoder(w).Encode(r)
}

func (h *handlers) writeErrorBody(w http.ResponseWriter, statuscode int, err string) {
	h.log.Debugf("API handlers ERR = %s", err)
	w.WriteHeader(statuscode)
	w.Write([]byte(err))
}

// Generate random name that satisfies EOS
// regex: "iryo[a-z1-5]{8}"

func retry(f func() error, wait time.Duration, attempts int) (err error) {
	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}

		time.Sleep(wait)

		log.Println("retrying after error:", err)
	}

	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
