package main

import (
	"log"
	"net/http"
)

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("MESSAGE: %s path: %s err: %s", r.Method, r.URL.Path, err)
	writeError(w, http.StatusBadRequest, err.Error())
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("MESSAGE: %s path: %s err: %s", r.Method, r.URL.Path, err)
	writeError(w, http.StatusBadRequest, err.Error())
}

func (app *application) unauthorizedErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("unauthorized error: %s path: %s err: %s", r.Method, r.URL.Path, err)
	writeError(w, http.StatusUnauthorized, "Unauthorized method used")
}

// forbiddenResponse — autentifikatsiya bor, lekin resurs boshqa business/kompaniyaga tegishli.
func (app *application) forbiddenResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("forbidden: %s path: %s err: %s", r.Method, r.URL.Path, err)
	writeError(w, http.StatusForbidden, err.Error())
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("not found: %s path: %s err: %s", r.Method, r.URL.Path, err)
	writeError(w, http.StatusNotFound, "NOT FOUND")
}
