package wp25

import "os"

// runHelperP runs the helper modes of P-2 and P-4
// (docs/wp2.5-prototype.md §14).
func runHelperP(mode string, args []string) (int, bool) {
	switch mode {
	case "marker-exit":
		// P-4's user code: it leaves a marker and exits 0.
		if os.WriteFile(os.Getenv(markerEnv), []byte("ran\n"), 0o600) != nil {
			return exitWrite, true
		}
		return 0, true
	}
	return p2Helper(mode, args)
}
