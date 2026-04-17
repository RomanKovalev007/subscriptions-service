package domain

import "errors"

var ErrNotFound = errors.New("not found")

type Error struct{
	Err error
	Message string
}