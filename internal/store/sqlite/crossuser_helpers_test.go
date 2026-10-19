package sqlite

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/gitx"
)

// victimOpenHelper opens the store in $SPECMETRY_HOME and records $CANARY as
// a fact, as an ordinary specmetry command would. The exit code is the CLI
// exit code of the outcome.
func victimOpenHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return apperr.ExitCode(err)
	}
	migs, err := helperMigrations(false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	db, _, err := Open(context.Background(), dir, Options{Migrations: migs})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+" "+strings.Join(detailStrings(err), "; "))
		return apperr.ExitCode(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := writeFact(context.Background(), db, "01J9Z3K5Q8R2T4V6W8X0Y2Z4A6", os.Getenv("CANARY")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return apperr.ExitCode(err)
	}
	return 0
}

// victimCleanHelper reads the state of the worktree $REPO through a
// gitx.Scratch in $SPECMETRY_HOME (review 058 B1), as a command reading git
// would, and prints the state's number. The exit code is the CLI exit code
// of the outcome.
func victimCleanHelper() int {
	dir, err := datadir.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return apperr.ExitCode(err)
	}
	s, err := gitx.NewScratch(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+" "+strings.Join(detailStrings(err), "; "))
		return apperr.ExitCode(err)
	}
	defer func() { _ = s.Close() }()
	state, err := gitx.Clean(context.Background(), s, os.Getenv("REPO"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return apperr.ExitCode(err)
	}
	fmt.Println(int(state))
	return 0
}

func detailStrings(err error) []string {
	var out []string
	for _, d := range apperr.From(err).Details {
		out = append(out, d.Problem)
	}
	return out
}

// attackerHelper performs one file operation as whatever user it runs as and
// prints "ok" or the error. ATTACK_OP=swap keeps replacing $ATTACK_A with a
// symbolic link to $ATTACK_B until $ATTACK_STOP exists; ATTACK_OP=mkdir
// makes the directory $ATTACK_A.
func attackerHelper() int {
	a, b := os.Getenv("ATTACK_A"), os.Getenv("ATTACK_B")
	var err error
	switch os.Getenv("ATTACK_OP") {
	case "rename":
		err = os.Rename(a, b)
	case "read":
		_, err = os.ReadFile(a)
	case "mkdir":
		err = os.Mkdir(a, 0o700)
	case "swap":
		for {
			if _, serr := os.Stat(os.Getenv("ATTACK_STOP")); serr == nil {
				break
			}
			if fi, lerr := os.Lstat(a); lerr == nil && fi.IsDir() {
				if os.Rename(a, a+".moved") == nil {
					_ = os.Symlink(b, a)
					_ = os.RemoveAll(a + ".moved")
				}
			}
			time.Sleep(200 * time.Microsecond)
		}
	default:
		err = fmt.Errorf("unknown ATTACK_OP %q", os.Getenv("ATTACK_OP"))
	}
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("ok")
	}
	return 0
}
