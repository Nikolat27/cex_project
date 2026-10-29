package handlers

import (
	"cex_project/internal/utils"
	"errors"
	"net/http"
)

const AuthCookieName = "auth_token"

func (h *Handlers) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(AuthCookieName)
		if err != nil && errors.Is(err, http.ErrNoCookie) {
			utils.WriteError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}

		if cookie.Value == "" {
			utils.WriteError(w, http.StatusUnauthorized, errors.New("unauthorized"))
			return
		}

		if _, err := h.Services.Paseto.VerifyToken(cookie.Value); err != nil {
			utils.WriteError(w, http.StatusUnauthorized, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}
