package handlers

import (
	"net/http"
	"cex_project/internal/utils"
)

const AuthCookieName = "auth_token"

func (h *Handlers) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie(AuthCookieName)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, err)
			return
		}

		if cookie.Value == "" {
			utils.WriteError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}

		payload, err := h.Paseto.VerifyToken(cookie.Value)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, err)
			return
		}
	
        next.ServeHTTP(w, r)
	})
}
