package handlers

import (
	"cex_project/internal/utils"
	"errors"
	"net/http"
)

func (h *Handlers) GetOrCreateDepositAddress(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.UserID(r)
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
		userID, symbol, network)
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
