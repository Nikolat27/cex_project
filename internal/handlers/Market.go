package handlers

import (
	"cex_project/internal/core"
	"cex_project/internal/utils"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) AddMarket(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Symbol     string `json:"symbol"`
		BaseAsset  string `json:"base_asset"`
		QuoteAsset string `json:"quote_asset"`
	}

	if err := utils.ReadJSON(r, &input, 1024); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := utils.ValidateAsset(input.BaseAsset); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := utils.ValidateAsset(input.QuoteAsset); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.Services.AddMarket(r.Context(), &core.Market{
		Symbol:     input.Symbol,
		BaseAsset:  core.Asset(input.BaseAsset),
		QuoteAsset: core.Asset(input.QuoteAsset),
	}); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"message": "market created"})
}

func (h *Handlers) GetOrderBook(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("symbol is required"))
		return
	}

	market, err := h.Services.Exchange.GetMarket(symbol)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	snapshot, err := h.Services.Exchange.GetOrderBookSnapshot(symbol)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	bids, err := displayLevels(snapshot.Bids, market)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	asks, err := displayLevels(snapshot.Asks, market)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"symbol": symbol,
		"bids":   bids,
		"asks":   asks,
	})
}

func displayLevels(levels []core.PriceLevel, market *core.Market) ([]map[string]float64, error) {
	displayed := make([]map[string]float64, 0, len(levels))
	for _, level := range levels {
		price, err := utils.PriceFromMinorUnits(level.Price, string(market.QuoteAsset))
		if err != nil {
			return nil, err
		}

		quantity, err := utils.FromMinorUnits(level.Quantity, string(market.BaseAsset))
		if err != nil {
			return nil, err
		}

		displayed = append(displayed, map[string]float64{
			"price":    price,
			"quantity": quantity,
		})
	}

	return displayed, nil
}
