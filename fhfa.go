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

// HPIgeo holds the HPI data for a single geo (state, MSA ...)
type HPIgeo struct {
	geoName string
	geoCode string
	dates   []int
	hpi     []float64
}

type HPIdata struct {
	geoLevel string
	series   map[string]*HPIgeo
}

// DateIndex returns the index in h.dates of the target date, dt. If dt is in the range of the
// data but not there, DateIndex returns the largest date less than dt.
// An error is returned if dt is outside the range of dates in h.Date.
//
// -- dt -- date to find the index for, in CCYYMMDD format.
func (h *HPIgeo) DateIndex(dt int) (int, error) {
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

func (h *HPIgeo) GeoCode() string {
	return h.geoCode
}

func (h *HPIgeo) GeoName() string {
	return h.geoName
}

func (h *HPIgeo) HPI(dt int) (float64, error) {
	var (
		indx int
		e    error
	)

	if indx, e = h.DateIndex(dt); e != nil {
		return 0, e
	}

	return h.hpi[indx], nil
}

// TODO
// Save saves the data as a CSV
func (d *HPIdata) Save(localFile string) error {

	return nil
}

func (d *HPIdata) Geo() string {
	return d.geoLevel
}

func (d *HPIdata) HPI(geo string, dt int) (float64, error) {
	if series, ok := d.series[geo]; ok {
		return series.HPI(dt)
	}

	return 0, fmt.Errorf("series not found")
}

// Best looks through the HPI series in order returning the first one that has data for the geo.
// The idea is that there is a preference of the HPI series to use, say msa, non-msa, state, Puerto Rico.
// So that if the location is in an MSA, that is used. If not, the non-msa is used, if not the State is
// used and if not, Puerto Rico is used.  An error occurs if the geo is in none of these.
//
// dt - date for the lookup
//
// keys - keys to use when looking in the corresponding hpis
//
// hpis - HPI series by geos
func Best(dt int, keys []string, hpis []HPIdata) (hpi float64, geoLevel string, e error) {
	if len(keys) != len(hpis) || len(hpis) == 0 {
		return 0, "", fmt.Errorf("invalid series")
	}

	for j, s := range hpis {
		if hpi, e := s.HPI(keys[j], dt); e == nil {
			return hpi, keys[j], nil
		}
	}

	return 0, "", fmt.Errorf("geo not found in Best")
}

// Fetch pulls the FHFA XLSX file and saves it locally
//
// source - one of zip3, msa, nonmas, state, us, pr, mh
//
// xlsxFile - file to save the data to.
func Fetch(source, xlsxFile string) error {
	var (
		url string
		e   error
	)

	if url, e = urls(source); e != nil {
		return nil
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

	r, _ := client.Do(req)
	defer func() { _ = r.Body.Close() }()

	var body []byte

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

// doRow converts row values to fields required for HPIgeo
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

// Parse works through the XLSX file, returning the data as an HPIdata object
//
// - source - either a file name or one of: zip3, msa, nonmsa, state, us, pr, mh
func Parse(source string) (*HPIdata, error) {
	geo := ""
	if in(strings.ToLower(source), []string{"zip3", "msa", "nonmsa", "state", "us", "pr", "mh"}) {
		tmpFile := fmt.Sprintf("%s/hpi.xlsx", os.TempDir())
		if e := Fetch(source, tmpFile); e != nil {
			return nil, e
		}
		defer os.Remove(tmpFile)

		geo = source
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
		geoLevel: geo,
		series:   make(map[string]*HPIgeo),
	}

	var series *HPIgeo
	hasGeoCode := 0
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}

		// find the start of the data
		if strings.ToLower(row[1]) == "year" {
			inData = true
			continue
		}

		// MSA data
		if strings.ToLower(row[2]) == "year" {
			inData = true
			hasGeoCode = 1

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

		if geo, yrqtr, index = doRow(row, hasGeoCode); index == 0 {
			continue
		}

		if geo != lastGeo {
			lastGeo = geo
			key := row[hasGeoCode]
			geoCode := ""
			if hasGeoCode == 1 {
				geoCode = row[1]
			}

			series = &HPIgeo{
				geoName: geo,
				geoCode: geoCode,
			}

			hd.series[key] = series
		}

		series.dates = append(series.dates, yrqtr)
		series.hpi = append(series.hpi, index)
	}

	return hd, nil
}

// ToYrQTR converts a date to a CCYYQ int
func ToYrQtr(dt time.Time) int {
	yr := dt.Year()
	mon := int(dt.Month())
	qtr := 1 + (mon-1)/3

	return 10*yr + qtr
}

// urls returns the url of the house price index at the requested geo.
//
// series - series requested. Options are "us", "state", "msa", "nonmsa", "pr" and "mg"
func urls(series string) (string, error) {
	series = strings.ToLower(series)

	switch series {
	case "us":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_us_and_census.xlsx", nil
	case "state":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_state.xlsx", nil
	case "msa":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_metro.xlsx", nil
	case "nonmsa":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_nonmetro.xlsx", nil
	case "pr":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_pr.xlsx", nil
	case "zip3":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_3zip.xlsx", nil
	case "mh":
		return "https://www.fhfa.gov/hpi/download/quarterly_datasets/hpi_at_mh.xlsx", nil
	default:
		return "", fmt.Errorf("unrecognized series in dataURL: %s", series)
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
