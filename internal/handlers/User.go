package handlers

import (
	"cex_project/internal/utils"
	"errors"
	"net/http"
	"strconv"
	"github.com/go-chi/chi/v5"
)

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.Services.CreateUser(r.Context())
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, user)
}

func (h *Handlers) Deposit(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("user ID is required"))
		return
	}

	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	var input struct {
		Asset string `json:"asset"`
		Amount int64 `json:"amount"`
	}

    if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if input.Asset == "" || input.Amount <= 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid input"))
		return
	}

	if err := h.Services.Deposit(r.Context(), userIDInt, input.Asset, input.Amount); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "deposit successful"})
}

func (h *Handlers) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("user ID is required"))
		return
	}

	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	var input struct {
		Asset string `json:"asset"`
		Amount int64 `json:"amount"`
	}

    if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if input.Asset == "" || input.Amount <= 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid input"))
		return
	}

	if err := h.Services.Withdraw(r.Context(), userIDInt, input.Asset, input.Amount); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "withdraw successful"})
}

func (h *Handlers) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("user ID is required"))
		return
	}

	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

    asset := chi.URLParam(r, "asset")
	if asset == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("asset is required"))
		return
	}

	amount, err := h.Services.GetBalance(r.Context(), userIDInt, asset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]int64{"balance": amount})
}
