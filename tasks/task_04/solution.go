package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {

	if len(nums) < 2 {
		return Stats{}
	}

	result := Stats{}
	result.Min = math.MaxInt
	result.Max = math.MinInt
	result.Count = len(nums) - 1
	result.Sum = nums[len(nums)-1] - nums[0] // Телескопическая сумма позволяет посчитать одним действием

	for i := 1; i < len(nums); i++ {

		diff := nums[i] - nums[i-1]

		if result.Min > diff {
			result.Min = diff
		}

		if result.Max < diff {
			result.Max = diff
		}
	}

	return result

}
