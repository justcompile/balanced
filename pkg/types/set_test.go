package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet_Add(t *testing.T) {
	tests := map[string]struct {
		set      Set[string]
		value    string
		expected Set[string]
	}{
		"adds value to set when it doesn't already exist": {
			make(Set[string]),
			"foo",
			Set[string]{"foo": struct{}{}},
		},
		"does not duplicate value in set when it already exists": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			"foo",
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
		},
	}

	for name, test := range tests {
		test.set.Add(test.value)
		assert.Equal(t, test.expected, test.set, name)
	}
}

func TestSet_Remove(t *testing.T) {
	tests := map[string]struct {
		set      Set[string]
		value    string
		expected Set[string]
	}{
		"removes value does not change set when value doesn't already exist": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			"fizz",
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
		},
		"removes value in set when it already exists": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			"foo",
			Set[string]{"bar": struct{}{}},
		},
	}

	for name, test := range tests {
		test.set.Remove(test.value)
		assert.Equal(t, test.expected, test.set, name)
	}
}

func TestSet_Has(t *testing.T) {
	tests := map[string]struct {
		set      Set[string]
		value    string
		expected bool
	}{
		"returns true when value exists in set": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			"foo",
			true,
		},
		"returns false when value does not exist in set": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			"fizz",
			false,
		},
	}

	for name, test := range tests {
		exists := test.set.Has(test.value)
		assert.Equal(t, test.expected, exists, name)
	}
}

func TestSet_Diff(t *testing.T) {
	tests := map[string]struct {
		set      Set[string]
		setB     Set[string]
		expected Set[string]
	}{
		"returns empty set when both sets are empty": {
			make(Set[string]),
			make(Set[string]),
			make(Set[string]),
		},
		"returns empty set when both sets are equal": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			Set[string]{"bar": struct{}{}, "foo": struct{}{}},
			make(Set[string]),
		},
		"returns values from set A when setB is empty": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			make(Set[string]),
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
		},
		"returns empty set when set is empty": {
			make(Set[string]),
			Set[string]{"bar": struct{}{}, "foo": struct{}{}},
			make(Set[string]),
		},
		"returns set containing values which exist in set but not in setB": {
			Set[string]{"bar": struct{}{}, "foo": struct{}{}, "fizz": struct{}{}},
			Set[string]{"bar": struct{}{}, "foo": struct{}{}},
			Set[string]{"fizz": struct{}{}},
		},
	}

	for name, test := range tests {
		result := test.set.Diff(test.setB)
		assert.Equal(t, test.expected, result, name)
	}
}

func TestSet_Intersect(t *testing.T) {
	tests := map[string]struct {
		set      Set[string]
		setB     Set[string]
		expected Set[string]
	}{
		"returns empty set when both sets are empty": {
			make(Set[string]),
			make(Set[string]),
			make(Set[string]),
		},
		"returns set of values which exist in both sets": {
			Set[string]{"foo": struct{}{}},
			Set[string]{"bar": struct{}{}, "foo": struct{}{}},
			Set[string]{"foo": struct{}{}},
		},
		"returns empty set of when no values match": {
			Set[string]{"foo": struct{}{}},
			Set[string]{"bar": struct{}{}, "fizz": struct{}{}},
			make(Set[string]),
		},
		"returns all values from both sets contain the same items": {
			Set[string]{"foo": struct{}{}, "bar": struct{}{}},
			Set[string]{"bar": struct{}{}, "foo": struct{}{}},
			Set[string]{"bar": struct{}{}, "foo": struct{}{}},
		},
	}

	for name, test := range tests {
		result := test.set.Intersect(test.setB)
		assert.Equal(t, test.expected, result, name)
	}
}
