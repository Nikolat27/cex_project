package internal

import (
	"errors"
	"sync"
)

type Asset string

const (
	BTC Asset = "BTC"
	USDT Asset = "USDT"
)

type User struct {
	ID uint64
	balances map[Asset]*Balance

	mu sync.RWMutex
}

type Balance struct {
	Available int64
	Locked int64
}

type Market struct {
	Symbol string
	BaseAsset Asset // BTC
	QuoteAsset Asset // USDT
}

func NewUser(id uint64) *User {
	return &User{
		ID: id,
		balances: make(map[Asset]*Balance),
	}
}

// Deposit -> Adds to available balance from external payment
func (u *User) Deposit(asset Asset, amount int64) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)

	userBalance.Available += amount

	return nil
}

func (u *User) Withdraw(asset Asset, amount int64) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)

    if userBalance.Available < amount {
		return errors.New("insufficient balance")
	}

	userBalance.Available -= amount
	return nil
}

func (u *User) GetBalance(asset Asset) Balance {
	u.mu.RLock()
	defer u.mu.RUnlock()

	balance := u.getOrCreateBalance(asset)
	return *balance
}

func (u *User) getOrCreateBalance(asset Asset) *Balance {
	balance, exist := u.balances[asset]
	if !exist {
		newBalance := &Balance{}
		u.balances[asset] = newBalance
		return newBalance
	}

	return balance
}

func (u *User) Lock(asset Asset, amount int64) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)

	if userBalance.Available < amount {
		return errors.New("insufficient balance")
	}

    userBalance.Locked += amount
	userBalance.Available -= amount

	return nil
}

func (u *User) Unlock(asset Asset, amount int64) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)

	if userBalance.Locked < amount {
		return errors.New("insufficient locked balance")
	}

	userBalance.Locked -= amount
	userBalance.Available += amount

	return nil
}

// DebitLocked -> Removes from 'locked balance' after a trade is executed
func (u *User) DebitLocked(asset Asset, amount int64) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)

	if userBalance.Locked < amount {
		return errors.New("insufficient locked balance")
	}

	userBalance.Locked -= amount
	return nil
}

// Credit -> Adds to available balance from internal trades (transfer from other users)
func (u *User) Credit(asset Asset, amount int64) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)

	userBalance.Available += amount

	return nil
}
