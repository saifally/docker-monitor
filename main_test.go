package main

import (
	"testing"

	"github.com/sirupsen/logrus"
)

// Test the formatBytes function
func TestFormatBytes(t *testing.T) {
	monitor := &Monitor{
		logger: logrus.New(),
	}

	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{
			name:     "bytes less than 1KB",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "exactly 1KB",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "1.5 KB",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "1 MB",
			bytes:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "1.2 MB",
			bytes:    1258291, // ~1.2 MB
			expected: "1.2 MB",
		},
		{
			name:     "1 GB",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "2.5 GB",
			bytes:    2684354560, // ~2.5 GB
			expected: "2.5 GB",
		},
		{
			name:     "1 TB",
			bytes:    1024 * 1024 * 1024 * 1024,
			expected: "1.0 TB",
		},
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.bytes, result, tt.expected)
			}
		})
	}
}

// Test the isFromSameComposeProject function
func TestIsFromSameComposeProject(t *testing.T) {
	monitor := &Monitor{
		project: "test-project",
		logger:  logrus.New(),
	}

	tests := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name: "matching com.docker.compose.project",
			labels: map[string]string{
				"com.docker.compose.project": "test-project",
			},
			expected: true,
		},
		{
			name: "non-matching com.docker.compose.project",
			labels: map[string]string{
				"com.docker.compose.project": "other-project",
			},
			expected: false,
		},
		{
			name: "matching com.docker.compose.project.name",
			labels: map[string]string{
				"com.docker.compose.project.name": "test-project",
			},
			expected: true,
		},
		{
			name: "matching docker-compose.project",
			labels: map[string]string{
				"docker-compose.project": "test-project",
			},
			expected: true,
		},
		{
			name: "matching service name with project prefix",
			labels: map[string]string{
				"com.docker.compose.service": "test-project-service",
			},
			expected: true,
		},
		{
			name: "non-matching service name",
			labels: map[string]string{
				"com.docker.compose.service": "other-service",
			},
			expected: false,
		},
		{
			name: "no relevant labels",
			labels: map[string]string{
				"some.other.label": "value",
			},
			expected: false,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: false,
		},
		{
			name:     "nil labels",
			labels:   nil,
			expected: false,
		},
		{
			name: "multiple labels with one matching",
			labels: map[string]string{
				"com.docker.compose.project":      "test-project",
				"com.docker.compose.project.name": "other-project",
				"some.other.label":                "value",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.isFromSameComposeProject(tt.labels)
			if result != tt.expected {
				t.Errorf("isFromSameComposeProject() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test the isFromSameComposeProject function with different project names
func TestIsFromSameComposeProjectWithDifferentProjects(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		labels      map[string]string
		expected    bool
	}{
		{
			name:        "empty project name",
			projectName: "",
			labels: map[string]string{
				"com.docker.compose.project": "some-project",
			},
			expected: false,
		},
		{
			name:        "default project name",
			projectName: "default",
			labels: map[string]string{
				"com.docker.compose.project": "default",
			},
			expected: true,
		},
		{
			name:        "project with special characters",
			projectName: "my-app_v2.0",
			labels: map[string]string{
				"com.docker.compose.project": "my-app_v2.0",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := &Monitor{
				project: tt.projectName,
				logger:  logrus.New(),
			}
			result := monitor.isFromSameComposeProject(tt.labels)
			if result != tt.expected {
				t.Errorf("isFromSameComposeProject() with project '%s' = %v, want %v", tt.projectName, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance-critical functions
func BenchmarkFormatBytes(b *testing.B) {
	monitor := &Monitor{
		logger: logrus.New(),
	}

	testBytes := uint64(1258291) // ~1.2 MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.formatBytes(testBytes)
	}
}

func BenchmarkIsFromSameComposeProject(b *testing.B) {
	monitor := &Monitor{
		project: "test-project",
		logger:  logrus.New(),
	}

	labels := map[string]string{
		"com.docker.compose.project":      "test-project",
		"com.docker.compose.project.name": "test-project",
		"com.docker.compose.service":      "web",
		"some.other.label":                "value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.isFromSameComposeProject(labels)
	}
}

// Test edge cases for formatBytes
func TestFormatBytesEdgeCases(t *testing.T) {
	monitor := &Monitor{
		logger: logrus.New(),
	}

	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{
			name:     "1023 bytes (just under 1KB)",
			bytes:    1023,
			expected: "1023 B",
		},
		{
			name:     "1025 bytes (just over 1KB)",
			bytes:    1025,
			expected: "1.0 KB",
		},
		{
			name:     "very large number (5 PB)",
			bytes:    5 * 1024 * 1024 * 1024 * 1024 * 1024,
			expected: "5.0 PB",
		},
		{
			name:     "maximum possible value test",
			bytes:    ^uint64(0), // Maximum uint64 value
			expected: "16.0 EB",   // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.bytes, result, tt.expected)
			}
		})
	}
}

// Test case sensitivity in project matching
func TestIsFromSameComposeProjectCaseSensitive(t *testing.T) {
	monitor := &Monitor{
		project: "Test-Project",
		logger:  logrus.New(),
	}

	tests := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name: "exact case match",
			labels: map[string]string{
				"com.docker.compose.project": "Test-Project",
			},
			expected: true,
		},
		{
			name: "different case - should not match",
			labels: map[string]string{
				"com.docker.compose.project": "test-project",
			},
			expected: false,
		},
		{
			name: "uppercase - should not match",
			labels: map[string]string{
				"com.docker.compose.project": "TEST-PROJECT",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.isFromSameComposeProject(tt.labels)
			if result != tt.expected {
				t.Errorf("isFromSameComposeProject() = %v, want %v", result, tt.expected)
			}
		})
	}
}