package task

import "errors"

var ErrNotFound = errors.New("task not found")
var ErrAlreadyExists = errors.New("task already exists")
