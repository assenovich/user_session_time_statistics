package main

import (
	"sort"
)

func calcMean(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, value := range values {
		sum += value
	}
	return sum / int64(len(values))
}

type Int64Slice []int64

func (a Int64Slice) Len() int           { return len(a) }
func (a Int64Slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Int64Slice) Less(i, j int) bool { return a[i] < a[j] }

func calcMedian(values []int64) int64 {
	l := len(values)
	sort.Sort(Int64Slice(values))
	if l == 0 {
		return 0
	} else if l%2 == 0 {
		return (values[l/2-1] + values[l/2+1]) / 2
	} else {
		return values[l/2]
	}
}
