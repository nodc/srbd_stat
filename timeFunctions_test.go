package main

import (
	"testing"
	"time"
	"fmt"
)

func TestIsActual(t *testing.T) {
	lastActualizeTime := time.Date(2015, 12, 18, 13, 43, 0, 0, time.UTC)
	endDateTime := time.Date(2015, 12, 18, 13, 43, 0, 0, time.UTC)
	freq := "unknown"
	processing := "observation"
	ok := IsActual(lastActualizeTime, endDateTime, freq, processing)
	fmt.Println(ok)
	// Output: true
}

func TestForecast(t *testing.T) {
	// test forecast
	lastActualizeTime := time.Date(2015, 6, 30, 10, 49, 0, 0, time.UTC)
	endDateTime := time.Date(2015, 7, 5, 0, 0, 0, 0, time.UTC)
	freq := "daily"
	processing := "forecast"
	ok := IsActual(lastActualizeTime, endDateTime, freq, processing)
	fmt.Println(ok)
	// Output: false
}

func TestBackward(t *testing.T) {
	now := time.Date(2016, 1, 1, 10, 0, 0, 0, time.UTC)
	shifted := Backward(now, "hourly");
	fmt.Println(shifted)
	// Output: 2016-01-01 09:00:00 +0000 UTC
}

