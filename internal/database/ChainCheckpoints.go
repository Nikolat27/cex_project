package database

import (
	"database/sql"
	"context"
	"cex_project/internal/utils"
)

type ChainCheckpointsModel struct {
	db *sql.DB
}

type ChainCheckpoint struct {
	Network string
	BlockNumber int64
}

func NewChainCheckpointsModel(db *sql.DB) *ChainCheckpointsModel {
	return &ChainCheckpointsModel{db: db}
}

func (m *ChainCheckpointsModel) Create(ctx context.Context, network string, blockNumber int64) error {
	query := `INSERT INTO chain_checkpoints (network, block_number) VALUES ($1, $2)`

	result, err := m.db.ExecContext(ctx, query, network, blockNumber)
	if err != nil {
		return err
	}

	return utils.RequireOneRowAffected(result)
}

func (m *ChainCheckpointsModel) Get(ctx context.Context, network string) (*ChainCheckpoint, error) {
	query := `SELECT network, block_number FROM chain_checkpoints WHERE network = $1`

	row := m.db.QueryRowContext(ctx, query, network)
	var checkpoint ChainCheckpoint

	if err := row.Scan(&checkpoint.Network, &checkpoint.BlockNumber); err != nil {
		return nil, err
	}

	return &checkpoint, nil
}

func (m *ChainCheckpointsModel) Update(ctx context.Context, network string, blockNumber int64) error {
	query := `UPDATE chain_checkpoints SET block_number = $1 WHERE network = $2`

	result, err := m.db.ExecContext(ctx, query, blockNumber, network)
	if err != nil {
		return err
	}
	
	return utils.RequireOneRowAffected(result)
}
