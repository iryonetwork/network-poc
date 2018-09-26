package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/db"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/storage/eos"
	"github.com/iryonetwork/network-poc/storage/token"
	"github.com/iryonetwork/network-poc/storage/ws/hub"
)

type handlers struct {
	eos    *eos.Storage
	hub    *hub.Hub
	token  *token.TokenList
	config *config.Config
	log    *logger.Log
	db     *db.Db
}

type storage struct {
	*handlers
}

func (h *handlers) loginHandler(w http.ResponseWriter, r *http.Request) {
	funcs := storage{h}
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
		code, err := funcs.checkKeyAndID(id, key)
		if err != nil {
			h.writeErrorJson(w, code, err.Error())
			return
		}

		exists = true
	}

	// Verify Signature
	if code, err := funcs.checkSignature(key, r.Form["sign"][0], []byte(r.Form["hash"][0])); err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	// Create new token
	token, validUntil, code, err := funcs.newToken(id, exists)
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

func (h *handlers) uploadHandler(w http.ResponseWriter, r *http.Request, fid string) {
	funcs := storage{h}

	response := uploadResponse{}
	if !isMultipart(r) {
		h.writeErrorJson(w, 400, "Request is not multipart/form-data")
		return
	}

	h.log.Debugf("API:: got upload request")

	// Authorize the user
	token := r.Header.Get("Authorization")
	account, code, err := funcs.tokenValidateGetName(token)
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

	// Save the file
	fid, ts, code, err := funcs.saveFileWithChecks(owner, account, key, r.FormValue("sign"), file, header, fid)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	funcs.notifyConnectedUpload(owner, account, fid)

	//Generate response
	response.FileID = fid
	response.CreatedAt = ts
	h.log.Debugf("API:: File %s uploaded", fid)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(response)
}

func (h *handlers) lsHandler(w http.ResponseWriter, r *http.Request) {
	funcs := storage{h}

	params := mux.Vars(r)
	owner := params["account"]

	//Authorize
	token := r.Header.Get("Authorization")
	account, code, err := funcs.tokenValidateGetName(token)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	if code, err = funcs.checkAccessGranted(owner, account); err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}
	response, code, err := funcs.listFiles(owner)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}
	h.log.Debugf("API:: Sending ls")
	json.NewEncoder(w).Encode(response)
}

func (h *handlers) downloadHandler(w http.ResponseWriter, r *http.Request) {
	funcs := storage{h}

	params := mux.Vars(r)

	fid := params["fid"]
	owner := params["account"]
	//Authorize
	token := r.Header.Get("Authorization")
	account, code, err := funcs.tokenValidateGetName(token)
	// make sure connected has access to data
	if code, err := funcs.checkAccessGranted(owner, account); err != nil {
		h.writeErrorBody(w, code, err.Error())
		return
	}
	f, code, err := funcs.readFileData(owner, fid)
	if err != nil {
		h.writeErrorBody(w, code, err.Error())
		return
	}
	w.WriteHeader(200)
	w.Write(f)

}

func (h *handlers) createaccHandler(w http.ResponseWriter, r *http.Request) {
	funcs := storage{h}

	response := make(map[string]string)
	r.ParseForm()
	// Authorize the user
	token := r.Header.Get("Authorization")
	h.log.Debugf("Checking token")
	key, code, err := funcs.tokenAccountCreation(token)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}
	h.log.Debugf("Creating new account")
	accountname, code, err := funcs.newAccount(key)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	h.db.AddName(accountname, r.Form["name"][0])
	response["account"] = accountname
	h.token.AccCreated(token, accountname, key)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(response)
}

func (h *handlers) accountToIDHandler(w http.ResponseWriter, r *http.Request) {
	funcs := storage{h}

	owner := mux.Vars(r)["account"]
	token := r.Header.Get("Authorization")

	account, code, err := funcs.tokenValidateGetName(token)
	if err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	if code, err = funcs.checkAccessGranted(owner, account); err != nil {
		h.writeErrorJson(w, code, err.Error())
		return
	}

	response := make(map[string]string)

	if response["name"], err = h.db.GetName(owner); err != nil {
		h.writeErrorJson(w, 500, err.Error())
		return
	}

	w.WriteHeader(200)
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
