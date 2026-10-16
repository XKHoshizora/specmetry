package cli

import (
	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/apperr"
	"github.com/XKHoshizora/specmetry/internal/channel"
)

// confirm has a person confirm a human-only operation (channel.Guard), on
// the stream the command's output allows (architecture §5.1, §9.1). Every
// human-only command confirms through it, never through channel.Guard
// itself (TestHumanOnlyCommandsConfirmThroughConfirm).
//
// Without --json the confirmation is shown on stdout, as it always was.
// With --json, stdout holds only what the command outputs (its JSON
// document, its text on success, or error.v1), so the confirmation is
// shown on stderr instead (codex-review-031 N2). A person must be able to
// read it there: when stderr is not a terminal, confirm refuses with
// environment (exit code 6) before anything is read, as Guard does for a
// confirmation it cannot show whole. The Agent channel is refused first,
// with human_required (exit code 3) and no prompt. Guard's own checks (the
// whole prompt written, the exact confirmation text) apply either way.
func confirm(cmd *cobra.Command, d channel.Detection, op channel.Operation) error {
	out := cmd.OutOrStdout()
	if wantsJSON(cmd) {
		if err := d.RequireHuman(op.Name); err != nil {
			return err
		}
		if !d.StderrTerminal {
			return apperr.New(apperr.Environment, op.Name+" needs a confirmation, which with --json is shown on stderr, and stderr is not a terminal;"+
				" nothing was changed (run it with stderr at the terminal, or without --json)")
		}
		out = cmd.ErrOrStderr()
	}
	return channel.Guard(d, cmd.InOrStdin(), out, op)
}
