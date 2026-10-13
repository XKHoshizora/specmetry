package cli

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/buildinfo"
	"github.com/XKHoshizora/specmetry/internal/config"
	"github.com/XKHoshizora/specmetry/internal/datadir"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
)

type checkStatus string

const (
	statusOK   checkStatus = "ok"
	statusInfo checkStatus = "info"
	// statusWarn is something doctor could not verify; it does not fail.
	statusWarn checkStatus = "warn"
	statusFail checkStatus = "fail"
)

type check struct {
	status checkStatus
	name   string
	detail string
}

// newDoctorCmd is the M0 doctor framework: read-only checks of the local
// environment. It never creates, migrates, recovers or modifies anything:
// the database is read from a snapshot that leaves its files untouched. Hook,
// Skill and spool checks join in M3; `doctor --json` needs a schema first
// (implementation-plan §3).
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the local Specmetry environment (read-only)",
		Args:  cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			checks := runChecks(cmd)
			failed := printChecks(cmd.OutOrStdout(), checks)
			if failed > 0 {
				return apperr.Newf(apperr.Environment, "doctor found %d problem(s)", failed)
			}
			return nil
		}),
	}
}

func runChecks(cmd *cobra.Command) []check {
	var checks []check
	dir, err := datadir.Resolve()
	if err != nil {
		return append(checks, check{statusFail, "data directory", err.Error()})
	}
	source := "default"
	if dir.FromEnv {
		source = "from " + datadir.EnvHome
	}
	exists, err := dir.Exists()
	switch {
	case err != nil:
		checks = append(checks, check{statusFail, "data directory", err.Error()})
	case !exists:
		checks = append(checks, check{statusInfo, "data directory", fmt.Sprintf("%s (%s) does not exist yet; it is created on first use", dir.Path, source)})
	default:
		checks = append(checks, check{statusOK, "data directory", fmt.Sprintf("%s (%s)", dir.Path, source)})
		checks = append(checks, permissionCheck(dir))
		checks = append(checks, configCheck(dir))
		checks = append(checks, databaseChecks(cmd, dir)...)
	}
	return append(checks, gitCheck())
}

func permissionCheck(dir datadir.Dir) check {
	problems, err := datadir.Check(dir)
	switch {
	case err != nil:
		return check{statusFail, "permissions", err.Error()}
	case len(problems) > 0:
		return check{statusFail, "permissions", strings.Join(problems, "; ")}
	default:
		return check{statusOK, "permissions", privateDescription()}
	}
}

func privateDescription() string {
	if runtime.GOOS == "windows" {
		return "accessible only to the current user and SYSTEM"
	}
	return "accessible only to the current user"
}

func configCheck(dir datadir.Dir) check {
	cfg, found, err := config.Load(dir.Join(datadir.ConfigFile))
	switch {
	case err != nil:
		return check{statusFail, "configuration", err.Error()}
	case !found:
		return check{statusOK, "configuration", "no config.toml; defaults apply"}
	case cfg.User.DisplayName == "":
		return check{statusOK, "configuration", "config.toml is valid"}
	default:
		return check{statusOK, "configuration", "config.toml is valid (display name: " + cfg.User.DisplayName + ")"}
	}
}

func databaseChecks(cmd *cobra.Command, dir datadir.Dir) []check {
	in, err := sqlite.Inspect(cmd.Context(), dir)
	if err != nil {
		return []check{{statusFail, "database", err.Error()}}
	}
	var out []check
	compat := in.Compatibility(buildinfo.Version)
	switch {
	case !in.Exists:
		out = append(out, check{statusInfo, "database", "not created yet"})
	case in.Limitation != "":
		out = append(out, check{statusWarn, "database", "cannot be inspected without modifying it: " + in.Limitation})
	case in.SchemaVersion > in.SupportedVersion:
		out = append(out, check{statusFail, "database", fmt.Sprintf("schema v%d is newer than this binary supports (v%d); upgrade specmetry", in.SchemaVersion, in.SupportedVersion)})
	case compat != nil:
		out = append(out, check{statusFail, "database", fmt.Sprintf("schema v%d, but %s", in.SchemaVersion, compat.Error())})
	case in.SchemaVersion < in.SupportedVersion:
		out = append(out, check{statusInfo, "database", fmt.Sprintf("schema v%d; will be migrated to v%d (with a backup) on next use", in.SchemaVersion, in.SupportedVersion)})
	default:
		out = append(out, check{statusOK, "database", fmt.Sprintf("schema v%d", in.SchemaVersion)})
	}
	if in.MigrationLocked {
		out = append(out, check{statusFail, "migration lock", dir.Join(datadir.MigrateLockFile) + " exists; delete it if no specmetry process is running"})
	}
	return out
}

func gitCheck() check {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return check{statusFail, "git", "git is required but was not found on PATH"}
	}
	return check{statusOK, "git", strings.TrimSpace(string(out))}
}

func printChecks(w io.Writer, checks []check) (failed int) {
	for _, c := range checks {
		if c.status == statusFail {
			failed++
		}
		_, _ = fmt.Fprintf(w, "%-6s %-15s %s\n", "["+string(c.status)+"]", c.name, c.detail)
	}
	return failed
}
