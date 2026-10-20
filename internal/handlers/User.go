package handlers

import (
	"cex_project/internal/utils"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"time"
)

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if input.Username == "" || len(input.Password) < 8 {
		utils.WriteError(w, http.StatusBadRequest,
			errors.New("username and a password of at least 8 characters are required"))
		return
	}

	user, err := h.Services.CreateUser(r.Context(), input.Username, input.Password)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, user)
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if input.Username == "" || len(input.Password) < 8 {
		utils.WriteError(w, http.StatusBadRequest,
			 errors.New("username and a password of at least 8 characters are required"))
		return
	}

	token, err := h.Services.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	cookie := http.Cookie{
		Name: AuthCookieName,
		Value: token,
		MaxAge: 43200,
		Path: "/",
		Domain: "localhost",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure: false,
		Expires: time.Now().Add(43200 * time.Second),
	}

	http.SetCookie(w, &cookie)

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "login successful"})
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

	if err := h.RequireUser(userIDInt, r); err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}

	var input struct {
		Asset  string  `json:"asset"`
		Amount float64 `json:"amount"`
	}

	if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := utils.ValidateAsset(input.Asset); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	amount, err := utils.ToMinorUnits(input.Amount, input.Asset)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.Services.Deposit(r.Context(), userIDInt, input.Asset, amount); err != nil {
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

	if err := h.RequireUser(userIDInt, r); err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}
     
	var input struct {
		Asset  string  `json:"asset"`
		Amount float64 `json:"amount"`
	}

	if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := utils.ValidateAsset(input.Asset); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	amount, err := utils.ToMinorUnits(input.Amount, input.Asset)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.Services.Withdraw(r.Context(), userIDInt, input.Asset, amount); err != nil {
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

	if err := h.RequireUser(userIDInt, r); err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}

	asset := chi.URLParam(r, "asset")
	if err := utils.ValidateAsset(asset); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	balance, err := h.Services.GetBalance(r.Context(), userIDInt, asset)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	availableAmount, err := utils.FromMinorUnits(balance.Available, asset)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	lockedAmount, err := utils.FromMinorUnits(balance.Locked, asset)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]float64{
		"available": availableAmount,
		"locked":    lockedAmount,
	})
}

// RequireUser -> Checks if the user`s input ID matches the UserID in the HttpOnly cookie
func (h *Handlers) RequireUser(userID int64, r *http.Request) error {
	cookie, err := r.Cookie(AuthCookieName)
	if err != nil {
		return err
	}

	if cookie.Value == "" {
		return errors.New("unauthorized")
	}

	payload, err := h.Services.Paseto.VerifyToken(cookie.Value)
	if err != nil {
		return err
	}

    if payload.UserID != userID {
		return errors.New("unauthorized")
	}

	return nil
}
