ALTER TABLE balances
    ALTER COLUMN available TYPE NUMERIC USING available::numeric,
    ALTER COLUMN locked TYPE NUMERIC USING locked::numeric;
