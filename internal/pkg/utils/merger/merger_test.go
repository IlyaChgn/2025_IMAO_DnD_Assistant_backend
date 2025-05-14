package merger_test

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils/merger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		patch    []byte
		expected []byte
	}{
		{
			name:     "must merge simple values",
			data:     []byte(`{"a": 1, "b": 2.45, "c": "start"}`),
			patch:    []byte(`{"a": 3, "b": 2.09, "c": "finish"}`),
			expected: []byte(`{"a":3,"b":2.09,"c":"finish"}`),
		},
		{
			name:     "must replace arrays",
			data:     []byte(`{"a": [1, 2, 3]}`),
			patch:    []byte(`{"a": [4, 5, 3]}`),
			expected: []byte(`{"a":[4,5,3]}`),
		},
		{
			name:     "must merge objects",
			data:     []byte(`{"obj1": {"a": 1, "b": 2, "c": 3}, "obj2": {"a": 4, "b": 5, "c": 6}}`),
			patch:    []byte(`{"obj1": {"a": 4, "b": 5, "c": 6}, "obj2": {"a": 1, "b": 2, "c": 3}}`),
			expected: []byte(`{"obj1":{"a":4,"b":5,"c":6},"obj2":{"a":1,"b":2,"c":3}}`),
		},
		{
			name:     "must sort by keys",
			data:     []byte(`{"c": 1, "a": 2, "b": 3}`),
			patch:    []byte(`{"c": 3, "b": 2, "a": 1}`),
			expected: []byte(`{"a":1,"b":2,"c":3}`),
		},
		{
			name:     "must save existing values",
			data:     []byte(`{"a": 1, "b": 2, "c": 3, "obj1": {"a": 1, "b": 2, "c": 4}, "arr1": ["1", "2", "3"]}`),
			patch:    []byte(`{"a": 3, "b": 3}`),
			expected: []byte(`{"a":3,"arr1":["1","2","3"],"b":3,"c":3,"obj1":{"a":1,"b":2,"c":4}}`),
		},
		{
			name:     "must add new fields",
			data:     []byte(`{"a": 1, "b": 2}`),
			patch:    []byte(`{"c": 3, "d": 4}`),
			expected: []byte(`{"a":1,"b":2,"c":3,"d":4}`),
		},
		{
			name: "must process complex structures",
			data: []byte(`{"a": 1, "b": "f", 
							"objArr": [{"a": 2.04, "b": [1, 2, 3]}, {"a": 2.04, "obj": {"a": 2, "b": "f"}}],
							"object": {"a": 1, "b": {"a": 2, "c": {"a": "1", "b": [1, 2]}}}}`),
			patch: []byte(`{"a": 1, "b": "q", 
							"objArr": [{"a": 2.1, "b": [1, 4], "c": "ff"}, 2, 3],
							"object": {"a": 1, "c": 5}}`),
			expected: []byte(`{"a":1,"b":"q","objArr":[{"a":2.1,"b":[1,4],"c":"ff"},2,3],"object":{"a":1,"b":{"a":2,"c":{"a":"1","b":[1,2]}},"c":5}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := merger.Merge(tt.data, tt.patch)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
