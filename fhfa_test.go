package fhfa

import (
	"fmt"
	"os"
	"strings"
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

func TestHPIdata_Index(t *testing.T) {
	dt := time.Date(2003, 7, 17, 0, 0, 0, 0, time.UTC)
	dtQtr := ToYrQtr(dt)

	exp := []float64{128.06, 204.16, 135.76, 176.88, 180.56, 117.09, 287.17}
	sources := []string{"metro", "state", "zip3", "nonmetro", "pr", "mh", "us"}
	geo := []string{"10180", "AR", "837", "CA", "PR", "USA", "USA"}

	for j, src := range sources {
		hd, e1 := Load(src)
		assert.Nil(t, e1)

		hpi, e2 := hd.Index(geo[j], dtQtr)
		assert.Nil(t, e2)
		assert.Equal(t, exp[j], hpi)
	}
}

func TestHPIdata_Change(t *testing.T) {
	exp := []float64{1.328, 1.350, 1.582, 1.322, 1.21, 1.448, 1.368}
	sources := []string{"metro", "state", "zip3", "nonmetro", "pr", "mh", "us"}
	geo := []string{"10180", "AR", "837", "CA", "PR", "USA", "USA"}

	for j, src := range sources {
		hd, e1 := Load(src)
		assert.Nil(t, e1)

		c, e2 := hd.Change(geo[j], 20201, 20222)
		assert.Nil(t, e2)
		assert.InEpsilon(t, exp[j], c, 0.001)
	}
}

func TestHPIdata_geoLevel(t *testing.T) {
	exp := []string{"metro", "state", "zip3", "nonmetro", "pr", "mh", "us"}
	files := []string{"hpi_at_metro", "hpi_at_state", "hpi_at_3zip", "hpi_at_nonmetro",
		"hpi_at_pr", "hpi_at_mh", "hpi_at_us_and_census"}

	for j, file := range files {
		fn := "/home/will/Downloads/" + file + ".xlsx"
		fn = exp[j]
		hd, e := Load(fn)
		assert.Nil(t, e)
		assert.Equal(t, exp[j], hd.GeoLevel())
	}
}

func TestBest(t *testing.T) {
	sources := []string{"metro", "nonmetro", "state", "pr"}
	//sources := []string{"/home/will/Downloads/hpi_at_metro.xlsx", "/home/will/Downloads/hpi_at_nonmetro.xlsx",
	//		"/home/will/Downloads/hpi_at_state.xlsx", "/home/will/Downloads/hpi_at_pr.xlsx"}

	var hpis []*HPIdata

	for _, src := range sources {
		hd, e3 := Load(src)
		assert.Nil(t, e3)

		hpis = append(hpis, hd)
	}

	keys := []string{"14260", "ID", "ID", "ID"}

	_, geoLevel, e := Best(20251, keys, hpis)
	assert.Nil(t, e)
	ok := strings.Contains(geoLevel, "metro")
	assert.Equal(t, true, ok)

	keys = []string{"XXXXX", "ID", "ID", "ID"}

	_, geoLevel, e = Best(20251, keys, hpis)
	assert.Nil(t, e)
	ok = strings.Contains(geoLevel, "nonmetro")
	assert.Equal(t, true, ok)

	keys = []string{"XXXXX", "PR", "PR", "PR"}

	_, geoLevel, e = Best(20251, keys, hpis)
	assert.Nil(t, e)
	ok = strings.Contains(geoLevel, "pr")
	assert.Equal(t, true, ok)

}

func TestHPIdata_Save(t *testing.T) {
	src := "/home/will/Downloads/hpi_at_metro.xlsx"
	hd, e := Load(src)
	assert.Nil(t, e)

	tmpFile := fmt.Sprintf("%s/hpi.csv", os.TempDir())
	e1 := hd.Save(tmpFile)
	assert.Nil(t, e1)
}

func TestTimes(t *testing.T) {
	hd, e3 := Load("/home/will/Downloads/hpi_at_metro.xlsx")
	assert.Nil(t, e3)

	const n = 100000

	now := time.Now()
	pulled := 0
	for j := range n {
		yr := 2001 + j%22
		dt := time.Date(yr, 7, 17, 0, 0, 0, 0, time.UTC)
		dtQtr := ToYrQtr(dt)
		for key := range hd.series {
			hpi, e := hd.series[key].index(dtQtr)
			assert.Nil(t, e)
			_ = hpi
			pulled++
		}
	}

	fmt.Printf("Time to pull %v values: %0.0f seconds", pulled, time.Since(now).Seconds())
}
