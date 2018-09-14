package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/openEHR/ehrdata"
	qrcode "github.com/skip2/go-qrcode"

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
	ehr, err := ehrdata.ExtractEhrData(h.config.EosAccount, h.ehr, h.config)
	if err != nil {
		outErr = err.Error()
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
		EHRData     *map[string]ehrdata.Entry
		Error       string
		Contract    string
		Requested   map[string]string
		Qr          string
		Connected   bool
	}{
		h.config.ClientType,
		user,
		h.config.GetEosPublicKey(),
		h.config.EosPrivate,
		h.config.GetNames(h.config.Connections),
		ehr,
		outErr,
		h.config.EosContractName,
		h.config.GetNames(mapKeysToArray(h.config.Requested)),
		base64.StdEncoding.EncodeToString(img),
		h.config.Connceted,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func mapKeysToArray(m map[string]*rsa.PublicKey) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	return out
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

	weight := r.Form["weight"][0]
	glucose := r.Form["glucose"][0]
	systolic := r.Form["systolic"][0]
	diastolic := r.Form["diastolic"][0]

	var err error
	data := ehrdata.NewVitalSigns(h.config)
	if err := ehrdata.AddVitalSigns(data, weight, glucose, systolic, diastolic); err == nil {
		err = ehrdata.SaveAndUpload(h.config.EosAccount, h.config, h.ehr, h.client, data)
	}

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
			log.Printf("Failed to connect to ws:%v", err)
		}
	}
	http.Redirect(w, r, "/", 302)
}
