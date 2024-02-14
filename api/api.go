package api

import "errors"

// ErrNotFound - Error to return when something is not found
var ErrNotFound = errors.New("Not Found")

// ErrUnknown - Error to return when an unknown server error occurs
var ErrUnknown = errors.New("An unknown server error occurred, please try again")

var ErrNoAuthToken = errors.New("No access token available. Please login with 'flyctl auth login'")
