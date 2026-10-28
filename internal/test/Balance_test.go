package test

import (
	"math/big"
	"sync"
	"testing"

	"cex_project/internal/core"
)

func TestUserDepositAndWithdraw(t *testing.T) {
	user := core.NewUser(1)

	if err := user.Deposit(core.USDT, 100); err != nil {
		t.Fatalf("unexpected deposit error: %v", err)
	}
	if err := user.Withdraw(core.USDT, 30); err != nil {
		t.Fatalf("unexpected withdrawal error: %v", err)
	}

	assertBalance(t, user, core.USDT, 70, 0)
}

func TestUserLockAndUnlock(t *testing.T) {
	user := core.NewUser(1)
	if err := user.Deposit(core.USDT, 100); err != nil {
		t.Fatalf("unexpected deposit error: %v", err)
	}

	if err := user.Lock(core.USDT, 60); err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}
	assertBalance(t, user, core.USDT, 40, 60)

	if err := user.Unlock(core.USDT, 20); err != nil {
		t.Fatalf("unexpected unlock error: %v", err)
	}
	assertBalance(t, user, core.USDT, 60, 40)
}

func TestUserDebit(t *testing.T) {
	user := core.NewUser(1)
	if err := user.Deposit(core.USDT, 100); err != nil {
		t.Fatalf("unexpected deposit error: %v", err)
	}
	if err := user.Lock(core.USDT, 60); err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}

	if err := user.Debit(core.USDT, 25); err != nil {
		t.Fatalf("unexpected debit error: %v", err)
	}

	assertBalance(t, user, core.USDT, 40, 35)
}

func TestUserCredit(t *testing.T) {
	user := core.NewUser(1)

	if err := user.Credit(core.BTC, 15); err != nil {
		t.Fatalf("unexpected credit error: %v", err)
	}

	assertBalance(t, user, core.BTC, 15, 0)
}

func TestUserRejectsNonPositiveAmounts(t *testing.T) {
	tests := []struct {
		name      string
		operation func(*core.User) error
	}{
		{"deposit zero", func(user *core.User) error {
			return user.Deposit(core.USDT, 0)
		}},
		{"withdraw negative", func(user *core.User) error {
			return user.Withdraw(core.USDT, -1)
		}},
		{"lock zero", func(user *core.User) error {
			return user.Lock(core.USDT, 0)
		}},
		{"unlock negative", func(user *core.User) error {
			return user.Unlock(core.USDT, -1)
		}},
		{"debit locked zero", func(user *core.User) error {
			return user.Debit(core.USDT, 0)
		}},
		{"credit negative", func(user *core.User) error {
			return user.Credit(core.USDT, -1)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := core.NewUser(1)
			if err := test.operation(user); err == nil {
				t.Fatal("expected an error")
			}
			assertBalance(t, user, core.USDT, 0, 0)
		})
	}
}

func TestUserRejectsInsufficientAvailableBalance(t *testing.T) {
	tests := []struct {
		name      string
		operation func(*core.User) error
	}{
		{"withdraw", func(user *core.User) error {
			return user.Withdraw(core.USDT, 51)
		}},
		{"lock", func(user *core.User) error {
			return user.Lock(core.USDT, 51)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := core.NewUser(1)
			if err := user.Deposit(core.USDT, 50); err != nil {
				t.Fatalf("unexpected deposit error: %v", err)
			}

			if err := test.operation(user); err == nil {
				t.Fatal("expected an insufficient balance error")
			}
			assertBalance(t, user, core.USDT, 50, 0)
		})
	}
}

func TestUserRejectsInsufficientLockedBalance(t *testing.T) {
	tests := []struct {
		name      string
		operation func(*core.User) error
	}{
		{"unlock", func(user *core.User) error {
			return user.Unlock(core.USDT, 31)
		}},
		{"debit locked", func(user *core.User) error {
			return user.Debit(core.USDT, 31)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := core.NewUser(1)
			if err := user.Deposit(core.USDT, 50); err != nil {
				t.Fatalf("unexpected deposit error: %v", err)
			}
			if err := user.Lock(core.USDT, 30); err != nil {
				t.Fatalf("unexpected lock error: %v", err)
			}

			if err := test.operation(user); err == nil {
				t.Fatal("expected an insufficient locked balance error")
			}
			assertBalance(t, user, core.USDT, 20, 30)
		})
	}
}

func TestUserGetMissingBalanceReturnsZero(t *testing.T) {
	user := core.NewUser(1)

	assertBalance(t, user, core.BTC, 0, 0)
}

func TestUserConcurrentDeposits(t *testing.T) {
	const (
		depositCount = 200
		amount       = 5
	)

	user := core.NewUser(1)
	var wg sync.WaitGroup
	errs := make(chan error, depositCount)

	for i := 0; i < depositCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := user.Deposit(core.USDT, amount); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected deposit error: %v", err)
	}

	assertBalance(t, user, core.USDT, depositCount*amount, 0)
}

func assertBalance(t *testing.T, user *core.User, asset core.Asset, available, locked int64) {
	t.Helper()

	balance := user.GetBalance(asset)
	if balance.Available.Cmp(big.NewInt(available)) != 0 || balance.Locked.Cmp(big.NewInt(locked)) != 0 {
		t.Fatalf(
			"expected balance available=%d locked=%d, got available=%s locked=%s",
			available,
			locked,
			balance.Available,
			balance.Locked,
		)
	}
}
