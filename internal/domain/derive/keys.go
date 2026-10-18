package derive

import (
	"strconv"
	"strings"
)

// compareKeys orders keys as the outputs list them: goals, requirements,
// criteria, questions, then anything else (source paths); within a kind by
// their numbers, so GOAL-2 comes before GOAL-10 and AC-002-9 before
// AC-002-10.
func compareKeys(a, b string) int {
	ka, na := splitKey(a)
	kb, nb := splitKey(b)
	if ka != kb {
		return ka - kb
	}
	for i := 0; i < len(na) && i < len(nb); i++ {
		if na[i] != nb[i] {
			if na[i] < nb[i] {
				return -1
			}
			return 1
		}
	}
	if len(na) != len(nb) {
		return len(na) - len(nb)
	}
	return strings.Compare(a, b)
}

var keyKinds = []string{"GOAL", "REQ", "AC", "Q"}

// splitKey returns a key's kind rank and numbers; anything that is not a
// key ranks last and has no numbers.
func splitKey(key string) (int, []int) {
	parts := strings.Split(key, "-")
	for rank, kind := range keyKinds {
		if parts[0] != kind || len(parts) < 2 {
			continue
		}
		var nums []int
		for _, p := range parts[1:] {
			n, err := strconv.Atoi(p)
			if err != nil {
				return len(keyKinds), nil
			}
			nums = append(nums, n)
		}
		return rank, nums
	}
	return len(keyKinds), nil
}
