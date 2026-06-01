package service

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidValue       = errors.New("invalid value")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
