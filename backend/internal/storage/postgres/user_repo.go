package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	FirstName    string
	SecondName   string
	Biography    sql.NullString
	Birthdate    sql.NullTime
	City         sql.NullString
	Gender       sql.NullString
	PasswordHash string
}

type CreateUserParams struct {
	FirstName    string
	SecondName   string
	Biography    string
	Birthdate    string
	City         string
	Gender       string
	PasswordHash string
}

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

const insertUser = `
	INSERT INTO users (first_name, second_name, biography, birthdate, city, gender, password_hash)
	VALUES ($1, $2, $3, NULLIF($4, '')::date, $5, NULLIF($6, '')::gender_type, $7)
	RETURNING id`

func (r *UserRepo) Create(ctx context.Context, u CreateUserParams) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, insertUser,
		u.FirstName, u.SecondName, u.Biography, u.Birthdate, u.City, u.Gender, u.PasswordHash,
	).Scan(&id)
	return id, err
}

const getUser = `
	SELECT id, first_name, second_name, biography, birthdate, city, gender
	FROM users WHERE id = $1`

func (r *UserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.db.QueryRowContext(ctx, getUser, id).Scan(
		&u.ID, &u.FirstName, &u.SecondName, &u.Biography, &u.Birthdate, &u.City, &u.Gender,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

const getPasswordHash = `SELECT password_hash FROM users WHERE id = $1`

func (r *UserRepo) GetPasswordHash(ctx context.Context, id uuid.UUID) (string, error) {
	var hash string
	err := r.db.QueryRowContext(ctx, getPasswordHash, id).Scan(&hash)
	return hash, err
}

func NullTimeToDateString(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.DateOnly)
}

const searchUsers = `
SELECT id, first_name, second_name, biography, birthdate, city, gender
FROM users
WHERE first_name LIKE $1 AND second_name LIKE $2`

func (r *UserRepo) Search(ctx context.Context, firstName, secondName string) ([]*User, error) {
	rows, err := r.db.QueryContext(ctx, searchUsers, firstName+"%", secondName+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []*User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.FirstName, &u.SecondName, &u.Biography, &u.Birthdate, &u.City, &u.Gender); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}
