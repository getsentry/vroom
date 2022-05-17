package errorutil

import "errors"

// ErrDataIntegrity is a base error type to use for failures that are due to
// unrecoverable data integrity issues.
var ErrDataIntegrity = errors.New("data integrity error")

// ErrNoResults represents situations in which no results were returned by the called API.
var ErrNoResults = errors.New("no results returned")
