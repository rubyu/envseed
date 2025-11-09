package renderer_test

import "testing"

// [EVT-MPF-1][EVT-MPF-4]
func FuzzRenderTemplateRoundTrip(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed int64, iteration uint32) {
		iter := clampFuzzIteration(iteration)
		meta, tpl := generatePropertyCase(t, seed, iter)
		propertyIterationRoundTrip(t, nil, meta, tpl)
	})
}

// [EVT-MPF-1]
func FuzzRenderTemplateSimple(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed int64, iteration uint32) {
		iter := clampFuzzIteration(iteration)
		meta, tpl := generatePropertyCase(t, seed, iter)
		propertyIterationSimple(t, meta, tpl)
	})
}
