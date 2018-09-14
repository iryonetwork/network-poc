package main

import (
	"encoding/base64"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/openEHR/ehrdata"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/storage/ehr"
	qrcode "github.com/skip2/go-qrcode"
)

type handlers struct {
	config *config.Config
	client *client.Client
	ehr    *ehr.Storage
	log    *logger.Log
}

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}
	qr, err := qrcode.New(h.client.NewRequestKeyQr(), qrcode.Highest)
	if err != nil {
		log.Fatalf("Error creating qr: %v", err)
	}
	img, err := qr.PNG(150)
	if err != nil {
		log.Fatalf("Error creating qr png: %v", err)
	}

	data := struct {
		Type        string
		Name        string
		Public      string
		Private     string
		Connections map[string]string
		Contract    string
		Connected   bool
		Granted     map[string]string
		Qr          string
	}{
		h.config.ClientType,
		h.config.EosAccount,
		h.config.GetEosPublicKey(),
		h.config.EosPrivate,
		h.config.GetNames(h.config.Connections),
		h.config.EosContractName,
		h.config.Connceted,
		h.config.GetNames(h.config.GrantedWithoutKeys),
		base64.StdEncoding.EncodeToString(img),
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

	ehr := make(map[string]string)
	// Download missing/new files/ check if access was removed
	err = h.client.Update(owner)
	if err != nil {

	} else {
		for k, v := range h.ehr.Get(owner) {
			ehr[k] = string(h.ehr.Getid(owner, k))
			v, err = h.ehr.Decrypt(owner, k, h.config.EncryptionKeys[owner])
			if err != nil {
				break
			}
			ehr[k+"_dec"] = string(v)
		}
	}
	if err != nil {
		outErr = err.Error()
	}
	if outErr == "Code: 404" {
		outErr = ""
	}

	data := struct {
		Type          string
		Public        string
		Contract      string
		Owner         string
		OwnerUsername string
		EHRData       map[string]string
		Error         string
	}{
		h.config.ClientType,
		h.config.EosAccount,
		h.config.EosContractName,
		h.config.Directory[owner],
		owner,
		ehr,
		outErr,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (h *handlers) closeHandler(w http.ResponseWriter, r *http.Request) {
	if h.config.Connceted {
		h.client.CloseWs()
	}
	http.Redirect(w, r, "/", 302)
}

func (h *handlers) connectHandler(w http.ResponseWriter, r *http.Request) {
	if !h.config.Connceted {
		err := h.client.ConnectWs()
		if err != nil {
			h.log.Printf("Failed to connect to ws:%v", err)
		}
	}
	http.Redirect(w, r, "/", 302)
}

func (h *handlers) requestHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	to := r.Form["to"][0]

	err := h.client.RequestAccess(to)
	if err != nil {
		h.log.Fatalf("Error requesting access: %v ", err)
	}

	http.Redirect(w, r, "/", 302)
}

func (h *handlers) ignoreHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	to := r.Form["to"][0]
	for i, v := range h.config.GrantedWithoutKeys {
		if v == to {
			h.config.GrantedWithoutKeys = append(h.config.GrantedWithoutKeys[:i], h.config.GrantedWithoutKeys[i+1:]...)
		}
	}
	http.Redirect(w, r, "/", 302)
}

func (h *handlers) saveEHRHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	owner := r.Form["owner"][0]
	weight := r.Form["weight"][0]
	glucose := r.Form["glucose"][0]
	systolic := r.Form["systolic"][0]
	diastolic := r.Form["diastolic"][0]

	var err error
	data := ehrdata.NewVitalSigns(h.config)
	if err = ehrdata.AddVitalSigns(data, weight, glucose, systolic, diastolic); err == nil {
		err = ehrdata.SaveAndUpload(owner, h.config, h.ehr, h.client, data)
	}

	url := "/ehr/" + owner
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}
