package renderer_test

import "testing"

// [EVT-MPF-3][EVT-MPF-4]
func FuzzRenderTemplateEnhanced(f *testing.F) {
	seeds := []struct {
		seed      int64
		iteration uint32
	}{
		{seed: 4242424242, iteration: 0},
		{seed: -1123581321, iteration: 15},
		{seed: 662607015, iteration: 47},
	}
	for _, s := range seeds {
		f.Add(s.seed, s.iteration)
	}
	f.Fuzz(func(t *testing.T, seed int64, iteration uint32) {
		iter := clampFuzzIteration(iteration)
		meta, tpl := generateEnhancedCase(t, seed, iter, modeEnhancedTemplates)
		enhancedIteration(t, nil, meta, tpl)
	})
}
