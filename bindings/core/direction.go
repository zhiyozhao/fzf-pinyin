package core

// matchingPolicy follows src/core.go's reverse traversal of effective sort
// criteria (score is always first). The highest-priority direction criterion
// wins; chunk requests positions without changing direction. These are the
// schemes' default criteria. Custom --tiebreak/--tac/--nth are not binding API
// options: if added later they must supply their effective criteria here.
func matchingPolicy(criteria []string) (forward, withPositions bool) {
	forward = true
	for i := len(criteria) - 1; i > 0; i-- {
		switch criteria[i] {
		case "chunk":
			withPositions = true
		case "end":
			forward = false
		case "begin":
			forward = true
		case "pathname":
			withPositions = true
			forward = false
		}
	}
	return
}

func schemeMatchingPolicy(scheme string) (bool, bool) {
	switch scheme {
	case "path":
		return matchingPolicy([]string{"score", "pathname", "length"})
	case "history":
		return matchingPolicy([]string{"score"})
	default:
		return matchingPolicy([]string{"score", "length"})
	}
}
