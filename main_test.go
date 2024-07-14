package main

import "testing"

func TestRun(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		run()
	})
}

func BenchmarkRun(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		run()
	}
}
