package test

import (
	"sync"
	"testing"

	"cex_project/internal"
)

func TestUserDepositAndWithdraw(t *testing.T) {
	user := internal.NewUser(1)

	if err := user.Deposit(internal.USDT, 100); err != nil {
		t.Fatalf("unexpected deposit error: %v", err)
	}
	if err := user.Withdraw(internal.USDT, 30); err != nil {
		t.Fatalf("unexpected withdrawal error: %v", err)
	}

	assertBalance(t, user, internal.USDT, 70, 0)
}

func TestUserLockAndUnlock(t *testing.T) {
	user := internal.NewUser(1)
	if err := user.Deposit(internal.USDT, 100); err != nil {
		t.Fatalf("unexpected deposit error: %v", err)
	}

	if err := user.Lock(internal.USDT, 60); err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}
	assertBalance(t, user, internal.USDT, 40, 60)

	if err := user.Unlock(internal.USDT, 20); err != nil {
		t.Fatalf("unexpected unlock error: %v", err)
	}
	assertBalance(t, user, internal.USDT, 60, 40)
}

func TestUserDebit(t *testing.T) {
	user := internal.NewUser(1)
	if err := user.Deposit(internal.USDT, 100); err != nil {
		t.Fatalf("unexpected deposit error: %v", err)
	}
	if err := user.Lock(internal.USDT, 60); err != nil {
		t.Fatalf("unexpected lock error: %v", err)
	}

	if err := user.Debit(internal.USDT, 25); err != nil {
		t.Fatalf("unexpected debit error: %v", err)
	}

	assertBalance(t, user, internal.USDT, 40, 35)
}

func TestUserCredit(t *testing.T) {
	user := internal.NewUser(1)

	if err := user.Credit(internal.BTC, 15); err != nil {
		t.Fatalf("unexpected credit error: %v", err)
	}

	assertBalance(t, user, internal.BTC, 15, 0)
}

func TestUserRejectsNonPositiveAmounts(t *testing.T) {
	tests := []struct {
		name      string
		operation func(*internal.User) error
	}{
		{"deposit zero", func(user *internal.User) error {
			return user.Deposit(internal.USDT, 0)
		}},
		{"withdraw negative", func(user *internal.User) error {
			return user.Withdraw(internal.USDT, -1)
		}},
		{"lock zero", func(user *internal.User) error {
			return user.Lock(internal.USDT, 0)
		}},
		{"unlock negative", func(user *internal.User) error {
			return user.Unlock(internal.USDT, -1)
		}},
		{"debit locked zero", func(user *internal.User) error {
			return user.Debit(internal.USDT, 0)
		}},
		{"credit negative", func(user *internal.User) error {
			return user.Credit(internal.USDT, -1)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := internal.NewUser(1)
			if err := test.operation(user); err == nil {
				t.Fatal("expected an error")
			}
			assertBalance(t, user, internal.USDT, 0, 0)
		})
	}
}

func TestUserRejectsInsufficientAvailableBalance(t *testing.T) {
	tests := []struct {
		name      string
		operation func(*internal.User) error
	}{
		{"withdraw", func(user *internal.User) error {
			return user.Withdraw(internal.USDT, 51)
		}},
		{"lock", func(user *internal.User) error {
			return user.Lock(internal.USDT, 51)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := internal.NewUser(1)
			if err := user.Deposit(internal.USDT, 50); err != nil {
				t.Fatalf("unexpected deposit error: %v", err)
			}

			if err := test.operation(user); err == nil {
				t.Fatal("expected an insufficient balance error")
			}
			assertBalance(t, user, internal.USDT, 50, 0)
		})
	}
}

func TestUserRejectsInsufficientLockedBalance(t *testing.T) {
	tests := []struct {
		name      string
		operation func(*internal.User) error
	}{
		{"unlock", func(user *internal.User) error {
			return user.Unlock(internal.USDT, 31)
		}},
		{"debit locked", func(user *internal.User) error {
			return user.Debit(internal.USDT, 31)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := internal.NewUser(1)
			if err := user.Deposit(internal.USDT, 50); err != nil {
				t.Fatalf("unexpected deposit error: %v", err)
			}
			if err := user.Lock(internal.USDT, 30); err != nil {
				t.Fatalf("unexpected lock error: %v", err)
			}

			if err := test.operation(user); err == nil {
				t.Fatal("expected an insufficient locked balance error")
			}
			assertBalance(t, user, internal.USDT, 20, 30)
		})
	}
}

func TestUserGetMissingBalanceReturnsZero(t *testing.T) {
	user := internal.NewUser(1)

	assertBalance(t, user, internal.BTC, 0, 0)
}

func TestUserConcurrentDeposits(t *testing.T) {
	const (
		depositCount = 200
		amount       = 5
	)

	user := internal.NewUser(1)
	var wg sync.WaitGroup
	errs := make(chan error, depositCount)

	for i := 0; i < depositCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := user.Deposit(internal.USDT, amount); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected deposit error: %v", err)
	}

	assertBalance(t, user, internal.USDT, depositCount*amount, 0)
}

func assertBalance(t *testing.T, user *internal.User, asset internal.Asset, available, locked int64) {
	t.Helper()

	balance := user.GetBalance(asset)
	if balance.Available != available || balance.Locked != locked {
		t.Fatalf(
			"expected balance available=%d locked=%d, got available=%d locked=%d",
			available,
			locked,
			balance.Available,
			balance.Locked,
		)
	}
}
