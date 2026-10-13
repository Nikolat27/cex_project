package internal

import (
	"cex_project/internal/utils"
	"errors"
	"sync"
)

const (
	EXCHANGE_FEE int64 = 25 // 0.25% = 25 / 10000
)

type AccountManager struct {
	users map[uint64]*User
	mu    sync.RWMutex
}

func NewAccountManager() *AccountManager {
	return &AccountManager{
		users: make(map[uint64]*User),
	}
}

func (am *AccountManager) GetUser(id uint64) *User {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.getUserUnlocked(id)
}

func (am *AccountManager) getUserUnlocked(id uint64) *User {
	return am.users[id]
}

func (am *AccountManager) CreateUser(id uint64) (*User, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.getUserUnlocked(id) != nil {
		return nil, errors.New("user with this ID already exists")
	}

	newUser := NewUser(id)

	am.users[id] = newUser

	return newUser, nil
}

func (am *AccountManager) DeleteUser(id uint64) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user := am.getUserUnlocked(id)
	if user == nil {
		return errors.New("user with this ID does not exist")
	}

	delete(am.users, id)
	return nil
}

func (am *AccountManager) settleTrade(market Market, trade *Trade,
	 feeBalances map[Asset]Balance) (map[Asset]Balance, error) {
		
	if trade == nil {
		return nil, errors.New("trade is nil")
	}

	if trade.BuyUserID == trade.SellUserID {
		return nil, errors.New("self-trading is not supported")
	}

	// limit price here means the maximum price the buyer is willing to pay
	if trade.Price > trade.BuyLimitPrice {
		return nil, errors.New("trade price exceeds buy limit")
	}

	reserved, err := utils.SafeMultiply(trade.BuyLimitPrice, trade.Quantity)
	if err != nil {
		return nil, err
	}

	actual, err := utils.SafeMultiply(trade.Price, trade.Quantity)
	if err != nil {
		return nil, err
	}

	refund := reserved - actual

	am.mu.RLock()
	defer am.mu.RUnlock()

	buyer := am.getUserUnlocked(trade.BuyUserID)
	if buyer == nil {
		return nil, errors.New("buyer user not found")
	}

	seller := am.getUserUnlocked(trade.SellUserID)
	if seller == nil {
		return nil, errors.New("seller user not found")
	}

	unlock := lockUsers(buyer, seller)
	defer unlock()

	buyerQuote := buyer.getOrCreateBalance(market.QuoteAsset)
	buyerBase := buyer.getOrCreateBalance(market.BaseAsset)
	sellerBase := seller.getOrCreateBalance(market.BaseAsset)
	sellerQuote := seller.getOrCreateBalance(market.QuoteAsset)

	if buyerQuote.Locked < reserved {
		return nil, errors.New("insufficient buyer locked balance")
	}

	if sellerBase.Locked < trade.Quantity {
		return nil, errors.New("insufficient seller locked balance")
	}

	updatedFees, buyerReceives, sellerReceives, err := am.deductExchangeFee(
		trade.Quantity,
		actual,
		market.BaseAsset,
		market.QuoteAsset,
		feeBalances,
	)
	if err != nil {
		return nil, err
	}

	newBuyerBaseAvailable, err := utils.SafeAdd(buyerBase.Available, buyerReceives)
	if err != nil {
		return nil, err
	}

	newSellerQuoteAvailable, err := utils.SafeAdd(sellerQuote.Available, sellerReceives)
	if err != nil {
		return nil, err
	}

	newBuyerQuoteAvailable, err := utils.SafeAdd(buyerQuote.Available, refund)
	if err != nil {
		return nil, err
	}

	buyerQuote.Locked -= reserved
	sellerBase.Locked -= trade.Quantity

	buyerBase.Available = newBuyerBaseAvailable
	sellerQuote.Available = newSellerQuoteAvailable
	buyerQuote.Available = newBuyerQuoteAvailable

	return updatedFees, nil
}

func (am *AccountManager) deductExchangeFee(baseQuantity, quoteAmount int64, buyerAsset,
	sellerAsset Asset, currentFeeBalances map[Asset]Balance) (map[Asset]Balance, int64, int64, error) {

	newFeeBalances := make(map[Asset]Balance)
	for asset, balance := range currentFeeBalances {
		newFeeBalances[asset] = balance
	}

	buyerFee, err := feeAmount(baseQuantity)
	if err != nil {
		return nil, 0, 0, err
	}

	sellerFee, err := feeAmount(quoteAmount)
	if err != nil {
		return nil, 0, 0, err
	}

	newBuyerFee, err := utils.SafeAdd(newFeeBalances[buyerAsset].Available, buyerFee)
	if err != nil {
		return nil, 0, 0, err
	}
	newFeeBalances[buyerAsset] = Balance{Available: newBuyerFee}

	newSellerFee, err := utils.SafeAdd(newFeeBalances[sellerAsset].Available, sellerFee)
	if err != nil {
		return nil, 0, 0, err
	}
	newFeeBalances[sellerAsset] = Balance{Available: newSellerFee}

	buyerReceives := baseQuantity - buyerFee
	sellerReceives := quoteAmount - sellerFee

	return newFeeBalances, buyerReceives, sellerReceives, nil
}

func feeAmount(amount int64) (int64, error) {
	if amount == 0 {
		return 0, nil
	}

	scaled, err := utils.SafeMultiply(amount, EXCHANGE_FEE)
	if err != nil {
		return 0, err
	}

	return scaled / 10000, nil
}

func lockUsers(first, second *User) func() {
	// We lock the user with the lower ID (older user) first to prevent deadlocks
	// because in our other goroutines, we lock the user with the lower ID first.
	if first.ID > second.ID {
		first, second = second, first
	}

	first.mu.Lock()
	second.mu.Lock()

	return func() {
		second.mu.Unlock()
		first.mu.Unlock()
	}
}
