package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/storage/ehr"
)

type handlers struct {
	config *config.Config
	client *client.RPCClient
	ehr    *ehr.Storage
}

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}

	outErr := r.URL.Query().Get("error")

	user := h.config.GetEthPublicAddress()

	h.client.Download(user)
	ehr, err := h.ehr.Decrypt(user, h.config.EncryptionKeys[user])
	if err != nil {
		outErr = err.Error()
	}

	data := struct {
		Type        string
		Public      string
		Connections []string
		EHRData     string
		Error       string
		Contract    string
	}{
		h.config.ClientType,
		user,
		h.config.Connections,
		string(ehr),
		outErr,
		h.config.EthContractAddr,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (h *handlers) grantAccessHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := h.client.GrantAccess(r.Form["to"][0])

	url := "/"
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}

func (h *handlers) revokeAccessHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := h.client.RevokeAccess(r.Form["to"][0])

	url := "/"
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}

func (h *handlers) saveEHRHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	data := []byte(r.Form["data"][0])
	user := h.config.GetEthPublicAddress()

	err := h.ehr.Encrypt(user, data, h.config.EncryptionKeys[user])
	if err != nil {
		http.Redirect(w, r, "/?error="+err.Error(), 302)
		return
	}

	err = h.client.Upload(user)

	url := "/"
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}
