package handlers

import (
	"cex_project/internal/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) GetOrCreateDepositAddress(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("user ID is required"))
		return
	}

	userIDInt, err := strconv.ParseInt(userID, 10 ,64)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	query := r.URL.Query()
	if !query.Has("symbol") || !query.Has("network") {
		utils.WriteError(w, http.StatusBadRequest, errors.New("symbol and network are required"))
		return
	}

	symbol := query.Get("symbol")
	network := query.Get("network")
	if symbol == "" || network == "" {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	address, err := h.Services.GetOrCreateDepositAddress(r.Context(),
	 userIDInt, symbol, network)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}


    utils.WriteJSON(w, http.StatusOK, map[string]any{
		"address":          address.Address,
		"symbol":           address.Symbol,
		"network":          address.Network,
		"derivation_index": address.DerivationIndex,
	})
}
