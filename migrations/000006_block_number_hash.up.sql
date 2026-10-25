ALTER TABLE chain_deposits
ADD COLUMN block_number BIGINT,
ADD COLUMN block_hash TEXT;