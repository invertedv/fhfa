package fhfa

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)


func TestToYrQtr(t *testing.T) {
	qtrs := []int{1, 1, 1, 2, 2, 2, 3, 3, 3, 4, 4, 4}
	for m := range 12 {
		dt := time.Date(2026, time.Month(m+1), 1, 0, 0, 0, 0, time.UTC)
		yrqtr := ToYrQtr(dt)
		exp := 20260 + qtrs[m]
		assert.Equal(t, exp, yrqtr)
	}
}

func TestHPIdata_HPI(t *testing.T) {
	dt := time.Date(2003, 7, 17, 0, 0, 0, 0, time.UTC)
	dtQtr := ToYrQtr(dt)

	exp := []float64{128.06, 204.16, 135.76, 176.88, 180.56, 117.09, 287.17}
	sources := []string{"msa", "state", "zip3", "nonmsa", "pr", "mh", "us"}
	geo := []string{"10180", "AR", "837", "CA", "PR", "USA", "USA"}
	tmpFile := fmt.Sprintf("%s/hpi.xlsx", os.TempDir())
	defer os.Remove(tmpFile)

	for j, src := range sources {
		if j!=0 {
			continue
		}
		_=src
		src = "/home/will/Downloads/hpi_at_metro.xlsx"
		hd, e3 := Parse(src)
		assert.Nil(t, e3)

		hpi, e4 := hd.series[geo[j]].HPI(dtQtr)
		assert.Nil(t, e4)
		assert.Equal(t, exp[j], hpi)
	}
}

func TestBest(t *testing.T) {
	sources := []string{"msa", "nonmsa", "state", "pr"}

	var hpis []*HPIdata

	for _, src := range sources {
		hd, e3 := Parse(src)
		assert.Nil(t, e3)

		hpis = append(hpis, hd)
	}

}

func TestTimes(t *testing.T) {
	hd, e3 := Parse("/home/will/Downloads/hpi_at_metro.xlsx")
	assert.Nil(t, e3)

	const n = 100000

	now := time.Now()
	pulled :=0
	for j := range n {
		yr := 2001 + j%22
		dt := time.Date(yr, 7, 17, 0, 0, 0, 0, time.UTC)
		dtQtr := ToYrQtr(dt)
		for key := range hd.series {
			hpi, e := hd.series[key].HPI(dtQtr)
			assert.Nil(t, e)
			_ = hpi
			pulled++
		}
	}

	fmt.Printf("Time to pull %v values: %0.0f seconds", pulled, time.Since(now).Seconds())
}
