package fhfa

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const source = "file" // "file" reads from local xlsx, anything else pulls from web

func sources() []string {
	if source == "file" {
		files := []string{"hpi_at_3zip", "hpi_at_metro", "hpi_at_nonmetro", "hpi_at_state", "hpi_at_us_and_census",
			"hpi_at_pr", "hpi_at_mh"}
		dir := os.Getenv("fhfaDir")
		for j, f := range files {
			files[j] = dir + f + ".xlsx"
		}

		return files
	}

	var srcs []string
	for _, s := range []string{"zip3", "metro", "nonmetro", "state", "us", "pr", "mh"} {
		srcs = append(srcs, URLs(s))
	}

	return srcs
}

func TestQtrDiff(t *testing.T) {
	dt0 := []int{20221, 20222, 20223, 20224}
	dt1 := []int{20232, 20213, 20221, 20251}
	exp := []int{5, 3, 2, 9}

	for j, d0 := range dt0 {
		d := QtrDiff(d0, dt1[j])
		assert.Equal(t, exp[j], d)
	}
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

func TestToDate(t *testing.T) {
	qtrs := []int{1, 2, 3, 4}
	mo := []time.Month{1, 4, 7, 10}
	yrs := []int{2000, 2020, 2021, 2025}
	for j, yr := range yrs {
		dt := 10*yr + qtrs[j]
		dttm, e := ToTime(dt)
		assert.Nil(t, e)
		exp := time.Date(yr, mo[j], 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, exp, dttm)
	}
}

func TestHPIdata_Index(t *testing.T) {
	dt := time.Date(2003, 7, 17, 0, 0, 0, 0, time.UTC)
	dtQtr := ToYrQtr(dt)

	sources := sources()
	geo := []string{"837", "10180", "CA", "AR", "USA", "PR", "USA"}
	exp := []float32{135.76, 128.06, 176.88, 204.16, 287.17, 180.56, 117.09}

	for j, src := range sources {
		hd, e1 := Load(src)
		assert.Nil(t, e1)

		hpi, e2 := hd.Index(geo[j], dtQtr)
		assert.Nil(t, e2)
		assert.Equal(t, exp[j], hpi)

		hpis, _ := hd.Geo(geo[j])
		fmt.Println(hpis)
	}
}

func TestHPIdata_Change(t *testing.T) {
	sources := sources()
	exp := []float32{1.582, 1.328, 1.322, 1.350, 1.368, 1.21, 1.448}
	geo := []string{"837", "10180", "CA", "AR", "USA", "PR", "USA"}
	for j, src := range sources {
		hd, e1 := Load(src)
		assert.Nil(t, e1)

		c, e2 := hd.Change(geo[j], 20201, 20222)
		assert.Nil(t, e2)
		assert.InEpsilon(t, exp[j], c, 0.001)

		if j == 0 {
			fmt.Println(hd)
		}
	}
}

func TestHPIdata_geoLevel(t *testing.T) {
	exp := []string{"zip3", "metro", "nonmetro", "state", "us", "pr", "mh"}
	sources := sources()

	for j, src := range sources {
		hd, e := Load(src)
		assert.Nil(t, e)
		assert.Equal(t, exp[j], hd.GeoLevel())
	}
}

func TestBest(t *testing.T) {
	s := sources()
	sources := []string{s[1], s[2], s[3], s[5]}

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
			hpi, e := hd.Index(key, dtQtr)
			assert.Nil(t, e)
			_ = hpi
			pulled++
		}
	}

	fmt.Printf("Time to pull %v values: %0.0f seconds", pulled, time.Since(now).Seconds())
}

func TestHPIdata_Append(t *testing.T) {
	const (
		growth = 3.0
		nQtr   = 12
	)

	srcs := sources()
	hd, e := Load(srcs[3])
	assert.Nil(t, e)

	g := float32(math.Pow(1.0+growth, 1/12.0))

	var dts []int
	for j, geo := range hd.Geos() {
		s, e := hd.Geo(geo)
		assert.Nil(t, e)
		ld, li := s.Last()
		liKeep := li
		if j == 0 {
			for range nQtr {
				ld = NextQtr(ld)
				dts = append(dts, ld)
			}
		}

		var ind []float32
		for range nQtr {
			li *= g
			ind = append(ind, li)
		}

		e1 := s.Append(dts, ind)
		assert.Nil(t, e1)

		act, e2 := s.Index(dts[len(dts)-1])
		assert.Nil(t, e2)
		exp := float32(math.Pow(growth, 3)) * liKeep
		assert.Equal(t, exp, act)
	}
}

// TODO delete unneeded funcs such as Fetch.
