package internal

import (
	"cex_project/internal/utils"
	"errors"
	"sync"
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

func (am *AccountManager) settleTrade(market Market, trade *Trade) error {
	if trade == nil {
		return errors.New("trade is nil")
	}

	if trade.BuyUserID == trade.SellUserID {
		return errors.New("self-trading is not supported")
	}

	// limit price here means the maximum price the buyer is willing to pay
	if trade.Price > trade.BuyLimitPrice {
		return errors.New("trade price exceeds buy limit")
	}

	reserved, err := utils.SafeMultiply(trade.BuyLimitPrice, trade.Quantity)
	if err != nil {
		return err
	}

	actual, err := utils.SafeMultiply(trade.Price, trade.Quantity)
	if err != nil {
		return err
	}

	refund := reserved - actual

	am.mu.RLock()
	defer am.mu.RUnlock()

	buyer := am.getUserUnlocked(trade.BuyUserID)
	if buyer == nil {
		return errors.New("buyer user not found")
	}

	seller := am.getUserUnlocked(trade.SellUserID)
	if seller == nil {
		return errors.New("seller user not found")
	}

	unlock := lockUsers(buyer, seller)
	defer unlock()

	buyerQuote := buyer.getOrCreateBalance(market.QuoteAsset)
	buyerBase := buyer.getOrCreateBalance(market.BaseAsset)
	sellerBase := seller.getOrCreateBalance(market.BaseAsset)
	sellerQuote := seller.getOrCreateBalance(market.QuoteAsset)

	if buyerQuote.Locked < reserved {
		return errors.New("insufficient buyer locked balance")
	}

	if sellerBase.Locked < trade.Quantity {
		return errors.New("insufficient seller locked balance")
	}

	newBuyerBaseAvailable, err := utils.SafeAdd(buyerBase.Available, trade.Quantity)
	if err != nil {
		return err
	}

	newSellerQuoteAvailable, err := utils.SafeAdd(sellerQuote.Available, actual)
	if err != nil {
		return err
	}

	newBuyerQuoteAvailable, err := utils.SafeAdd(buyerQuote.Available, refund)
	if err != nil {
		return err
	}

	buyerQuote.Locked -= reserved
	sellerBase.Locked -= trade.Quantity
	buyerBase.Available = newBuyerBaseAvailable
	sellerQuote.Available = newSellerQuoteAvailable
	buyerQuote.Available = newBuyerQuoteAvailable

	return nil
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
