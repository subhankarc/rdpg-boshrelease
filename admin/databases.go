package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/starkandwayne/rdpgd/instances"
	"github.com/starkandwayne/rdpgd/log"
)

/*
POST /databases/register
PUT /databases/assign
*/
func DatabasesHandler(w http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	log.Trace(fmt.Sprintf("admin.DatabasesHandler() > %s /databases/%s %+v", request.Method, vars["action"], vars))
	switch request.Method {
	case "GET":
		switch vars["action"] {
		case "": // List All Databases
			instances, err := instances.All()
			if err != nil {
				msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
				log.Error(fmt.Sprintf(`admin.DatabasesHandler(): instances.All() %s %+v ! %s`, msg, vars, err))
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			jsonInstances, err := json.Marshal(instances)
			if err != nil {
				msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
				log.Error(fmt.Sprintf(`admin.DatabasesHandler(): json.Marshal(instances) %s %+v ! %s`, msg, vars, err))
				http.Error(w, msg, http.StatusInternalServerError)
			} else {
				w.Header().Set("Content-Type", "application/json; charset=UTF-8")
				w.WriteHeader(http.StatusOK)
				w.Write(jsonInstances)
			}
		case "available": // Lists Available Databases
			instances, err := instances.Available()
			if err != nil {
				msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
				log.Error(fmt.Sprintf(`admin.DatabasesHandler(): instances.Available() %s %+v ! %s`, msg, vars, err))
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			jsonInstances, err := json.Marshal(instances)
			if err != nil {
				msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
				log.Error(fmt.Sprintf(`admin.DatabasesHandler(): json.Marshal(instances) %s %+v ! %s`, msg, vars, err))
				http.Error(w, msg, http.StatusInternalServerError)
			} else {
				w.Header().Set("Content-Type", "application/json; charset=UTF-8")
				w.WriteHeader(http.StatusOK)
				w.Write(jsonInstances)
			}
		default:
			msg := fmt.Sprintf(`{"status": %d, "description": "Invalid Action %s"}`, http.StatusBadRequest, vars["action"])
			log.Error(fmt.Sprintf(`admin.DatabasesHandler(): %s %s`, msg, vars))
			http.Error(w, msg, http.StatusBadRequest)
		}
	case `POST`:
		var i instances.Instance
		decoder := json.NewDecoder(request.Body)
		err := decoder.Decode(&i)
		if err != nil {
			msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
			log.Error(fmt.Sprintf(`admin.DatabasesHandler(): decoder.Decode() %s %s ! %s`, msg, vars, err))
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		switch vars[`action`] {
		case `register`: // Creates a new record.
			err = i.Register()
			if err != nil {
				msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
				log.Error(fmt.Sprintf(`admin.DatabasesHandler(): Instance#Register() %s %s ! %s`, msg, vars, err))
				http.Error(w, msg, http.StatusInternalServerError)
				return
			} else {
				w.Header().Set(`Content-Type`, `application/json; charset=UTF-8`)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
				return
			}
		default:
			msg := fmt.Sprintf(`{"status": %d, "description": "Invalid Action %s"}`, http.StatusBadRequest, vars[`action`])
			log.Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
		}
	case `PUT`:
		var i instances.Instance
		decoder := json.NewDecoder(request.Body)
		err := decoder.Decode(&i)
		if err != nil {
			msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
			log.Error(fmt.Sprintf(`admin.DatabasesHandler(): decoder.Decode() %s %s ! %s`, msg, vars, err))
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		switch vars[`action`] {
		case `assign`: // updates an existing record.
			err = i.Assign()
			if err != nil {
				msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
				log.Error(fmt.Sprintf(`admin.DatabasesHandler(): instances.Assign() %s %s ! %s`, msg, vars, err))
				http.Error(w, msg, http.StatusInternalServerError)
				return
			} else {
				w.Header().Set(`Content-Type`, `application/json; charset=UTF-8`)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
				return
			}
		default:
			msg := fmt.Sprintf(`{"status": %d, "description": "Invalid Action %s"}`, http.StatusBadRequest, vars[`action`])
			log.Error(fmt.Sprintf(`admin.DatabasesHandler(): %s %s`, msg, vars))
			http.Error(w, msg, http.StatusBadRequest)
		}
	case `DELETE`:
		switch vars[`action`] {
		case `decommission`:
			i, err := instances.FindByDatabase(vars[`database`])
			if err != nil {
				msg := fmt.Sprintf(`{"status": %d, "description": "%s"}`, http.StatusInternalServerError, err)
				log.Error(fmt.Sprintf(`admin.DatabasesHandler(): instance.FindByDatabase() %s %s ! %s`, msg, vars, err))
				http.Error(w, msg, http.StatusInternalServerError)
				return
			} else {
				err = i.Decommission()
				if err != nil {
					msg := fmt.Sprintf(`{"status": %d, "description": "There was an error decommissioning the database (%s)"}`, http.StatusInternalServerError, err)
					log.Error(fmt.Sprintf(`admin.DatabasesHandler(): instance#Decommission() %s %s ! %s`, msg, vars, err))
					http.Error(w, msg, http.StatusInternalServerError)
				}
			}
		default:
			msg := fmt.Sprintf(`{"status": %d, "description": "Invalid Action %s"}`, http.StatusBadRequest, vars[`action`])
			log.Error(fmt.Sprintf(`admin.DatabasesHandler(): %s %s`, msg, vars))
			http.Error(w, msg, http.StatusBadRequest)
		}
	default:
		msg := fmt.Sprintf(`{"status": %d, "description": "Method not allowed %s"}`, http.StatusMethodNotAllowed, request.Method)
		log.Error(fmt.Sprintf(`admin.DatabasesHandler(): %s %s`, msg, vars))
		http.Error(w, msg, http.StatusMethodNotAllowed)
		return
	}
}
