ALTER TABLE balances
    ALTER COLUMN available TYPE BIGINT USING available::bigint,
    ALTER COLUMN locked TYPE BIGINT USING locked::bigint;
