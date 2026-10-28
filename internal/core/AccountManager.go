package core

import (
	"cex_project/internal/utils"
	"errors"
	"math/big"
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

	if buyerQuote.lockedLessThan(big.NewInt(settlement.BuyerQuoteLocked)) {
		return nil, errors.New("insufficient buyer locked balance")
	}

	if sellerBase.lockedLessThan(big.NewInt(settlement.SellerBaseLocked)) {
		return nil, errors.New("insufficient seller locked balance")
	}

	buyerBase.addAvailable(big.NewInt(settlement.BuyerBaseReceived))
	sellerQuote.addAvailable(big.NewInt(settlement.SellerQuoteReceived))
	buyerQuote.addAvailable(big.NewInt(settlement.BuyerQuoteRefund))
	buyerQuote.subLocked(big.NewInt(settlement.BuyerQuoteLocked))
	sellerBase.subLocked(big.NewInt(settlement.SellerBaseLocked))

	updatedFees := make(map[Asset]Balance, len(feeBalances))
	for asset, balance := range feeBalances {
		updatedFees[asset] = balance.Clone()
	}

	baseFee := updatedFees[market.BaseAsset]
	baseFee.addAvailable(big.NewInt(settlement.BuyerFee))
	quoteFee := updatedFees[market.QuoteAsset]
	quoteFee.addAvailable(big.NewInt(settlement.SellerFee))
	updatedFees[market.BaseAsset] = baseFee
	updatedFees[market.QuoteAsset] = quoteFee

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
