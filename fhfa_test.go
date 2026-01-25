package fhfa

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {

	//	data, e := Fetch("")
	//	assert.Nil(t, e)

	//	e = Save(data, "/home/will/tmp/test.xlsx")
	//	assert.Nil(t, e)

	hd, e := Parse("/home/will/Downloads/hpi_at_metro.xlsx")
	_ = hd
	assert.Nil(t, e)
}

func TestToYrQtr(t *testing.T) {
	qtrs := []int{1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 4}
	for m := range 12 {
		dt := time.Date(2026, time.Month(m+1), 1, 0, 0, 0, 0, time.UTC)
		yrqtr := ToYrQtr(dt)
		exp := 20260 + qtrs[m]
		assert.Equal(t, exp, yrqtr)
	}
}

func TestHPI(t *testing.T) {
	dt := time.Date(2003, 7, 17, 0, 0, 0, 0, time.UTC)
	dtQtr := ToYrQtr(dt)

	exp := []float64{128.06, 204.16, 135.76, 176.88, 180.56, 117.09, 287.17}
	sources := []string{"msa", "state", "zip3", "nonmsa", "pr", "mh", "us"}
	geo := []string{"10180", "AR", "837", "CA", "PR", "USA", "USA"}
	tmpFile := fmt.Sprintf("%s/hpi.xlsx", os.TempDir())
	defer os.Remove(tmpFile)

	for j, src := range sources {
		url, e0 := URLs(src)
		assert.Nil(t, e0)
		_ = url

		data, e1 := Fetch(url)
		assert.Nil(t, e1)

		e2 := Save(data, tmpFile)
		assert.Nil(t, e2)

		hd, e3 := Parse(tmpFile)
		assert.Nil(t, e3)

		hpi, e4 := hd[geo[j]].HPI(dtQtr)
		assert.Nil(t, e4)
		assert.Equal(t, exp[j], hpi)
	}
}

func TestTime(t *testing.T) {
	hd, e3 := Parse("/home/will/Downloads/hpi_at_metro.xlsx")
	assert.Nil(t, e3)

	const n = 100000

	now := time.Now()
	pulled :=0
	for j := range n {
		yr := 2001 + j%22
		dt := time.Date(yr, 7, 17, 0, 0, 0, 0, time.UTC)
		dtQtr := ToYrQtr(dt)
		for key := range hd {
			hpi, e := hd[key].HPI(dtQtr)
			assert.Nil(t, e)
			_ = hpi
			pulled++
		}
	}

	fmt.Printf("Time to pull %v values: %0.0f seconds", pulled, time.Since(now).Seconds())
}
