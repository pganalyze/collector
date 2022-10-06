package postgres

import (
	"fmt"
	"reflect"
	"testing"
)

var testGraph = []Edge{
	{
		from: 641449,
		to:   641586,
	},
	{
		from: 641449,
		to:   641594,
	},
	{
		from: 641586,
		to:   641594,
	},
	{
		from: 641594,
		to:   641588,
	},
	{
		from: 641594,
		to:   641590,
	},
	{
		from: 641590,
		to:   641599,
	},
	{
		from: 641667,
		to:   641669,
	},
}

var findBlockingTests = []struct {
	input    int64
	expected []int64
}{
	{
		641449,
		[]int64{641586, 641594},
	},
	{
		641586,
		[]int64{641594},
	},
	{
		641594,
		[]int64{641588, 641590},
	},
	{
		641588,
		[]int64{},
	},
	{
		641590,
		[]int64{641599},
	},
	{
		641599,
		[]int64{},
	},
	{
		641667,
		[]int64{641669},
	},
	{
		641669,
		[]int64{},
	},
}

var findIndirectlyBlockingTests = []struct {
	input    []int64
	expected []int64
}{
	{
		[]int64{641586, 641594},
		[]int64{641594, 641588, 641590, 641599},
	},
	{
		[]int64{641594},
		[]int64{641588, 641590, 641599},
	},
	{
		[]int64{641588, 641590},
		[]int64{641599},
	},
	{
		[]int64{641599},
		[]int64{},
	},
	{
		[]int64{641669},
		[]int64{},
	},
	{
		[]int64{},
		[]int64{},
	},
}

var findIndirectlyBlockedByTests = []struct {
	input    []int64
	expected []int64
}{
	{
		[]int64{641449},
		[]int64{},
	},
	{
		[]int64{641586, 641449},
		[]int64{641449},
	},
	{
		[]int64{641594},
		[]int64{641449, 641586},
	},
	{
		[]int64{641590},
		[]int64{641594, 641449, 641586},
	},
	{
		[]int64{641667},
		[]int64{},
	},
	{
		[]int64{},
		[]int64{},
	},
}

func TestFindBlocking(t *testing.T) {
	for _, test := range findBlockingTests {
		t.Run(fmt.Sprintf("with pid %d", test.input), func(t *testing.T) {
			actual := findBlocking(testGraph, test.input)
			if reflect.DeepEqual(actual, test.expected) == false {
				t.Errorf("got %d, want %d", actual, test.expected)
			}
		})
	}
}

func TestFindIndirectlyBlocking(t *testing.T) {
	for _, test := range findIndirectlyBlockingTests {
		t.Run(fmt.Sprintf("with pids %d", test.input), func(t *testing.T) {
			actual := findIndirectlyBlocking(testGraph, test.input)
			if reflect.DeepEqual(actual, test.expected) == false {
				t.Errorf("got %d, want %d", actual, test.expected)
			}
		})
	}
}

func TestFindIndirectlyBlockedBy(t *testing.T) {
	for _, test := range findIndirectlyBlockedByTests {
		t.Run(fmt.Sprintf("with pids %d", test.input), func(t *testing.T) {
			actual := findIndirectlyBlockedBy(testGraph, test.input)
			if reflect.DeepEqual(actual, test.expected) == false {
				t.Errorf("got %d, want %d", actual, test.expected)
			}
		})
	}
}
