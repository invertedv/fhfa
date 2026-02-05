/*
# Purpose

The purpose of this package is to make the FHFA house price indices available in a format that is readily usable within Go.

# Details

This package will:

1. pull the non-seasonally adjusted house price indices from the FHFA web site or load them from a local xlsx.

2. find the house price index for individual geos and dates.

The series available are:

- metro

- non-metro

- state

- us

- Puerto Rico

- manufactured housing

Note that all dates are ints in CCYYQ format.

The XLSX format is chosen since not all of the data is available as a CSV but all are as XLSX.

There are two basic data types here:

- HPIseries holds the data for a specific geographic location (e.g. State=NY, zip3=837).

- HPIdata holds the data for all the geographic locations of a specific type (e.g. zip3, MSA, State).
*/
package fhfa

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/invertedv/dass"
)

// HPIdata manages all the series at a geographic level (e.g. all states, MSAs, etc)
type HPIdata struct {
	source   string
	geoLevel string
	series   map[string]*HPIseries
}

// NewHPIdata creates a HPIdata struct
//
// geoLevel - geographic level of the data, e.g. zip3, msa, state
//
// series - individual series
func NewHPIdata(geoLevel string, series map[string]*HPIseries) (*HPIdata, error) {
	if !in(geoLevel, []string{"zip3", "metro", "nonmetro", "state", "us", "pr", "mh"}) {
		return nil, fmt.Errorf("invalid geo level: %s", geoLevel)
	}

	return &HPIdata{
		source:   "NewHPIdata()",
		geoLevel: geoLevel,
		series:   series,
	}, nil
}

func Load(source string) (*HPIdata, error) {
	var (
		r    [][]string
		rows *dass.Rows
		e    error
	)

	if r, e = dass.FetchXLSX(source); e != nil {
		return nil, e
	}
	geoLevel := geoLevel(r[0][0])
	template := []string{"string", "int", "int", "float"}
	names := []string{"geoCode", "year", "qtr", "index"}
	miss := []string{"skip", "skip", "skip", "skip"}
	if geoLevel == "metro" {
		template = []string{"string", "string", "int", "int", "float"}
		names = []string{"areaName", "geoCode", "year", "qtr", "index"}
		miss = []string{"skip", "skip", "skip", "skip", "skip"}
	}

	if rows, e = dass.ParseRows(r, names, template, miss, 0); e != nil {
		return nil, e
	}

	lastGeo := ""

	hd := &HPIdata{
		source:   source,
		geoLevel: geoLevel,
		series:   make(map[string]*HPIseries),
	}

	var series *HPIseries

	for _, row := range rows.Iter() {
		if len(row) < 4 {
			continue
		}

		geo := row["geoCode"].(string)

		// New geo?
		if geo != lastGeo {
			lastGeo = geo
			key := row["geoCode"].(string)

			series = &HPIseries{
				geoName: geo,
				geoCode: key,
			}

			hd.series[key] = series
		}

		yrQtr := 10*row["year"].(int) + row["qtr"].(int)
		indx := row["index"].(float32)

		series.dates = append(series.dates, yrQtr)
		series.indx = append(series.indx, indx)
		series.lastDt = yrQtr
		series.lastIndx = indx
	}

	return hd, nil
}

// Append appends ta to the existing HPIData.
func (hd *HPIdata) Append(ta *HPIdata) error {
	if hd.geoLevel != ta.geoLevel {
		return fmt.Errorf("geoLevel not the same in append")
	}

	for k, v := range hd.series {
		var (
			va *HPIseries
			e  error
		)
		if va, e = ta.Geo(k); e != nil {
			return fmt.Errorf("cannot find geo %s in append data", k)
		}

		if e1 := v.Append(va.dates, va.indx); e1 != nil {
			return e1
		}
	}

	return nil
}

// Change returns the ratio of the house price index at dtEnd (CCYYQ) to dtStart (CCYYQ)
func (hd *HPIdata) Change(geo string, dtStart, dtEnd int) (float32, error) {
	var (
		s *HPIseries
		e error
	)

	if s, e = hd.Geo(geo); e != nil {
		return 0, e
	}

	return s.Change(dtStart, dtEnd)
}

// ChangeTime returns the ratio of the house price index at dtEnd to dtStart
func (hd *HPIdata) ChangeTime(geo string, dtStart, dtEnd time.Time) (float32, error) {
	var (
		s *HPIseries
		e error
	)

	if s, e = hd.Geo(geo); e != nil {
		return 0, e
	}

	return s.ChangeTime(dtStart, dtEnd)
}

// Copy returns a copy of hd
func (hd *HPIdata) Copy() *HPIdata {
	s := make(map[string]*HPIseries)
	for k, v := range hd.series {
		s[k] = v.Copy()
	}

	return &HPIdata{
		geoLevel: hd.geoLevel,
		series:   s,
	}
}

// Geo returns the house price data for location geo (e.g. TX).
func (hd *HPIdata) Geo(geo string) (*HPIseries, error) {
	var (
		h  *HPIseries
		ok bool
	)

	if h, ok = hd.series[geo]; !ok {
		return nil, fmt.Errorf("geo %s not found", geo)
	}

	return h, nil
}

// GeoLevel returns the aggregation level of the data (e.g. metro, nonmetro, state).
func (hd *HPIdata) GeoLevel() string {
	return hd.geoLevel
}

// Geos returns a slice of geo values in HPIdata (e.g. State postal names, MSA codes).
func (hd *HPIdata) Geos() []string {
	var geos []string
	for k := range hd.series {
		geos = append(geos, k)
	}

	return geos
}

// Last returns the date and value of the last date that was not appended.
func (hd *HPIdata) Last(geo string) (int, float32, error) {
	var (
		s *HPIseries
		e error
	)

	if s, e = hd.Geo(geo); e != nil {
		return 0, 0, e
	}

	dt, indx := s.Last()

	return dt, indx, nil
}

// Index returns the house price index for location geo (e.g. CA) at date dt (CCYYQ)
func (hd *HPIdata) Index(geo string, dt int) (float32, error) {
	var (
		s *HPIseries
		e error
	)

	if s, e = hd.Geo(geo); e != nil {
		return 0, e
	}

	return s.Index(dt)
}

// Save saves the data as a CSV.
func (hd *HPIdata) Save(localFile string) error {
	var (
		e    error
		file *os.File
	)

	if file, e = os.Create(localFile); e != nil {
		return e
	}
	defer file.Close()

	var line strings.Builder

	var geos []string
	for g := range hd.series {
		geos = append(geos, g)
	}
	sort.Strings(geos)

	hasCode := hd.series[geos[0]].geoCode != ""
	header := "geo,date,index\n"
	if hasCode {
		header = "geo,code,date,index\n"
	}

	line.WriteString(header)

	for _, g := range geos {
		v := hd.series[g]
		for j := range len(v.dates) {
			linex := fmt.Sprintf("%s,%v,%0.2f\n", v.geoName, v.dates[j], v.indx[j])
			if hasCode {
				linex = fmt.Sprintf("\"%s\",%s,%v,%0.2f\n", v.geoName, v.geoCode, v.dates[j], v.indx[j])
			}

			line.WriteString(linex)
		}
	}

	if _, e := file.WriteString(line.String()); e != nil {
		return e
	}

	return nil
}

func (hd *HPIdata) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Geo Level: %s\n", hd.geoLevel))
	s.WriteString(fmt.Sprintf("Source: %s\n\n", hd.source))
	s.WriteString("Sample Geos\n")

	cnt := 0
	for _, v := range hd.series {
		s.WriteString(v.String())
		cnt++
		if cnt == 3 {
			break
		}
	}

	return s.String()
}

// Source returns the source of the data
func (hd *HPIdata) Source() string {
	return hd.source
}

///////////////

// HPIseries holds the HPI data for a single geo value (e.g. CA).
type HPIseries struct {
	geoName  string
	geoCode  string
	dates    []int
	indx     []float32
	lastDt   int
	lastIndx float32
}

func NewHPIseries(geoName, geoCode string, dates []int, indx []float32) (*HPIseries, error) {
	if len(dates) == 0 || len(dates) != len(indx) {
		return nil, fmt.Errorf("dates and indx don't agree")
	}

	if !QtrsOK(dates) {
		return nil, fmt.Errorf("dates don't increment by quarter")
	}

	return &HPIseries{
		geoName:  geoName,
		geoCode:  geoCode,
		dates:    dates,
		indx:     indx,
		lastDt:   dates[len(dates)-1],
		lastIndx: indx[len(indx)-1],
	}, nil
}

// Append appends (dts,indx) to h. Note this does not change the values returned by Last().
func (h *HPIseries) Append(dts []int, indx []float32) error {
	// check dates are OK
	if QtrDiff(dts[0], h.lastDt) != 1 || !QtrsOK(dts) {
		return fmt.Errorf("dates don't increment by quarter")
	}

	h.dates = append(h.dates, dts...)
	h.indx = append(h.indx, indx...)

	return nil
}

// Change returns the ratio of the house price index at date dtEnd (CCYYQ) to date dtStart (CCYYQ).
func (h *HPIseries) Change(dtStart, dtEnd int) (float32, error) {
	var (
		hpiS, hpiE float32
		e          error
	)

	if hpiS, e = h.Index(dtStart); e != nil {
		return 0, e
	}

	if hpiE, e = h.Index(dtEnd); e != nil {
		return 0, e
	}

	return hpiE / hpiS, nil
}

// ChangeTime returns the house price index at date dtEnd to dtStart.
func (h *HPIseries) ChangeTime(dateStart, dateEnd time.Time) (float32, error) {
	var (
		hpiS, hpiE float32
		e          error
	)

	if hpiS, e = h.Index(ToYrQtr(dateStart)); e != nil {
		return 0, e
	}

	if hpiE, e = h.Index(ToYrQtr(dateEnd)); e != nil {
		return 0, e
	}

	return hpiE / hpiS, nil
}

// Copy returns a copy of h.
func (h *HPIseries) Copy() *HPIseries {
	dts, indx := h.Data()

	return &HPIseries{
		geoName:  h.geoName,
		geoCode:  h.geoCode,
		dates:    dts,
		indx:     indx,
		lastDt:   h.lastDt,
		lastIndx: h.lastIndx,
	}
}

// data returns the data.
func (h *HPIseries) Data() (dts []int, hpi []float32) {
	copy(dts, h.dates)
	copy(hpi, h.indx)

	return dts, hpi
}

// DateIndex returns the index in h.dates of the target date, dt. If dt is in the range of the
// data but not there, dateIndex returns the largest date less than dt.
// An error is returned if dt is outside the range of dates in h.date.
//
// -- dt -- date to find the index for, in CCYYMMDD format.
func (h *HPIseries) DateIndex(dt int) (int, error) {
	if dt > h.dates[len(h.dates)-1] {
		return -1, fmt.Errorf("date too large")
	}

	if dt < h.dates[0] {
		return -1, fmt.Errorf("date too small")
	}

	indx := sort.SearchInts(h.dates, dt)

	// decrement if not a match
	if h.dates[indx] != dt {
		indx--
	}

	return indx, nil
}

// Index returns the house price index at date dt (CCYYQ).
func (h *HPIseries) Index(dt int) (float32, error) {
	var (
		indx int
		e    error
	)

	if indx, e = h.DateIndex(dt); e != nil {
		return 0, e
	}

	return h.indx[indx], nil
}

// Name returns the series Name.  Uninteresting unless this is MSA-level data.
func (h *HPIseries) Name() string {
	return h.geoName
}

// Last returns the date and index value of the last date in the series.  This is unchanged if Append() is used.
func (h *HPIseries) Last() (dt int, indx float32) {
	return h.lastDt, h.lastIndx
}

func (h *HPIseries) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("name: %s\ngeocode: %s\n", h.geoName, h.geoCode))
	s.WriteString(fmt.Sprintf("Last Date (YrQtr): %d\n\n", h.lastDt))
	s.WriteString("YearQtr   Index\n")
	for j, dt := range h.dates {
		s.WriteString(fmt.Sprintf("%d     %0.2f\n", dt, h.indx[j]))
		if j == 4 {
			break
		}
	}

	s.WriteString("\n")

	return s.String()
}

/////////////

// Best looks through the HPI series returning the first one that has data for the geo.
// The idea is that there is a preference of the HPI series to use, say metro then nonmetro.
//
// dt - date for the lookup (CCYYQ)
//
// keys - keys to use when looking in the corresponding hpis
//
// hpis - house price index data ordered by preference
func Best(dt int, keys []string, hpis []*HPIdata) (hpi float32, geoLevel string, e error) {
	if len(keys) != len(hpis) || len(hpis) == 0 {
		return 0, "", fmt.Errorf("invalid series")
	}

	for j, s := range hpis {
		if indx, e := s.Index(keys[j], dt); e == nil {
			return indx, s.geoLevel, nil
		}
	}

	return 0, "", fmt.Errorf("geo/dt not found in Best")
}

// ToDate converts a CCYYQ int to a date. The date returned is the first day of the first
// month of the quarter
func ToTime(dt int) (time.Time, error) {
	yr := dt / 10
	qtr := dt - 10*yr

	if yr < 1960 || yr > 2060 || qtr < 1 || qtr > 4 {
		return time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), fmt.Errorf("illegal date conversion")
	}

	month := time.Month(1 + 3*(qtr-1))

	return time.Date(yr, month, 1, 0, 0, 0, 0, time.UTC), nil
}

// ToYrQTR converts a date to a CCYYQ int
func ToYrQtr(dt time.Time) int {
	yr := dt.Year()
	mon := int(dt.Month())
	qtr := 1 + (mon-1)/3

	return 10*yr + qtr
}

// NextQtr advances dt (CCYYQ) by 1 quarter
func NextQtr(dt int) int {
	yr := dt / 10
	qtr := dt - 10*yr

	if yr < 1960 || qtr < 1 || qtr > 4 {
		panic(fmt.Errorf("illegal date: %v", dt))
	}

	qtr++
	if qtr == 5 {
		qtr = 1
		yr++
	}

	return 10*yr + qtr
}

// QtrDiff returns the number of quarters between dt0 (CCYYQ) and dt1 (CCYYQ)
func QtrDiff(dt0, dt1 int) int {
	if dt1 < dt0 {
		dt1, dt0 = dt0, dt1 //TODO: check
	}

	yr0, yr1 := dt0/10, dt1/10
	qtr0, qtr1 := dt0-10*yr0, dt1-10*yr1

	return 4*(yr1-yr0) + qtr1 - qtr0
}

// QtrsOK checks that the elements of dt increment 1 quarter at a time.
func QtrsOK(dt []int) bool {
	for j := 1; j < len(dt); j++ {
		if QtrDiff(dt[j-1], dt[j]) != 1 {
			return false
		}
	}

	return true
}

////////////

// geoLevel returns the geographic level of the data (e.g. metro, us,..)
func geoLevel(header string) string {
	header = strings.ToLower(header)

	if strings.Contains(header, "three-digit zip") {
		return "zip3"
	}

	if strings.Contains(header, "metropolitan areas") {
		return "metro"
	}

	if strings.Contains(header, "not in metropolitan statistical areas") {
		return "nonmetro"
	}

	if strings.Contains(header, "states and the district of columbia") {
		return "state"
	}

	if strings.Contains(header, "census divisions") {
		return "us"
	}

	if strings.Contains(header, "puerto rico") {
		return "pr"
	}

	if strings.Contains(header, "manufactured homes") {
		return "mh"
	}

	return "unknown"
}

func in[T comparable](needle T, haystack []T) bool {
	for _, s := range haystack {
		if needle == s {
			return true
		}
	}

	return false
}

func URLs(series string) string {
	series = strings.ToLower(series)

	switch series {
	case "us":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_us_and_census.xlsx"
	case "state":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_state.xlsx"
	case "metro":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_metro.xlsx"
	case "nonmetro":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_nonmetro.xlsx"
	case "pr":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_pr.xlsx"
	case "zip3":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_3zip.xlsx"
	case "mh":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_mh.xlsx"
	default:
		panic(fmt.Errorf("unrecognized series in dataURL: %s", series))
	}
}
