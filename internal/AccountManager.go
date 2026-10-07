package internal

import (
	"sync"
	"errors"
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
