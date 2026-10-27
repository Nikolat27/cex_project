package database

import "database/sql"

type Models struct {
	User *UserModel
	Market *MarketModel
	Balance *BalanceModel
	Order *OrderModel
	Trade *TradeModel
	ExchangeFee *ExchangeFeeModel
	DepositAddress *DepositAddressModel
	ChainDeposit *ChainDepositModel
	Withdrawal *WithdrawalModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		User:   NewUserModel(db),
		Market: NewMarketModel(db),
		Balance: NewBalanceModel(db),
		Order: NewOrderModel(db),
		Trade: NewTradeModel(db),
		ExchangeFee: NewExchangeFeeModel(db),
		DepositAddress: NewDepositAddressModel(db),
		ChainDeposit: NewChainDepositModel(db),
		Withdrawal: NewWithdrawalModel(db),
	}
}
