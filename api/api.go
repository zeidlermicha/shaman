// Package api provides a restful interface to manage entries in the DNS database.
package api

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/pat"
	nanoauth "github.com/nanobox-io/golang-nanoauth"

	"github.com/nanopack/shaman/config"
)



var (
	auth            nanoauth.Auth
	errBadJson      = errors.New("Bad JSON syntax received in body")
	errBodyReadFail = errors.New("Body Read Failed")
)

// Start starts shaman's http api
func Start() error {
	auth.Header = "X-AUTH-TOKEN"

	// handle config.Insecure
	if config.Insecure {
		config.Log.Info("Shaman listening at http://%s...", config.ApiListen)
		return fmt.Errorf("API stopped - %v", auth.ListenAndServe(config.ApiListen, config.ApiToken, routes()))
	}

	var cert *tls.Certificate
	var err error
	if config.ApiCrt == "" {
		cert, err = nanoauth.Generate(config.ApiDomain)
	} else {
		cert, err = nanoauth.Load(config.ApiCrt, config.ApiKey, config.ApiKeyPassword)
	}
	if err != nil {
		return fmt.Errorf("Failed to generate or load cert - %s", err.Error())
	}

	auth.Certificate = cert

	config.Log.Info("Shaman listening at https://%v", config.ApiListen)

	return fmt.Errorf("API stopped - %v", auth.ListenAndServeTLS(config.ApiListen, config.ApiToken, routes()))
}

func routes() *pat.Router {
	router := pat.New()

	router.Delete("/records/{domain}", deleteRecord) // delete resource
	router.Put("/records/{domain}", updateRecord)    // reset resource's records
	router.Get("/records/{domain}", getRecord)       // return resource's records

	router.Post("/records", createRecord) // add a resource
	router.Get("/records", listRecords)   // return all domains
	router.Put("/records", updateAnswers) // reset all resources

	return router
}

func writeBody(rw http.ResponseWriter, req *http.Request, v interface{}, status int) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// print the error only if there is one
	var msg map[string]string
	json.Unmarshal(b, &msg)

	var errMsg string
	if msg["error"] != "" {
		errMsg = msg["error"]
	}

	config.Log.Debug("%s %d %s %s %s", req.RemoteAddr, status, req.Method, req.RequestURI, errMsg)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	rw.Write(append(b, byte('\n')))

	return nil
}

// parseBody parses the json body into v
func parseBody(req *http.Request, v interface{}) error {

	// read the body
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		config.Log.Error(err.Error())
		return errBodyReadFail
	}
	defer req.Body.Close()

	// parse body and store in v
	err = json.Unmarshal(b, v)
	if err != nil {
		return errBadJson
	}

	return nil
}
