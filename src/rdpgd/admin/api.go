package admin

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/starkandwayne/rdpgd/log"
)

var (
	adminPort, adminUser, adminPass string
	pgPort, pbPort                  string
)

type Admin struct {
}

func init() {
	adminPort = os.Getenv(`RDPGD_ADMIN_PORT`)
	if adminPort == "" {
		adminPort = `58888`
	}
	adminUser = os.Getenv(`RDPGD_ADMIN_USER`)
	if adminUser == "" {
		adminUser = `admin`
	}
	adminPass = os.Getenv(`RDPGD_ADMIN_PASS`)
	if adminPass == "" {
		adminPass = `admin`
	}
	pgPort = os.Getenv(`RDPGD_PG_PORT`)
	if pgPort == `` {
		pgPort = `5432`
	}
	pbPort = os.Getenv(`RDPGD_PB_PORT`)
	if pbPort == `` {
		pbPort = `5432`
	}
}

func API() (err error) {
	AdminMux := http.NewServeMux()
	router := mux.NewRouter()
	router.HandleFunc(`/health/{check}`, httpAuth(HealthHandler))
	router.HandleFunc(`/services/{service}/{action}`, httpAuth(ServiceHandler))
	router.HandleFunc(`/databases`, httpAuth(DatabasesHandler))
	router.HandleFunc(`/databases/{action}`, httpAuth(DatabasesHandler))
	router.HandleFunc(`/databases/{action}/{database}`, httpAuth(DatabasesHandler))
	AdminMux.Handle("/", router)
	err = http.ListenAndServe(":"+adminPort, AdminMux)
	log.Error(fmt.Sprintf(`admin.API() ! %s`, err))
	return
}

func httpAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		if len(request.Header[`Authorization`]) == 0 {
			log.Trace(fmt.Sprintf(`httpAuth(): Authorization Required`))
			http.Error(w, `Authorization Required`, http.StatusUnauthorized)
			return
		}

		auth := strings.SplitN(request.Header[`Authorization`][0], " ", 2)
		if len(auth) != 2 || auth[0] != `Basic` {
			log.Error(fmt.Sprintf(`httpAuth(): Unhandled Authorization Type, Expected Basic`))
			http.Error(w, `Unhandled Authroization Type, Expected Basic\n`, http.StatusBadRequest)
			return
		}
		payload, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			log.Error(fmt.Sprintf(`httpAuth(): Authorization Failed`))
			http.Error(w, `Authorization Failed\n`, http.StatusUnauthorized)
			return
		}
		nv := strings.SplitN(string(payload), ":", 2)
		if (len(nv) != 2) || !isAuthorized(nv[0], nv[1]) {
			log.Error(fmt.Sprintf(`httpAuth(): Authorization Failed`))
			http.Error(w, `Authorization Failed\n`, http.StatusUnauthorized)
			return
		}
		h(w, request)
	}
}

func isAuthorized(username, password string) bool {
	if username == adminUser && password == adminPass {
		return true
	}
	return false
}
