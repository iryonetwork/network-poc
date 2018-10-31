package main

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/iryonetwork/network-poc/state"

	"github.com/iryonetwork/network-poc/openEHR/ehrdata"
	qrcode "github.com/skip2/go-qrcode"
)

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}
	outErr := r.URL.Query().Get("error")

	user := h.state.EosAccount
	err = h.client.Update(user)
	if err != nil {
		outErr = err.Error()
	}
	ehr, err := ehrdata.ExtractEhrData(h.state.EosAccount, h.ehr, h.state)
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
		h.state.PersonalData.Name,
		user,
		h.state.GetEosPublicKey(),
		h.state.EosPrivate,
		h.state.GetNames(h.state.Connections.GrantedTo),
		string(jsonEhr),
		outErr,
		h.state.GetNames(mapKeysToArray(h.state.Connections.Requested)),
		base64.StdEncoding.EncodeToString(img),
		h.state.Connected,
		h.state.GetNames(h.state.Connections.WithoutKey),
		h.state.GetNames(h.state.Connections.WithKey),
		h.state.IsDoctor,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func mapKeysToArray(m map[string]state.Request) []string {
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
		h.state.PersonalData.Name,
		h.state.EosAccount,
		h.state.GetEosPublicKey(),
		h.state.EosPrivate,
		h.state.GetNames(h.state.Connections.WithKey),
		h.config.EosContractName,
		h.state.Connected,
		h.state.GetNames(h.state.Connections.WithoutKey),
		base64.StdEncoding.EncodeToString(img),
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}
