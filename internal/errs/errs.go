package errs

import "errors"

var (
	ErrNotFound = errors.New("not found")
	ErrBadData  = errors.New("bad data")
	ErrNoData   = errors.New("no data")
)
