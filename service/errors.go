package service

import "errors"

// Common service errors
var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidPackage    = errors.New("invalid package")
)
