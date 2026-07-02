package main

import "errors"

var ErrMissingCmd = errors.New("missing command")

var ErrUnknownCmd = errors.New("unknown command")

var ErrUnclosedQuotes = errors.New("missing closing quotation")
