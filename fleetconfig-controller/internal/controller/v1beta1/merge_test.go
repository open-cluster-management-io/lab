package v1beta1

import (
	"reflect"
	"testing"
)

func TestDeepMergeNonZero(t *testing.T) {
	tests := []struct {
		name string
		dst  map[string]any
		src  map[string]any
		want map[string]any
	}{
		{
			name: "merge non-zero values only",
			dst: map[string]any{
				"stringField": "original",
				"intField":    10,
				"boolField":   true,
				"floatField":  1.5,
				"sliceField":  []any{"original"},
				"mapField":    map[string]any{"key": "original"},
			},
			src: map[string]any{
				"stringField": "",               // zero value - should be ignored
				"intField":    0,                // zero value - should be ignored
				"boolField":   false,            // valid boolean - should override
				"floatField":  0.0,              // zero value - should be ignored
				"sliceField":  []any{},          // zero value - should be ignored
				"mapField":    map[string]any{}, // zero value - should be ignored
				"newField":    "new",            // new non-zero value - should be added
			},
			want: map[string]any{
				"stringField": "original",                        // kept from dst (src was zero)
				"intField":    10,                                // kept from dst (src was zero)
				"boolField":   false,                             // overridden by src (boolean always valid)
				"floatField":  1.5,                               // kept from dst (src was zero)
				"sliceField":  []any{"original"},                 // kept from dst (src was zero)
				"mapField":    map[string]any{"key": "original"}, // kept from dst (src was zero)
				"newField":    "new",                             // added from src
			},
		},
		{
			name: "nested map merging",
			dst: map[string]any{
				"nested": map[string]any{
					"keepThis":     "original",
					"overrideThis": "original",
					"zeroThis":     "original",
				},
				"topLevel": "keep",
			},
			src: map[string]any{
				"nested": map[string]any{
					"overrideThis": "new",
					"zeroThis":     "",    // zero value - should be ignored
					"addThis":      "add", // new field
				},
				"topLevel": "", // zero value - should be ignored
			},
			want: map[string]any{
				"nested": map[string]any{
					"keepThis":     "original", // unchanged
					"overrideThis": "new",      // overridden
					"zeroThis":     "original", // kept (src was zero)
					"addThis":      "add",      // added
				},
				"topLevel": "keep", // kept (src was zero)
			},
		},
		{
			name: "all data types with zero and non-zero values",
			dst: map[string]any{
				"string":    "dst",
				"int":       1,
				"int8":      int8(1),
				"int16":     int16(1),
				"int32":     int32(1),
				"int64":     int64(1),
				"uint":      uint(1),
				"uint8":     uint8(1),
				"uint16":    uint16(1),
				"uint32":    uint32(1),
				"uint64":    uint64(1),
				"float32":   float32(1.1),
				"float64":   1.1,
				"boolTrue":  true,
				"boolFalse": false,
				"slice":     []any{"dst"},
				"map":       map[string]any{"key": "dst"},
			},
			src: map[string]any{
				"string":    "",               // zero - ignore
				"int":       0,                // zero - ignore
				"int8":      int8(0),          // zero - ignore
				"int16":     int16(0),         // zero - ignore
				"int32":     int32(0),         // zero - ignore
				"int64":     int64(0),         // zero - ignore
				"uint":      uint(0),          // zero - ignore
				"uint8":     uint8(0),         // zero - ignore
				"uint16":    uint16(0),        // zero - ignore
				"uint32":    uint32(0),        // zero - ignore
				"uint64":    uint64(0),        // zero - ignore
				"float32":   float32(0.0),     // zero - ignore
				"float64":   0.0,              // zero - ignore
				"boolTrue":  false,            // valid boolean - override
				"boolFalse": true,             // valid boolean - override
				"slice":     []any{},          // zero - ignore
				"map":       map[string]any{}, // zero - ignore
				"newString": "new",            // non-zero - add
				"newInt":    42,               // non-zero - add
				"newBool":   true,             // non-zero - add
			},
			want: map[string]any{
				"string":    "dst",                        // kept
				"int":       1,                            // kept
				"int8":      int8(1),                      // kept
				"int16":     int16(1),                     // kept
				"int32":     int32(1),                     // kept
				"int64":     int64(1),                     // kept
				"uint":      uint(1),                      // kept
				"uint8":     uint8(1),                     // kept
				"uint16":    uint16(1),                    // kept
				"uint32":    uint32(1),                    // kept
				"uint64":    uint64(1),                    // kept
				"float32":   float32(1.1),                 // kept
				"float64":   1.1,                          // kept
				"boolTrue":  false,                        // overridden
				"boolFalse": true,                         // overridden
				"slice":     []any{"dst"},                 // kept
				"map":       map[string]any{"key": "dst"}, // kept
				"newString": "new",                        // added
				"newInt":    42,                           // added
				"newBool":   true,                         // added
			},
		},
		{
			name: "nil values handling",
			dst: map[string]any{
				"existing": "value",
			},
			src: map[string]any{
				"existing": nil,   // nil - should be ignored
				"newNil":   nil,   // nil - should be ignored
				"newValue": "add", // non-zero - should be added
			},
			want: map[string]any{
				"existing": "value", // kept (src was nil)
				"newValue": "add",   // added
			},
		},
		{
			name: "empty dst map",
			dst:  map[string]any{},
			src: map[string]any{
				"zero":    0,
				"nonZero": 42,
				"empty":   "",
				"valid":   "value",
			},
			want: map[string]any{
				"nonZero": 42,
				"valid":   "value",
			},
		},
		{
			name: "empty src map",
			dst: map[string]any{
				"keep": "this",
			},
			src: map[string]any{},
			want: map[string]any{
				"keep": "this",
			},
		},
		{
			name: "mixed type override (non-map to map)",
			dst: map[string]any{
				"field": "string",
			},
			src: map[string]any{
				"field": map[string]any{"nested": "value"},
			},
			want: map[string]any{
				"field": map[string]any{"nested": "value"}, // src overrides dst
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a deep copy of dst to avoid modifying the test case
			dstCopy := make(map[string]any)
			for k, v := range tt.dst {
				dstCopy[k] = deepCopyValue(v)
			}

			deepMergeNonZero(dstCopy, tt.src)

			if !reflect.DeepEqual(dstCopy, tt.want) {
				t.Errorf("deepMergeNonZero() result mismatch\ngot:  %+v\nwant: %+v", dstCopy, tt.want)
			}
		})
	}
}

// deepCopyValue creates a deep copy of a value for test isolation
func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		copy := make(map[string]any)
		for k, v := range val {
			copy[k] = deepCopyValue(v)
		}
		return copy
	case []any:
		copy := make([]any, len(val))
		for i, v := range val {
			copy[i] = deepCopyValue(v)
		}
		return copy
	default:
		return v
	}
}
