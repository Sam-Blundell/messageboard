package board

import "errors"

var ErrNotFound = errors.New("board not found")

var ErrDuplicateName = errors.New("board name already exists")
