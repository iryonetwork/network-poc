package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/storage/ehr"
	"github.com/iryonetwork/network-poc/storage/eos"
)

// TODO: Implement eos in client
type handlers struct {
	config *config.Config
	client *client.RPCClient
	eos    *eos.Storage
	ehr    *ehr.Storage
}

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}

	outErr := r.URL.Query().Get("error")

	user := h.config.EosAccount
	ehr := h.ehr.Get(user)
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
		h.config.EosContractName,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (h *handlers) grantAccessHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := h.eos.GrantAccess(r.Form["to"][0])

	url := "/"
	if err != nil {
		url += "?error=" + err.Error()
	} else {
		h.config.Connections = append(h.config.Connections, r.Form["to"][0])
	}
	http.Redirect(w, r, url, 302)
}

func (h *handlers) revokeAccessHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := h.eos.RevokeAccess(r.Form["to"][0])

	url := "/"
	if err != nil {
		url += "?error=" + err.Error()
	} else {
		//TODO: Would having a map[doctor]doctor be better performace wise / would it take too much space?
		for n, v := range h.config.Connections {
			if v == r.Form["to"][0] {
				h.config.Connections = append(h.config.Connections[:n], h.config.Connections[n+1:]...)
			}
		}
	}
	http.Redirect(w, r, url, 302)
}

func (h *handlers) saveEHRHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	data := []byte(r.Form["data"][0])
	user := h.config.GetEosPublicKey()

	err := h.ehr.Encrypt(user, data, h.config.EncryptionKeys[user])
	if err != nil {
		http.Redirect(w, r, "/?error="+err.Error(), 302)
		return
	}

	url := "/"
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}
