package handlers

import (
	"cex_project/internal/core"
	"cex_project/internal/utils"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) ListTradesBySymbol(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("symbol is required"))
		return
	}

	trades, err := h.Services.Exchange.ListTradesBySymbol(symbol)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	displayed, err := displayTrades(trades)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, displayed)
}

func (h *Handlers) ListTradesByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || userID == 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}

	trades, err := h.Services.Exchange.ListTradesByUser(userID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	displayed, err := displayTrades(trades)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, displayed)
}

func displayTrades(trades []core.Trade) ([]map[string]any, error) {
	displayed := make([]map[string]any, 0, len(trades))
	for _, trade := range trades {
		item, err := displayTrade(trade)
		if err != nil {
			return nil, err
		}
		displayed = append(displayed, item)
	}

	return displayed, nil
}

func displayTrade(trade core.Trade) (map[string]any, error) {
	price, err := utils.PriceFromMinorUnits(trade.Price, string(trade.BaseAsset), string(trade.QuoteAsset))
	if err != nil {
		return nil, err
	}

	quantity, err := utils.FromMinorUnits(trade.Quantity, string(trade.BaseAsset))
	if err != nil {
		return nil, err
	}

	settlement, err := trade.Settlement()
	if err != nil {
		return nil, err
	}

	buyerFee, err := utils.FromMinorUnits(settlement.BuyerFee, string(trade.BaseAsset))
	if err != nil {
		return nil, err
	}

	sellerFee, err := utils.FromMinorUnits(settlement.SellerFee, string(trade.QuoteAsset))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":             trade.ID,
		"symbol":         trade.Symbol,
		"price":          price,
		"quantity":       quantity,
		"buy_order_id":   trade.BuyOrderID,
		"sell_order_id":  trade.SellOrderID,
		"buy_user_id":    trade.BuyUserID,
		"sell_user_id":   trade.SellUserID,
		"buyer_fee":      buyerFee,
		"seller_fee":     sellerFee,
		"executed_at":    trade.ExecutedAt,
	}, nil
}
