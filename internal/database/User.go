package database

import (
	"context"
	"database/sql"
)

type UserModel struct {
	db *sql.DB
}

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{db: db}
}

func (um *UserModel) Create(ctx context.Context, username, passwordHash string) (*User, error) {
	query := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id`

	rows, err := um.db.QueryContext(ctx, query, username, passwordHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var user User

	if err := rows.Scan(&user.ID); err != nil {
		return nil, err
	}

	return &user, nil
}

func (um *UserModel) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `
	 	SELECT id, username, password_hash FROM users WHERE username = $1
	`

	var user User

	if err := um.db.QueryRowContext(ctx, query, username).Scan(&user.ID,
		 &user.Username, &user.PasswordHash); err != nil {
			return nil, err
	}

	return &user, nil
}

func (um *UserModel) GetAll(ctx context.Context) ([]User, error) {
	query := `
	 	SELECT id FROM users
	`

	rows, err := um.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	defer rows.Close()
	users := []User{}

	for rows.Next() {
		var user User

		if err := rows.Scan(&user.ID); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

func (um *UserModel) GetByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id FROM users WHERE id = $1`

	var user User
	if err := um.db.QueryRowContext(ctx, query, id).Scan(&user.ID); err != nil {
		return nil, err
	}

	return &user, nil
}

func (um *UserModel) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := um.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}
