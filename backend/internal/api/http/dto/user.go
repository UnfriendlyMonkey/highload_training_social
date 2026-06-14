package dto

import "github.com/UnfriendlyMonkey/hsn/internal/storage/postgres"

type RegisterRequest struct {
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
	Birthdate  string `json:"birthdate"`
	Biography  string `json:"biography"`
	City       string `json:"city"`
	Password   string `json:"password"`
}

type RegisterResponse struct {
	UserID string `json:"user_id"`
}

type UserResponse struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
	Birthdate  string `json:"birthdate"`
	Biography  string `json:"biography"`
	City       string `json:"city"`
}

func (r *UserResponse) FromEntity(user *postgres.User) *UserResponse {
	r.ID = user.ID.String()
	r.FirstName = user.FirstName
	r.SecondName = user.SecondName
	r.Birthdate = postgres.NullTimeToDateString(user.Birthdate)
	if user.Biography.Valid {
		r.Biography = user.Biography.String
	}
	if user.City.Valid {
		r.City = user.City.String
	}
	return r
}

func UserResponsesFromEntities(users []*postgres.User) []*UserResponse {
	resp := make([]*UserResponse, 0, len(users))
	for _, user := range users {
		resp = append(resp, new(UserResponse).FromEntity(user))
	}
	return resp
}
