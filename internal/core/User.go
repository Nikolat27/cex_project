package core

import (
	"errors"
	"math/big"
	"sync"
)

type User struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	balances map[Asset]*Balance

	mu sync.RWMutex
}

func NewUser(id uint64) *User {
	return &User{
		ID:       id,
		balances: make(map[Asset]*Balance),
	}
}

func (u *User) Deposit(asset Asset, amount int64) error {
	return u.DepositAmount(asset, big.NewInt(amount))
}

func (u *User) DepositAmount(asset Asset, amount *big.Int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount == nil || amount.Sign() <= 0 {
		return errors.New("amount must be greater than 0")
	}

	u.getOrCreateBalance(asset).addAvailable(amount)
	return nil
}

func (u *User) Withdraw(asset Asset, amount int64) error {
	return u.WithdrawAmount(asset, big.NewInt(amount))
}

func (u *User) WithdrawAmount(asset Asset, amount *big.Int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount == nil || amount.Sign() <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)
	if userBalance.availableLessThan(amount) {
		return errors.New("insufficient balance")
	}

	userBalance.subAvailable(amount)
	return nil
}

func (u *User) GetBalance(asset Asset) Balance {
	u.mu.RLock()
	defer u.mu.RUnlock()

	balance, exists := u.balances[asset]
	if !exists {
		return NewBalance().Clone()
	}
	return balance.Clone()
}

func (u *User) getOrCreateBalance(asset Asset) *Balance {
	balance, exist := u.balances[asset]
	if !exist {
		balance = NewBalance()
		u.balances[asset] = balance
	}
	balance.normalize()
	return balance
}

func (u *User) SetBalance(asset Asset, balance *Balance) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if balance == nil {
		return errors.New("balance is nil")
	}

	cloned := balance.Clone()
	u.balances[asset] = &cloned
	return nil
}

func (u *User) Lock(asset Asset, amount int64) error {
	return u.LockAmount(asset, big.NewInt(amount))
}

func (u *User) LockAmount(asset Asset, amount *big.Int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount == nil || amount.Sign() <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)
	if userBalance.availableLessThan(amount) {
		return errors.New("insufficient balance")
	}

	userBalance.addLocked(amount)
	userBalance.subAvailable(amount)
	return nil
}

func (u *User) Unlock(asset Asset, amount int64) error {
	return u.UnlockAmount(asset, big.NewInt(amount))
}

func (u *User) UnlockAmount(asset Asset, amount *big.Int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount == nil || amount.Sign() <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)
	if userBalance.lockedLessThan(amount) {
		return errors.New("insufficient locked balance")
	}

	userBalance.subLocked(amount)
	userBalance.addAvailable(amount)
	return nil
}

func (u *User) Credit(asset Asset, amount int64) error {
	return u.DepositAmount(asset, big.NewInt(amount))
}

func (u *User) Debit(asset Asset, amount int64) error {
	return u.DebitAmount(asset, big.NewInt(amount))
}

func (u *User) DebitAmount(asset Asset, amount *big.Int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount == nil || amount.Sign() <= 0 {
		return errors.New("amount must be greater than 0")
	}

	userBalance := u.getOrCreateBalance(asset)
	if userBalance.lockedLessThan(amount) {
		return errors.New("insufficient locked balance")
	}

	userBalance.subLocked(amount)
	return nil
}

func (u *User) UndoDebit(asset Asset, amount int64) error {
	return u.UndoDebitAmount(asset, big.NewInt(amount))
}

func (u *User) UndoDebitAmount(asset Asset, amount *big.Int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if amount == nil || amount.Sign() <= 0 {
		return errors.New("amount must be greater than 0")
	}

	u.getOrCreateBalance(asset).addLocked(amount)
	return nil
}
