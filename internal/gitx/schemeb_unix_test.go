//go:build !windows

package gitx

import (
	"os"
	"path/filepath"
	"testing"
)

// These schedules retain Review 060's original final-window counterexamples.
// The owner explicitly excluded this uncooperative same-user intervention;
// passing this test records the limitation, NOT a claim the code prevents it.
func TestApprovedUnixFinalWindow(t *testing.T) {
	requireSafeGit(t)
	for _, which := range []string{"owner", "directory"} {
		t.Run(which, func(t *testing.T) {
			s, _ := scratchIn(t)
			var original, replacement string
			testHookUnlink = func(path string, dir bool) {
				if (which == "owner" && filepath.Base(path) != ownerName) || (which == "directory" && (!dir || !privateName.MatchString(filepath.Base(path)))) {
					return
				}
				testHookUnlink = nil
				replacement = path
				original = filepath.Join(filepath.Dir(s.path), "saved-original-"+which)
				if err := os.Rename(path, original); err != nil {
					t.Fatal(err)
				}
				if dir {
					mkPrivate(t, path)
				} else {
					write(t, path, "replacement-owner")
				}
			}
			t.Cleanup(func() { testHookUnlink = nil })
			got, err := Clean(ctx, s, baseRepo(t).Dir)
			if err != nil || got != StateClean {
				t.Fatalf("excluded schedule: %v %v", got, err)
			}
			if replacement == "" {
				t.Fatal("the final window was not reached")
			}
			if _, err := os.Stat(replacement); !os.IsNotExist(err) {
				t.Fatalf("replacement unexpectedly retained: %v", err)
			}
			if _, err := os.Stat(original); err != nil {
				t.Fatalf("original lost: %v", err)
			}
			t.Log("final unlink/rmdir can remove the excluded same-user replacement; retained as an approved limitation")
		})
	}
}
