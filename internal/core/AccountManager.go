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

	settlement, err := trade.Settlement()
	if err != nil {
		return nil, err
	}

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

	if buyerQuote.Locked < settlement.BuyerQuoteLocked {
		return nil, errors.New("insufficient buyer locked balance")
	}

	if sellerBase.Locked < settlement.SellerBaseLocked {
		return nil, errors.New("insufficient seller locked balance")
	}

	newBuyerBaseAvailable, err := utils.SafeAdd(buyerBase.Available, settlement.BuyerBaseReceived)
	if err != nil {
		return nil, err
	}

	newSellerQuoteAvailable, err := utils.SafeAdd(sellerQuote.Available, settlement.SellerQuoteReceived)
	if err != nil {
		return nil, err
	}

	newBuyerQuoteAvailable, err := utils.SafeAdd(buyerQuote.Available, settlement.BuyerQuoteRefund)
	if err != nil {
		return nil, err
	}

	buyerQuote.Locked -= settlement.BuyerQuoteLocked
	sellerBase.Locked -= settlement.SellerBaseLocked

	buyerBase.Available = newBuyerBaseAvailable
	sellerQuote.Available = newSellerQuoteAvailable
	buyerQuote.Available = newBuyerQuoteAvailable

	updatedFees := make(map[Asset]Balance, len(feeBalances))
	for asset, balance := range feeBalances {
		updatedFees[asset] = balance
	}

	baseFee, err := utils.SafeAdd(updatedFees[market.BaseAsset].Available, settlement.BuyerFee)
	if err != nil {
		return nil, err
	}
	quoteFee, err := utils.SafeAdd(updatedFees[market.QuoteAsset].Available, settlement.SellerFee)
	if err != nil {
		return nil, err
	}
	updatedFees[market.BaseAsset] = Balance{Available: baseFee}
	updatedFees[market.QuoteAsset] = Balance{Available: quoteFee}

	return updatedFees, nil
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
