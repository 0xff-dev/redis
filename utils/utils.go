package utils

import "math"

func Min(args ...int) int {
	if len(args) == 0 {
		return -1
	}
	min := args[0]
	for index := 1; index < len(args); index++ {
		if args[index] < min {
			min = args[index]
		}
	}
	return min
}

func Int64Min(args ...int64) int64 {
	if len(args) == 0 {
		return -1
	}
	min := args[0]
	for index := 1; index < len(args); index++ {
		if args[index] < min {
			min = args[index]
		}
	}
	return min
}

func Max(args ...int) int {
	if len(args) == 0 {
		return math.MaxInt32
	}
	max := args[0]
	for index := 1; index < len(args); index++ {
		if args[index] > max {
			max = args[index]
		}
	}
	return max
}

func Int64Max(args ...int64) int64 {
	if len(args) == 0 {
		return math.MaxInt64
	}

	max := args[0]
	for index := 1; index < len(args); index++ {
		if args[index] > max {
			max = args[index]
		}
	}
	return max
}


