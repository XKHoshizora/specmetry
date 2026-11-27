//go:build !unix

package pseudo

import "errors"

func raiseTerm() error { return errors.New("pseudo: no signals on this platform") }
