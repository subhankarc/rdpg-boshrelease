package cfsb

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/starkandwayne/rdpgd/instances"
	"github.com/starkandwayne/rdpgd/log"
)

var (
	sbPort, sbUser, sbPass string
	pgPort, pbPort, pgPass string
)

type CFSB struct {
}

func init() {
	sbPort = os.Getenv("RDPGD_SB_PORT")
	if sbPort == "" {
		sbPort = "8888"
	}
	sbUser = os.Getenv("RDPGD_SB_USER")
	if sbUser == "" {
		sbUser = "cfadmin"
	}
	sbPass = os.Getenv("RDPGD_SB_PASS")
	if sbPass == "" {
		sbPass = "cfadmin"
	}
	pbPort = os.Getenv(`RDPGD_PB_PORT`)
	if pbPort == `` {
		pbPort = `6432`
	}
	pgPass = os.Getenv(`RDPGD_PG_PASS`)
}

func API() (err error) {
	CFSBMux := http.NewServeMux()
	router := mux.NewRouter()
	router.HandleFunc("/v2/catalog", httpAuth(CatalogHandler))
	router.HandleFunc("/v2/service_instances/{instance_id}", httpAuth(InstanceHandler))
	CFSBMux.Handle("/", router)
	router.HandleFunc("/v2/service_instances/{instance_id}/service_bindings/{binding_id}", httpAuth(BindingHandler))

	http.Handle("/", router)
	err = http.ListenAndServe(":"+sbPort, CFSBMux)
	log.Error(fmt.Sprintf(`cfsbapi.API() ! %s`, err))
	return err
}

func httpAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if len(request.Header["Authorization"]) == 0 {
			log.Trace(fmt.Sprintf("httpAuth(): Authorization Required"))
			http.Error(w, "Authorization Required", http.StatusUnauthorized)
			return
		}

		auth := strings.SplitN(request.Header["Authorization"][0], " ", 2)
		if len(auth) != 2 || auth[0] != "Basic" {
			log.Error(fmt.Sprintf("httpAuth(): Unhandled Authorization Type, Expected Basic"))
			http.Error(w, "Unhandled Authroization Type, Expected Basic\n", http.StatusBadRequest)
			return
		}
		payload, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			log.Error(fmt.Sprintf("httpAuth(): Authorization Failed"))
			http.Error(w, "Authorization Failed\n", http.StatusUnauthorized)
			return
		}
		nv := strings.SplitN(string(payload), ":", 2)
		if (len(nv) != 2) || !isAuthorized(nv[0], nv[1]) {
			log.Error(fmt.Sprintf("httpAuth(): Authorization Failed"))
			http.Error(w, "Authorization Failed\n", http.StatusUnauthorized)
			return
		}
		h(w, request)
	}
}

func isAuthorized(username, password string) bool {
	if username == sbUser && password == sbPass {
		return true
	}
	return false
}

/*
(FC) GET /v2/catalog
*/
func CatalogHandler(w http.ResponseWriter, request *http.Request) {
	log.Trace(fmt.Sprintf("%s /v2/catalog", request.Method))
	switch request.Method {
	case "GET":
		c := Catalog{}
		err := c.Fetch()
		if err != nil {
			msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
			log.Error(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		jsonCatalog, err := json.Marshal(c)
		if err != nil {
			msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
			log.Error(msg)
			http.Error(w, msg, http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonCatalog)
		}
	default:
		msg := fmt.Sprintf(`{"status": %d, "description": "Allowed Methods: GET"}`, http.StatusMethodNotAllowed)
		log.Error(msg)
		http.Error(w, msg, http.StatusMethodNotAllowed)
	}
	return
}

/*
(PI) PUT /v2/service_instances/:id
(RI) DELETE /v2/service_instances/:id
*/
func InstanceHandler(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	log.Trace(fmt.Sprintf("%s /v2/service_instances/:instance_id :: %+v", request.Method, vars))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	switch request.Method {
	case "PUT":
		type instanceRequest struct {
			ServiceID      string `json:"service_id"`
			Plan           string `json:"plan_id"`
			OrganizationID string `json:"organization_guid"`
			SpaceID        string `json:"space_guid"`
		}
		ir := instanceRequest{}
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id %s", request.Method, err))
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, err)
			return
		}
		err = json.Unmarshal(body, &ir)
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id ! %s", request.Method, err))
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, err)
			return
		}
		instance, err := NewServiceInstance(
			vars["instance_id"],
			ir.ServiceID,
			ir.Plan,
			ir.OrganizationID,
			ir.SpaceID,
		)
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id ! %s", request.Method, err))
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, err)
			return
		}
		err = instance.Provision()
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id ! %s", request.Method, err))
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		msg := fmt.Sprintf(`Provisioned Instance %s`, instance.InstanceID)
		log.Trace(msg)
		fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusOK, msg)
		return
	case "DELETE":
		instance, err := instances.FindByInstanceID(vars["instance_id"])
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id ! %s", request.Method, err))
			w.WriteHeader(http.StatusInternalServerError)
			msg := fmt.Sprintf(`Could not find instance %s, perhaps it was already deleted?`, vars["instance_id"])
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, msg)
			return
		}
		err = instance.Decommission()
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id %s", request.Method, err))
			msg := fmt.Sprintf(`{"status": %d,"description": "There was an error decommissioning instance %s"}`, http.StatusInternalServerError, instance.InstanceID)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, msg, http.StatusInternalServerError)
			return
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": %d,"description": "Successfully Deprovisioned Instance %s"}`, http.StatusOK, instance.InstanceID)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"status": %d,"description": "Allowed Methods: PUT, DELETE"}`, http.StatusMethodNotAllowed)
		return
	}
}

/*
(CB) PUT /v2/service_instances/:instance_id/service_bindings/:binding_id
(RB) DELETE /v2/service_instances/:instance_id/service_bindings/:binding_id
*/
func BindingHandler(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	log.Trace(fmt.Sprintf("%s /v2/service_instances/:instance_id/service_bindings/:binding_id :: %+v", request.Method, vars))
	switch request.Method {
	case "PUT":
		binding := Binding{InstanceID: vars["instance_id"], BindingID: vars["binding_id"]}
		err := binding.Create()
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id/service_bindings/:binding_id %s", request.Method, err))
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, err)
			return
		}
		j, err := json.Marshal(binding)
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id/service_bindings/:binding_id %s", request.Method, err))
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, err)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(j)
			return
		}
	case "DELETE":
		binding := Binding{BindingID: vars["binding_id"]}
		err := binding.Remove()
		if err != nil {
			log.Error(fmt.Sprintf("%s /v2/service_instances/:instance_id/service_bindings/:binding_id %s", request.Method, err))
			msg := "Binding does not exist or has already been deleted."
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status": %d,"description": "%s"}`, http.StatusInternalServerError, msg)
			return
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": %d,"description": "Binding Removed"}`, http.StatusOK)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, `{"status": %d,"description": "Allowed Methods: PUT, DELETE"}`, http.StatusMethodNotAllowed)
		return
	}
}
