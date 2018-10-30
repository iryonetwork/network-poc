package main

import (
	"crypto/rand"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/iryonetwork/network-poc/client"
	"github.com/iryonetwork/network-poc/logger"
	"github.com/iryonetwork/network-poc/openEHR/ehrdata"

	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/storage/ehr"
)

type handlers struct {
	config *config.Config
	client *client.Client
	ehr    *ehr.Storage
	log    *logger.Log
}

func (h *handlers) ehrHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/ehr.html")
	if err != nil {
		log.Fatalf("error parsing template files: %v", err)
	}

	parts := strings.Split(r.URL.EscapedPath(), "/")
	owner := parts[2]
	outErr := ""

	var graphData *map[string]ehrdata.Entry
	var graphDataJSON []byte
	ehr := make(map[string]string)
	// Download missing/new files/ check if access was removed
	err = h.client.Update(owner)
	if err != nil {
		outErr = err.Error()
		goto show
	} else {
		for k := range h.ehr.Get(owner) {
			ehr[k] = string(h.ehr.Getid(owner, k))
			var v []byte
			v, err = h.ehr.Decrypt(owner, k, h.config.EncryptionKeys[owner])
			if err != nil {
				break
			}
			ehr[k+"_dec"] = string(v)
		}
	}
	if err != nil {
		outErr = err.Error()
		goto show
	}

	graphData, err = ehrdata.ExtractEhrData(owner, h.ehr, h.config)
	if err != nil {
		outErr = err.Error()
		goto show
	}

	graphDataJSON, err = json.Marshal(graphData)
	if err != nil {
		outErr = err.Error()
	}

show:
	data := struct {
		Name          string
		Public        string
		Contract      string
		Owner         string
		OwnerUsername string
		EHRData       map[string]string
		GraphData     string
		Error         string
	}{
		h.config.PersonalData.Name,
		h.config.EosAccount,
		h.config.EosContractName,
		h.config.Directory[owner],
		owner,
		ehr,
		string(graphDataJSON),
		outErr,
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

func (h *handlers) closeHandler(w http.ResponseWriter, r *http.Request) {
	if h.config.Connected {
		h.client.CloseWs()
	}
	http.Redirect(w, r, "/", 302)
}

func (h *handlers) connectHandler(w http.ResponseWriter, r *http.Request) {
	if !h.config.Connected {
		err := h.client.ConnectWs()
		if err != nil {
			h.log.Printf("Failed to connect to ws:%v", err)
		}
	}
	http.Redirect(w, r, "/", 302)
}

func (h *handlers) saveEHRHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	owner := h.config.EosAccount
	if owners, ok := r.Form["owner"]; ok {
		owner = owners[0]
	}

	weight := r.Form["weight"][0]
	glucose := r.Form["glucose"][0]
	systolic := r.Form["systolic"][0]
	diastolic := r.Form["diastolic"][0]

	var err error
	data := ehrdata.NewVitalSigns(h.config)
	if err = ehrdata.AddVitalSigns(data, weight, glucose, systolic, diastolic); err == nil {
		err = h.client.SaveAndUploadEhrData(owner, data)
	}
	url := "/"
	if h.config.ClientType == "Doctor" {
		url = "/ehr/" + owner
	}
	if err != nil {
		url += "?error=" + err.Error()
	}

	http.Redirect(w, r, url, 302)
}

func (h *handlers) requestHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	to := r.Form["to"][0]

	err := h.client.RequestAccess(to, "test")
	if err != nil {
		h.log.Fatalf("Error requesting access: %v ", err)
	}

	http.Redirect(w, r, "/", 302)
}

func (h *handlers) ignoreHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	to := r.Form["to"][0]
	for i, v := range h.config.Connections.WithoutKey {
		if v == to {
			h.config.Connections.WithoutKey = append(h.config.Connections.WithoutKey[:i], h.config.Connections.WithoutKey[i+1:]...)
		}
	}
	http.Redirect(w, r, "/", 302)
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
	delete(h.config.Connections.Requested, to)
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

func (h *handlers) switchModeHandler(w http.ResponseWriter, r *http.Request) {
	h.config.IsDoctor = !h.config.IsDoctor
	http.Redirect(w, r, "/", 302)
}

func (h *handlers) configHandler(w http.ResponseWriter, r *http.Request) {
	a, err := json.Marshal(h.config)
	if err != nil {
		h.log.Debugf("error marshaling config: %v", err)
	}
	w.Write(a)
}
