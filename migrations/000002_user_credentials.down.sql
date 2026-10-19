ALTER TABLE users DROP CONSTRAINT users_username_key;
ALTER TABLE users DROP COLUMN username;
ALTER TABLE users DROP COLUMN password_hash;
