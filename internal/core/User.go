package core

import (
	"cex_project/internal/utils"
	"errors"
	"sync"
)

type User struct {
	ID       uint64
	balances map[Asset]*Balance

	mu sync.RWMutex
}

func NewUser(id uint64) *User {
	return &User{
		ID:       id,
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

	newAvailable, err := utils.SafeAdd(userBalance.Available, amount)
	if err != nil {
		return err
	}
	userBalance.Available = newAvailable

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

	balance, exists := u.balances[asset]
	if !exists {
		return Balance{}
	}
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

	newLocked, err := utils.SafeAdd(userBalance.Locked, amount)
	if err != nil {
		return err
	}

	userBalance.Locked = newLocked
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

	newAvailable, err := utils.SafeAdd(userBalance.Available, amount)
	if err != nil {
		return err
	}

	userBalance.Locked -= amount
	userBalance.Available = newAvailable

	return nil
}

// Credit -> Adds to 'available balance' from internal trades (transfer from other users)
func (u *User) Credit(asset Asset, amount int64) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)

	newAvailable, err := utils.SafeAdd(userBalance.Available, amount)
	if err != nil {
		return err
	}
	userBalance.Available = newAvailable

	return nil
}

// Debit -> Removes from 'locked balance' after a trade is executed
func (u *User) Debit(asset Asset, amount int64) error {
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
