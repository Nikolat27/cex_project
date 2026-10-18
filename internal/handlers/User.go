package handlers

import "net/http"

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}
