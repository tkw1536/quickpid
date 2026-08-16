//spellchecker:words service
package service

//spellchecker:words errors
import (
	"errors"
)

var (
	errInsufficientEntropy = errors.New("insufficient entropy")
)
