package database

import "database/sql"

type Models struct {
	User *UserModel
	Market *MarketModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		User:   NewUserModel(db),
		Market: NewMarketModel(db),
	}
}
