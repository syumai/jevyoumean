// Package fallback provides offline suggestions based on edit distance,
// used when the Jev API key is missing or the request fails. Candidates
// are shown without probabilities and marked "(offline)".
package fallback

import (
	"sort"
	"strings"

	"github.com/agnivade/levenshtein"

	"github.com/syumai/jevyoumean/internal/helptext"
)

// Match is an edit-distance candidate.
type Match struct {
	Name     string
	Distance int
}

// Suggest returns candidate names within edit distance 2 or one third
// of the typed length, plus prefix matches, sorted by distance.
// At most max matches are returned.
func Suggest(typed string, commands []helptext.Command, max int) []Match {
	var matches []Match
	for _, c := range commands {
		for _, name := range append([]string{c.Name}, c.Aliases...) {
			d := levenshtein.ComputeDistance(typed, name)
			if d <= 2 || d <= len(typed)/3 || strings.HasPrefix(name, typed) {
				matches = append(matches, Match{Name: c.Name, Distance: d})
				break
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Distance != matches[j].Distance {
			return matches[i].Distance < matches[j].Distance
		}
		return matches[i].Name < matches[j].Name
	})
	if len(matches) > max {
		matches = matches[:max]
	}
	return matches
}
