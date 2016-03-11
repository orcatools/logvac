package api

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gorilla/pat"
	"github.com/nanobox-io/golang-nanoauth"

	"github.com/nanopack/logvac/authenticator"
	"github.com/nanopack/logvac/config"
)

// start the web server with the logvac functions
func Start(collector http.HandlerFunc, retriever http.HandlerFunc) error {
	router := pat.New()

	router.Get("/add-token", handleRequest(addKey))
	router.Get("/remove-token", handleRequest(removeKey))

	router.Post("/", verify(handleRequest(collector)))
	router.Get("/", verify(handleRequest(retriever)))

	// blocking...
	if config.Insecure {
		config.Log.Info("Api Listening on http://%s...", config.ListenHttp)
		return http.ListenAndServe(config.ListenHttp, router)
	}
	config.Log.Info("Api Listening on https://%s...", config.ListenHttp)
	cert, _ := nanoauth.Generate("nanobox.io")
	auth := nanoauth.Auth{
		Header:      "X-ADMIN-TOKEN",
		Certificate: cert,
	}
	return auth.ListenAndServeTLS(config.ListenHttp, config.Token, router, "/")
}

// handleRequest add a bit of logging
func handleRequest(fn http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if config.Insecure {
			rw.Header().Set("Access-Control-Allow-Origin", "*")
		}

		fn(rw, req)

		// must be after req returns
		getStatus := func(trw http.ResponseWriter) string {
			r, _ := regexp.Compile("status:([0-9]*)")
			return r.FindStringSubmatch(fmt.Sprintf("%+v", trw))[1]
		}

		getWrote := func(trw http.ResponseWriter) string {
			r, _ := regexp.Compile("written:([0-9]*)")
			return r.FindStringSubmatch(fmt.Sprintf("%+v", trw))[1]
		}

		config.Log.Debug(`%v - [%v] %v %v %v(%s) - "User-Agent: %s"`,
			req.RemoteAddr, req.Proto, req.Method, req.RequestURI,
			getStatus(rw), getWrote(rw), // %v(%s)
			req.Header.Get("User-Agent"))
	}
}

// verify that the token is allowed throught the authenticator
func verify(fn http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		key := req.Header.Get("X-AUTH-TOKEN")
		// allow browsers to authenticate/fetch logs
		if key == "" {
			query := req.URL.Query()
			key = query.Get("auth")
		}
		if !authenticator.Valid(key) {
			rw.WriteHeader(401)
			return
		}
		fn(rw, req)
	}
}
