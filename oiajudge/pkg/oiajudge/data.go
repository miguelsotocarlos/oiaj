package oiajudge

import (
	"fmt"
)

type Id = int64

// Tokens are of the form <token_id>:<token_secret>
// the secret is base64-encoded
type Token string

type OiaError struct {
	HttpCode      int
	Message       string
	InternalError error
}

func (oe *OiaError) Error() string {
	if oe.InternalError != nil {
		return fmt.Sprintf("%s: %s", oe.Message, oe.InternalError)
	} else {
		return oe.Message
	}
}

type Authenticatable interface {
	Uid() Id
}

type Config struct {
	OiaDbConnectionString string
	OiaServerPort         int64
}
