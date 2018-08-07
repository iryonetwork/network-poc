package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/iryonetwork/network-poc/logger"

	"github.com/iryonetwork/network-poc/storage/ws"

	"github.com/iryonetwork/network-poc/config"
	client "github.com/iryonetwork/network-poc/eosclient"
	"github.com/iryonetwork/network-poc/storage/ehr"
)

type handlers struct {
	config    *config.Config
	client    *client.Client
	ehr       *ehr.Storage
	connected bool
	log       *logger.Log
}

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}

	data := struct {
		Type        string
		Public      string
		Connections []string
		Contract    string
		Connected   bool
	}{
		h.config.ClientType,
		h.config.EosAccount,
		h.config.Connections,
		h.config.EosContractName,
		h.connected,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (h *handlers) ehrHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/ehr.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}

	parts := strings.Split(r.URL.EscapedPath(), "/")
	owner := parts[2]
	outErr := ""

	// Download missing/new files/ check if access was removed
	err = h.client.Update(owner)

	ehr := make(map[string]string)
	for k, v := range h.ehr.Get(owner) {
		ehr[k] = string(h.ehr.Getid(owner, k))
		v, err = h.ehr.Decrypt(owner, k, h.config.EncryptionKeys[owner])
		if err != nil {
			break
		}
		ehr[k+"_dec"] = string(v)
	}

	if err != nil {
		outErr = err.Error()
	}

	data := struct {
		Type     string
		Public   string
		Contract string
		Owner    string
		EHRData  map[string]string
		Error    string
	}{
		h.config.ClientType,
		h.config.EosAccount,
		h.config.EosContractName,
		owner,
		ehr,
		outErr,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (h *handlers) closeHandler(w http.ResponseWriter, r *http.Request) {
	if h.connected {
		h.client.CloseWs()
		h.connected = false
	}
	http.Redirect(w, r, "/", 302)
}

func (h *handlers) connectHandler(w http.ResponseWriter, r *http.Request) {
	if !h.connected {
		ws, err := ws.Connect(h.config, h.log)
		if err != nil {
			log.Printf("error closing connection: %v", err)
		}
		h.client.AddWs(ws)
		h.client.Subscribe()

		h.connected = true
	}
	http.Redirect(w, r, "/", 302)
}

// func (h *handlers) saveEHRHandler(w http.ResponseWriter, r *http.Request) {
// 	r.ParseForm()

// 	owner := r.Form["owner"][0]
// 	data := []byte(r.Form["data"][0])

// 	err := h.ehr.Encrypt(owner, data, h.config.EncryptionKeys[owner])
// 	if err != nil {
// 		http.Redirect(w, r, "/ehr/"+owner+"?error="+err.Error(), 302)
// 		return
// 	}

// 	err = h.client.Upload(owner)

// 	url := "/ehr/" + owner
// 	if err != nil {
// 		url += "?error=" + err.Error()
// 	}

// 	http.Redirect(w, r, url, 302)
// }
