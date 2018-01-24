package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"

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

	data := struct {
		Type        string
		Public      string
		Connections []string
	}{
		h.config.ClientType,
		h.config.GetEthPublicAddress(),
		h.config.Connections,
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

	var ehr []byte
	err = h.client.Download(owner)

	if err == nil {
		ehr, err = h.ehr.Decrypt(owner, h.config.EncryptionKeys[owner])
	}

	if err != nil {
		outErr = err.Error()
	}

	data := struct {
		Type    string
		Public  string
		Owner   string
		EHRData string
		Error   string
	}{
		h.config.ClientType,
		h.config.GetEthPublicAddress(),
		owner,
		string(ehr),
		outErr,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (h *handlers) saveEHRHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	owner := r.Form["owner"][0]
	data := []byte(r.Form["data"][0])

	err := h.ehr.Encrypt(owner, data, h.config.EncryptionKeys[owner])
	if err != nil {
		http.Redirect(w, r, "/ehr/"+owner+"?error="+err.Error(), 302)
		return
	}

	err = h.client.Upload(owner)

	url := "/ehr/" + owner
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}
