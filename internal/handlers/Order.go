package handlers

import (
	"cex_project/internal/core"
	"cex_project/internal/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID   int64   `json:"user_id"`
		Symbol   string  `json:"symbol"`
		Side     string  `json:"side"`
		Type     string  `json:"type"`
		Price    float64 `json:"price"`
		Quantity float64 `json:"quantity"`
	}

	if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if input.UserID <= 0 || input.Symbol == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid input"))
		return
	}

	if err := h.RequireUser(input.UserID, r); err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}

	var side core.Side
	switch input.Side {
	case "buy":
		side = core.Buy
	case "sell":
		side = core.Sell
	default:
		utils.WriteError(w, http.StatusBadRequest, errors.New("unknown order side"))
		return
	}

	if input.Type != "limit" && input.Type != "market" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("unknown order type"))
		return
	}

	h.Services.Mu.RLock()
	market, err := h.Services.Exchange.GetMarket(input.Symbol)
	h.Services.Mu.RUnlock()

	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := utils.ValidateAsset(string(market.BaseAsset)); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if err := utils.ValidateAsset(string(market.QuoteAsset)); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	quantity, err := utils.ToMinorUnits(input.Quantity, string(market.BaseAsset))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	price, err := utils.PriceToMinorUnits(input.Price, string(market.QuoteAsset))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	order := &core.Order{
		UserID:   uint64(input.UserID),
		Symbol:   input.Symbol,
		Side:     side,
		Price:    price,
		Quantity: quantity,
	}

	if input.Type == "limit" {
		err = h.Services.PlaceLimitOrder(r.Context(), order)
	} else {
		err = h.Services.PlaceMarketOrder(r.Context(), order)
	}

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{"order_id": order.ID,
		"status": order.Status},
	)
}

func (h *Handlers) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || orderID == 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid order id"))
		return
	}

	var input struct {
		UserID int64  `json:"user_id"`
		Symbol string `json:"symbol"`
	}

	if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if input.UserID <= 0 || input.Symbol == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid input"))
		return
	}

	if err := h.RequireUser(input.UserID, r); err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}

	if err := h.Services.CancelOrder(r.Context(), &core.Order{
		ID:     orderID,
		UserID: uint64(input.UserID),
		Symbol: input.Symbol,
	}); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "order cancelled"})
}

func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	h.Services.Mu.RLock()
	defer h.Services.Mu.RUnlock()

	orderID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || orderID == 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid order id"))
		return
	}

	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("symbol is required"))
		return
	}

	market, err := h.Services.Exchange.GetMarket(symbol)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	order, err := h.Services.Exchange.GetOrder(symbol, orderID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	if err := h.RequireUser(int64(order.UserID), r); err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}

	displayed, err := displayOrder(order, market)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, displayed)
}

func (h *Handlers) ListUserOrders(w http.ResponseWriter, r *http.Request) {
	h.Services.Mu.RLock()
	defer h.Services.Mu.RUnlock()

	id, err := utils.UserID(r)
	if err != nil || id <= 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}
	userID := uint64(id)

	if err := h.RequireUser(id, r); err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}

	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("symbol is required"))
		return
	}

	market, err := h.Services.Exchange.GetMarket(symbol)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	orders, err := h.Services.Exchange.ListOrdersByUserID(symbol, userID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	displayed := make([]map[string]any, 0, len(orders))
	for _, order := range orders {
		item, err := displayOrder(order, market)
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err)
			return
		}

		displayed = append(displayed, item)
	}

	utils.WriteJSON(w, http.StatusOK, displayed)
}

func displayOrder(order core.Order, market *core.Market) (map[string]any, error) {
	price, err := utils.PriceFromMinorUnits(order.Price, string(market.QuoteAsset))
	if err != nil {
		return nil, err
	}

	quantity, err := utils.FromMinorUnits(order.Quantity, string(market.BaseAsset))
	if err != nil {
		return nil, err
	}

	filled, err := utils.FromMinorUnits(order.Filled, string(market.BaseAsset))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":       order.ID,
		"side":     orderSide(order.Side),
		"type":     orderType(order.Type),
		"status":   order.Status,
		"price":    price,
		"quantity": quantity,
		"filled":   filled,
	}, nil
}

func orderSide(side core.Side) string {
	if side == core.Sell {
		return "sell"
	}
	return "buy"
}

func orderType(orderType core.OrderType) string {
	if orderType == core.MarketType {
		return "market"
	}
	return "limit"
}
