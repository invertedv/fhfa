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

The XLSX format is chosen since not all of the data is available as a CSV but all are as XLSX.
*/
package fhfa

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// also, tests for Change

// HPIdata manages the data for a single geo level (e.g. metro, state)
type HPIdata struct {
	geoLevel string
	series   map[string]*hpiSeries
}

// Change returns the ratio of the house price index at dtEnd to dtStart
func (d *HPIdata) Change(geo string, dtStart, dtEnd int) (float64, error) {
	if series, ok := d.series[geo]; ok {
		return series.change(dtStart, dtEnd)
	}

	return 0, fmt.Errorf("series not found")
}

// Data returns the house price series (dates, index) for geo.
func (d *HPIdata) Data(geo string) (dts []int, index []float64, e error) {
	if series, ok := d.series[geo]; ok {
		dts, index = series.data()
		return dts, index, nil
	}

	return nil, nil, fmt.Errorf("series not found")

}

// GeoLevel returns the aggregation level of the data (e.g. metro, non-metro)
func (d *HPIdata) GeoLevel() string {
	return d.geoLevel
}

// Name returns the name of the geo. Meaninful only for MSAs.
func (d *HPIdata) Name(geo string) (string, error){
		if series, ok := d.series[geo]; ok {
		return series.name(), nil
	}

	return "", fmt.Errorf("series not found")
}

// Geos returns a slice of geo values in HPIdata (e.g. State postal names, MSA codes)
func (d *HPIdata) Geos() []string {
	var geos []string
	for k := range d.series {
		geos = append(geos, k)
	}

	return geos
}

// Index returns the house price index for the geo at date dt
//
// geo - geo level (e.g. metro, nonmetro, mh)
//
// dt  - date in CCYYQ format
func (d *HPIdata) Index(geo string, dt int) (float64, error) {
	if series, ok := d.series[geo]; ok {
		return series.index(dt)
	}

	return 0, fmt.Errorf("series not found")
}

// Save saves the data as a CSV
func (d *HPIdata) Save(localFile string) error {
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
	for g := range d.series {
		geos = append(geos, g)
	}
	sort.Strings(geos)

	hasCode := d.series[geos[0]].geoCode != ""
	header := "geo,date,index\n"
	if hasCode {
		header = "geo,code,date,index\n"
	}

	line.WriteString(header)

	for _, g := range geos {
		v := d.series[g]
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

// hpiSeries holds the HPI data for a single geo (state, metro (msa) ...)
type hpiSeries struct {
	geoName string
	geoCode string
	dates   []int
	indx    []float64
}

// dateIndex returns the index in h.dates of the target date, dt. If dt is in the range of the
// data but not there, dateIndex returns the largest date less than dt.
// An error is returned if dt is outside the range of dates in h.date.
//
// -- dt -- date to find the index for, in CCYYMMDD format.
func (h *hpiSeries) dateIndex(dt int) (int, error) {
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

func (h *hpiSeries) change(dtStart, dtEnd int) (float64, error) {
	var (
		hpiS, hpiE float64
		e          error
	)

	if hpiS, e = h.index(dtStart); e != nil {
		return 0, e
	}

	if hpiE, e = h.index(dtEnd); e != nil {
		return 0, e
	}

	return hpiE / hpiS, nil
}

// data returns the data series for the geo
func (h *hpiSeries) data() (dts []int, hpi []float64) {
	copy(dts, h.dates)
	copy(hpi, h.indx)

	return dts, hpi
}

func (h *hpiSeries) index(dt int) (float64, error) {
	var (
		indx int
		e    error
	)

	if indx, e = h.dateIndex(dt); e != nil {
		return 0, e
	}

	return h.indx[indx], nil
}

func (h *hpiSeries) name() string{
	return h.geoName
}


// Best looks through the HPI series returning the first one that has data for the geo.
// The idea is that there is a preference of the HPI series to use, say metro then non-metro.
//
// dt - date for the lookup
//
// keys - keys to use when looking in the corresponding hpis
//
// hpis - house price index data ordered by preference
func Best(dt int, keys []string, hpis []*HPIdata) (hpi float64, geoLevel string, e error) {
	if len(keys) != len(hpis) || len(hpis) == 0 {
		return 0, "", fmt.Errorf("invalid series")
	}

	for j, s := range hpis {
		if hpi, e := s.Index(keys[j], dt); e == nil {
			return hpi, s.geoLevel, nil
		}
	}

	return 0, "", fmt.Errorf("geo not found in Best")
}

// Fetch pulls the FHFA XLSX file and saves it locally
//
// source - one of zip3, metro, nonmetro, state, us, pr, mh
//
// xlsxFile - file to create
func Fetch(source, xlsxFile string) error {
	url := urls(source)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

	r, _ := client.Do(req)
	defer func() { _ = r.Body.Close() }()

	var (
		body []byte
		e    error
	)

	if body, e = io.ReadAll(r.Body); e != nil {
		return e
	}

	return save(string(body), xlsxFile)
}

// Save saves the XLSX to a file.
//
// - data -- string respresentation of the FHFA XLSX as pulled by Fetch()
//
// - localFile -- file to create.
func save(data, localFile string) error {
	var (
		e    error
		file *os.File
	)

	if file, e = os.Create(localFile); e != nil {
		return e
	}
	defer file.Close()

	_, e = file.WriteString(data)

	return e
}

// Load loads HPIdata
//
//   - source - either a file name or one of: zip3, metros, nonmetro, state, us, pr, mh. The last options pull
//     the data from the FHFA web site.
func Load(source string) (*HPIdata, error) {
	// fetch from web?
	if in(strings.ToLower(source), []string{"zip3", "metro", "nonmetro", "state", "us", "pr", "mh"}) {
		tmpFile := fmt.Sprintf("%s/hpi.xlsx", os.TempDir())
		if e := Fetch(source, tmpFile); e != nil {
			return nil, e
		}
		defer os.Remove(tmpFile)

		source = tmpFile
	}

	xlr, e := excelize.OpenFile(source)
	if e != nil {
		return nil, e
	}
	defer xlr.Close()

	rows, _ := xlr.GetRows(xlr.GetSheetName(0))
	inData := false
	lastGeo := ""

	hd := &HPIdata{
		geoLevel: geoLevel(rows[0][0]),
		series:   make(map[string]*hpiSeries),
	}

	var series *hpiSeries
	hasGeoCode := 0
	if hd.geoLevel == "metro" {
		hasGeoCode = 1
	}

	for _, row := range rows {
		if len(row) < 4 {
			continue
		}

		// find the start of the data
		if !inData && (strings.ToLower(row[1]) == "year" || strings.ToLower(row[2]) == "year")  {
			inData = true
			continue
		}

		if !inData {
			continue
		}

		var (
			geo   string
			yrqtr int
			index float64
		)

		// some index values are missing
		if geo, yrqtr, index = doRow(row, hasGeoCode); index == 0 {
			continue
		}

		// New geo?
		if geo != lastGeo {
			lastGeo = geo
			key := row[hasGeoCode]

			series = &hpiSeries{
				geoName: geo,
				geoCode: row[hasGeoCode],
			}

			hd.series[key] = series
		}

		series.dates = append(series.dates, yrqtr)
		series.indx = append(series.indx, index)
	}

	return hd, nil
}

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

// doRow converts excelize row values to fields required for hpiSeries
func doRow(row []string, offset int) (geo string, yrqtr int, indx float64) {
	geo = row[0]
	var (
		year, qtr int64
		e         error
	)

	if year, e = strconv.ParseInt(row[1+offset], 10, 64); e != nil {
		panic(e)
	}

	if qtr, e = strconv.ParseInt(row[2+offset], 10, 64); e != nil {
		panic(e)
	}

	yrqtr = 10*int(year) + int(qtr)

	if indx, e = strconv.ParseFloat(row[3+offset], 64); e != nil {
		// some rows have no index value
		return "", 0, 0
	}

	return geo, yrqtr, indx
}

// ToYrQTR converts a date to a CCYYQ int
func ToYrQtr(dt time.Time) int {
	yr := dt.Year()
	mon := int(dt.Month())
	qtr := 1 + (mon-1)/3

	return 10*yr + qtr
}

func urls(series string) string {
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

func in[T comparable](needle T, haystack []T) bool {
	for _, s := range haystack {
		if needle == s {
			return true
		}
	}

	return false
}
