package zhifou

import (
	"strconv"
	"fmt"
)

func ParseFloat(f float64, n int) float64 {
	value, _ := strconv.ParseFloat(fmt.Sprintf("%." + fmt.Sprintf("%d", n) + "f", f), 64)
	return value
}