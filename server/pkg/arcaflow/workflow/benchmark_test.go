package workflow

import (
	"context"
	"path/filepath"
	"testing"
)

func BenchmarkGenerateInputFile(b *testing.B) {
	workflowPath := filepath.Join(
		"..",
		"..",
		"..",
		"..",
		"test",
		"fixtures",
		"workflow-basic.yaml",
	)
	loader := NewLoader()
	index, err := loader.LoadFromFilesystem(context.Background(), workflowPath)
	if err != nil {
		b.Fatalf("load workflow fixture: %v", err)
	}
	parser := NewParser()
	parsed, err := parser.Parse(context.Background(), index.Workflows[0])
	if err != nil {
		b.Fatalf("parse workflow fixture: %v", err)
	}
	payload := []byte(`{"nickname":"benchmark"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := GenerateInputFile(
			context.Background(),
			parsed,
			payload,
			ExportFormatJSON,
		); err != nil {
			b.Fatalf("generate input: %v", err)
		}
	}
}
