package core

import "math/big"

type Balance struct {
	Available *big.Int
	Locked    *big.Int
}

func NewBalance() *Balance {
	return &Balance{
		Available: new(big.Int),
		Locked:    new(big.Int),
	}
}

func (b *Balance) normalize() {
	if b.Available == nil {
		b.Available = new(big.Int)
	}
	if b.Locked == nil {
		b.Locked = new(big.Int)
	}
}

func (b Balance) Clone() Balance {
	b.normalize()
	return Balance{
		Available: new(big.Int).Set(b.Available),
		Locked:    new(big.Int).Set(b.Locked),
	}
}

func (b *Balance) addAvailable(amount *big.Int) {
	b.normalize()
	b.Available = new(big.Int).Add(b.Available, amount)
}

func (b *Balance) subAvailable(amount *big.Int) {
	b.normalize()
	b.Available = new(big.Int).Sub(b.Available, amount)
}

func (b *Balance) addLocked(amount *big.Int) {
	b.normalize()
	b.Locked = new(big.Int).Add(b.Locked, amount)
}

func (b *Balance) subLocked(amount *big.Int) {
	b.normalize()
	b.Locked = new(big.Int).Sub(b.Locked, amount)
}

func (b *Balance) availableLessThan(amount *big.Int) bool {
	b.normalize()
	return b.Available.Cmp(amount) < 0
}

func (b *Balance) lockedLessThan(amount *big.Int) bool {
	b.normalize()
	return b.Locked.Cmp(amount) < 0
}
