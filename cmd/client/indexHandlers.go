package main

import (
	"crypto/rsa"
	"encoding/base64"
	"html/template"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/openEHR/ehrdata"
	qrcode "github.com/skip2/go-qrcode"
)

func (h *handlers) patientIndexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/patient/index.html")
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
