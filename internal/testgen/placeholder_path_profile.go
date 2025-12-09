package testgen

import (
	"fmt"
	"math/rand"
)

// PlaceholderPathProfile generates simple templates focusing on PATH IRI
// behavior for placeholders. It alternates between valid and invalid PATH
// values so parser tests can assert EVE-103-206.
type PlaceholderPathProfile struct{}

// Generate implements Profile.
func (p *PlaceholderPathProfile) Generate(r *rand.Rand, _ uint32) Case {
	// Valid PATHs: authority + path without query/fragment components.
	validPaths := []string{
		"host",
		"example.com/service",
		"service/api-token",
		"日本語/鍵",
	}
	// Invalid PATHs for EnvSeed: introduce unencoded '?' or '#'.
	invalidPaths := []string{
		"host?query",
		"host#fragment",
	}

	useValid := r.Intn(2) == 0
	if useValid {
		path := validPaths[r.Intn(len(validPaths))]
		tpl := fmt.Sprintf("VAR=<pass://%s>\n", path)
		return Case{
			Template: tpl,
			Resolver: map[string]string{},
			Expect:   Expectation{},
		}
	}

	path := invalidPaths[r.Intn(len(invalidPaths))]
	tpl := fmt.Sprintf("VAR=<pass://%s>\n", path)
	return Case{
		Template: tpl,
		Resolver: map[string]string{},
		Expect: Expectation{
			ShouldErr:  true,
			DetailCode: "EVE-103-206",
			Phase:      "parse",
		},
	}
}
