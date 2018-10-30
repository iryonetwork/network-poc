package main

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/config"

	"github.com/iryonetwork/network-poc/openEHR/ehrdata"
	qrcode "github.com/skip2/go-qrcode"
)

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}
	outErr := r.URL.Query().Get("error")

	user := h.config.EosAccount
	err = h.client.Update(user)
	if err != nil {
		outErr = err.Error()
	}
	ehr, err := ehrdata.ExtractEhrData(h.config.EosAccount, h.ehr, h.config)
	if err != nil {
		outErr = err.Error()
	}
	jsonEhr, err := json.Marshal(ehr)
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
		Username    string
		Public      string
		Private     string
		GrantedTo   map[string]string
		EHRData     string
		Error       string
		Requested   map[string]string
		Qr          string
		Connected   bool
		Granted     map[string]string
		GrantedFrom map[string]string
		IsDoctor    bool
	}{
		h.config.ClientType,
		h.config.PersonalData.Name,
		user,
		h.config.GetEosPublicKey(),
		h.config.EosPrivate,
		h.config.GetNames(h.config.Connections.GrantedTo),
		string(jsonEhr),
		outErr,
		h.config.GetNames(mapKeysToArray(h.config.Connections.Requested)),
		base64.StdEncoding.EncodeToString(img),
		h.config.Connected,
		h.config.GetNames(h.config.Connections.WithoutKey),
		h.config.GetNames(h.config.Connections.WithKey),
		h.config.IsDoctor,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func mapKeysToArray(m map[string]config.Request) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (h *handlers) doctorIndexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/doctor/index.html")
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
		Username    string
		Public      string
		Private     string
		Connections map[string]string
		Contract    string
		Connected   bool
		Granted     map[string]string
		Qr          string
	}{
		h.config.ClientType,
		h.config.PersonalData.Name,
		h.config.EosAccount,
		h.config.GetEosPublicKey(),
		h.config.EosPrivate,
		h.config.GetNames(h.config.Connections.WithKey),
		h.config.EosContractName,
		h.config.Connected,
		h.config.GetNames(h.config.Connections.WithoutKey),
		base64.StdEncoding.EncodeToString(img),
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}
