package main

import (
	"crypto/rand"
	"crypto/rsa"
	"html/template"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"

	"github.com/iryonetwork/network-poc/storage/ehr"
)

type handlers struct {
	config *config.Config
	client *client.Client
	ehr    *ehr.Storage
}

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}
	outErr := r.URL.Query().Get("error")

	user := h.config.EosAccount
	err = h.client.Update(user)
	ehr := make(map[string]string)
	for k := range h.ehr.Get(user) {
		ehr[k] = string(h.ehr.Getid(user, k))
		v, err := h.ehr.Decrypt(user, k, h.config.EncryptionKeys[user])
		if err != nil {
			break
		}
		ehr[k+"_dec"] = string(v)
	}

	if err != nil {
		outErr = err.Error()
	}
	// If there are no ehr entries yet 404 error is retuned
	if outErr == "Code: 404" {
		outErr = ""
	}

	data := struct {
		Type        string
		Public      string
		Connections []string
		EHRData     map[string]string
		Error       string
		Contract    string
		Requested   map[string]*rsa.PublicKey
	}{
		h.config.ClientType,
		user,
		h.config.Connections,
		ehr,
		outErr,
		h.config.EosContractName,
		h.config.Requested,
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
func (h *handlers) denyAccessHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	to := r.Form["to"][0]
	delete(h.config.Requested, to)
	http.Redirect(w, r, "/", 302)
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
	user := h.config.EosAccount

	id, err := h.ehr.Encrypt(user, data, h.config.EncryptionKeys[user])
	if err != nil {
		http.Redirect(w, r, "/?error="+err.Error(), 302)
		return
	}
	err = h.client.Upload(user, id, false)

	url := "/"
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}

func (h *handlers) reencryptHandler(w http.ResponseWriter, r *http.Request) {
	// We need new key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		http.Redirect(w, r, "/?error="+err.Error(), 302)
		return
	}

	err = h.client.Reencrypt(key)
	if err != nil {
		http.Redirect(w, r, "/?error="+err.Error(), 302)
		return
	}

	http.Redirect(w, r, "/", 302)
}
