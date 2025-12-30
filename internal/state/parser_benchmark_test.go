package state

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// generateStateFile creates a synthetic Terraform state file of the specified size
// by adding multiple outputs with realistic structure.
// targetSize is approximate in bytes.
func generateStateFile(targetSize int) []byte {
	// Base structure overhead
	baseState := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.5.0",
		"serial":            1,
		"lineage":           "abc-123-def-456-ghi-789",
		"outputs":           make(map[string]interface{}),
	}

	// Calculate approximate bytes needed per output
	// Each output has: name, value, type, sensitive fields
	sampleOutput := map[string]interface{}{
		"value":     "vpc-1234567890abcdef",
		"type":      "string",
		"sensitive": false,
	}
	sampleJSON, _ := json.Marshal(sampleOutput)
	bytesPerOutput := len(sampleJSON) + 30 // Add overhead for key and formatting

	// Calculate how many outputs we need
	numOutputs := targetSize / bytesPerOutput
	if numOutputs < 1 {
		numOutputs = 1
	}

	// Generate outputs
	outputs := make(map[string]interface{})
	for i := 0; i < numOutputs; i++ {
		// Vary output types for realism
		var output map[string]interface{}
		switch i % 5 {
		case 0:
			// String output
			output = map[string]interface{}{
				"value":     fmt.Sprintf("resource-id-%08d", i),
				"type":      "string",
				"sensitive": false,
			}
		case 1:
			// Number output
			output = map[string]interface{}{
				"value":     float64(i),
				"type":      "number",
				"sensitive": false,
			}
		case 2:
			// Boolean output
			output = map[string]interface{}{
				"value":     i%2 == 0,
				"type":      "bool",
				"sensitive": false,
			}
		case 3:
			// List output
			output = map[string]interface{}{
				"value": []string{
					fmt.Sprintf("item-%d-a", i),
					fmt.Sprintf("item-%d-b", i),
					fmt.Sprintf("item-%d-c", i),
				},
				"type":      "list(string)",
				"sensitive": false,
			}
		case 4:
			// Map output
			output = map[string]interface{}{
				"value": map[string]interface{}{
					"name":        fmt.Sprintf("resource-%d", i),
					"environment": "production",
					"region":      "us-east-1",
				},
				"type":      "map(string)",
				"sensitive": i%10 == 0, // 10% sensitive
			}
		}
		outputs[fmt.Sprintf("output_%08d", i)] = output
	}

	baseState["outputs"] = outputs
	data, _ := json.Marshal(baseState)
	return data
}

// generateNestedStateFile creates a state file with deeply nested object structures
// to test parser performance with complex JSON.
func generateNestedStateFile(targetSize int) []byte {
	baseState := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.5.0",
		"serial":            1,
		"lineage":           "abc-123-def-456-ghi-789",
		"outputs":           make(map[string]interface{}),
	}

	// Create nested structure
	outputs := make(map[string]interface{})
	nestedCount := 0

	for len(outputs) < 100 && nestedCount*500 < targetSize {
		nested := map[string]interface{}{
			"vpc": map[string]interface{}{
				"id":   fmt.Sprintf("vpc-%d", nestedCount),
				"cidr": "10.0.0.0/16",
				"subnets": []interface{}{
					map[string]interface{}{
						"id":   fmt.Sprintf("subnet-%d-a", nestedCount),
						"cidr": "10.0.1.0/24",
						"az":   "us-east-1a",
					},
					map[string]interface{}{
						"id":   fmt.Sprintf("subnet-%d-b", nestedCount),
						"cidr": "10.0.2.0/24",
						"az":   "us-east-1b",
					},
				},
				"tags": map[string]interface{}{
					"Name":        fmt.Sprintf("vpc-%d", nestedCount),
					"Environment": "production",
					"ManagedBy":   "terraform",
				},
			},
		}

		outputs[fmt.Sprintf("network_config_%d", nestedCount)] = map[string]interface{}{
			"value":     nested,
			"type":      "object({vpc=object({...})})",
			"sensitive": false,
		}
		nestedCount++
	}

	baseState["outputs"] = outputs
	data, _ := json.Marshal(baseState)
	return data
}

// BenchmarkParseStateFile_1KB tests parsing performance for small state files (~1KB)
func BenchmarkParseStateFile_1KB(b *testing.B) {
	data := generateStateFile(1024) // 1KB
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_10KB tests parsing performance for typical state files (~10KB)
func BenchmarkParseStateFile_10KB(b *testing.B) {
	data := generateStateFile(10 * 1024) // 10KB
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_100KB tests parsing performance for medium state files (~100KB)
func BenchmarkParseStateFile_100KB(b *testing.B) {
	data := generateStateFile(100 * 1024) // 100KB
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_1MB tests parsing performance for large state files (~1MB)
func BenchmarkParseStateFile_1MB(b *testing.B) {
	data := generateStateFile(1024 * 1024) // 1MB
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_10MB tests parsing performance for very large state files (~10MB)
func BenchmarkParseStateFile_10MB(b *testing.B) {
	data := generateStateFile(10 * 1024 * 1024) // 10MB
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_Nested_1MB tests parsing performance with deeply nested structures
func BenchmarkParseStateFile_Nested_1MB(b *testing.B) {
	data := generateNestedStateFile(1024 * 1024) // 1MB nested
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_Nested_10MB tests parsing performance with deeply nested structures
func BenchmarkParseStateFile_Nested_10MB(b *testing.B) {
	data := generateNestedStateFile(10 * 1024 * 1024) // 10MB nested
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_ManySmallOutputs tests parsing with many small outputs
func BenchmarkParseStateFile_ManySmallOutputs(b *testing.B) {
	baseState := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.5.0",
		"serial":            1,
		"lineage":           "abc-123-def-456-ghi-789",
		"outputs":           make(map[string]interface{}),
	}

	outputs := make(map[string]interface{})
	// Create 10,000 small outputs
	for i := 0; i < 10000; i++ {
		outputs[fmt.Sprintf("output_%d", i)] = map[string]interface{}{
			"value":     fmt.Sprintf("val-%d", i),
			"type":      "string",
			"sensitive": false,
		}
	}
	baseState["outputs"] = outputs
	data, _ := json.Marshal(baseState)

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_FewLargeOutputs tests parsing with few but very large outputs
func BenchmarkParseStateFile_FewLargeOutputs(b *testing.B) {
	baseState := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.5.0",
		"serial":            1,
		"lineage":           "abc-123-def-456-ghi-789",
		"outputs":           make(map[string]interface{}),
	}

	outputs := make(map[string]interface{})
	// Create 10 very large outputs with long strings
	for i := 0; i < 10; i++ {
		// Generate a large string (~100KB each)
		largeString := strings.Repeat(fmt.Sprintf("data-%d-", i), 10000)
		outputs[fmt.Sprintf("large_output_%d", i)] = map[string]interface{}{
			"value":     largeString,
			"type":      "string",
			"sensitive": false,
		}
	}
	baseState["outputs"] = outputs
	data, _ := json.Marshal(baseState)

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_RealWorldMixed tests with a realistic mix of output types
func BenchmarkParseStateFile_RealWorldMixed(b *testing.B) {
	baseState := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.5.0",
		"serial":            42,
		"lineage":           "f7b3c2a1-9d8e-4f5a-b1c2-3d4e5f6a7b8c",
		"outputs":           make(map[string]interface{}),
	}

	outputs := make(map[string]interface{})

	// Simulate real-world AWS infrastructure outputs
	outputs["vpc_id"] = map[string]interface{}{
		"value":     "vpc-0a1b2c3d4e5f6g7h8",
		"type":      "string",
		"sensitive": false,
	}

	outputs["subnet_ids"] = map[string]interface{}{
		"value": []string{
			"subnet-0a1b2c3d",
			"subnet-1e2f3g4h",
			"subnet-2i3j4k5l",
		},
		"type":      "list(string)",
		"sensitive": false,
	}

	outputs["security_group_id"] = map[string]interface{}{
		"value":     "sg-0123456789abcdef",
		"type":      "string",
		"sensitive": false,
	}

	outputs["database_config"] = map[string]interface{}{
		"value": map[string]interface{}{
			"endpoint": "mydb.cluster-abc123.us-east-1.rds.amazonaws.com:5432",
			"port":     5432,
			"database": "production",
			"username": "dbadmin",
		},
		"type":      "object({endpoint=string, port=number, database=string, username=string})",
		"sensitive": false,
	}

	outputs["database_password"] = map[string]interface{}{
		"value":     "super-secret-password-12345",
		"type":      "string",
		"sensitive": true,
	}

	outputs["load_balancer_dns"] = map[string]interface{}{
		"value":     "my-lb-1234567890.us-east-1.elb.amazonaws.com",
		"type":      "string",
		"sensitive": false,
	}

	outputs["instance_count"] = map[string]interface{}{
		"value":     float64(3),
		"type":      "number",
		"sensitive": false,
	}

	outputs["enable_monitoring"] = map[string]interface{}{
		"value":     true,
		"type":      "bool",
		"sensitive": false,
	}

	outputs["tags"] = map[string]interface{}{
		"value": map[string]interface{}{
			"Environment": "production",
			"Project":     "nomos-provider",
			"ManagedBy":   "terraform",
			"Owner":       "platform-team",
		},
		"type":      "map(string)",
		"sensitive": false,
	}

	baseState["outputs"] = outputs
	data, _ := json.Marshal(baseState)

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ParseStateFile(data)
		if err != nil {
			b.Fatalf("ParseStateFile failed: %v", err)
		}
	}
}

// BenchmarkParseStateFile_Parallel tests concurrent parsing performance
func BenchmarkParseStateFile_Parallel(b *testing.B) {
	data := generateStateFile(100 * 1024) // 100KB
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := ParseStateFile(data)
			if err != nil {
				b.Fatalf("ParseStateFile failed: %v", err)
			}
		}
	})
}
