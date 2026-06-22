package main

import "errors"

var ErrMissingCmd = errors.New("missing command")

var ErrUnknownCmd = errors.New("unknown command")
