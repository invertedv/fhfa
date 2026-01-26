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

type HPIdata map[string]*HPIgeo

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

// Save saves the data as a CSV
func (d HPIdata) Save(localFile string) error {

	return nil
}

// Best looks through the HPI series in order returning the first one that has data for the geo.
// The idea is that there is a preference of the HPI series to use, say msa, non-msa, state, Puerto Rico.
// So that if the location is in an MSA, that is used. If not, the non-msa is used, if not the State is
// used and if not, Puerto Rico is used.  An error occurs if the geo is in none of these.
//
// keys - keys to use when looking in the corresponding hpis
// hpis - HPI series
func Best(keys []string, hpis []HPIdata) (HPIdata, error) {
	if len(keys) != len(hpis) || len(hpis) == 0 {
		return nil, fmt.Errorf("invalid series")
	}
	for j, hpi := range hpis {
		if _, ok := hpi[keys[j]]; ok {
			return hpi, nil
		}
	}

	return nil, fmt.Errorf("geo not found in Best")
}

// Fetch returns the FHFA XLSX sheet as a string
//
// source - either the url from urls() or a file name
func Fetch(source string) (data string, e error) {
	if !strings.Contains(source, "http") {
		var (
			e     error
			file  *os.File
			bytes []byte
		)

		defer file.Close()
		if file, e = os.Open(source); e != nil {
			return "", e
		}

		if bytes, e = io.ReadAll(file); e != nil {
			return "", e
		}

		return string(bytes), nil
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", source, nil)

	r, _ := client.Do(req)
	defer func() { _ = r.Body.Close() }()

	var body []byte
	if body, e = io.ReadAll(r.Body); e != nil {
		return "", e
	}

	return string(body), nil
}

// Parse works through the XLSX file, returning the data as an HPIdata object
//
// - xlsxFile - location of the XLSX file
func Parse(xlsxFile string) (HPIdata, error) {
	xlr, e := excelize.OpenFile(xlsxFile)
	if e != nil {
		return nil, e
	}
	defer xlr.Close()

	rows, _ := xlr.GetRows(xlr.GetSheetName(0))
	inData := false
	lastGeo := ""
	hd := make(HPIdata)
	var series *HPIgeo
	hasGeoCode := 0
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}

		// find the start of the data
		if strings.ToLower(row[1]) == "year" || strings.ToLower(row[2]) == "year" {
			inData = true
			// the MSA file has an extra column for MSA value
			if strings.ToLower(row[2]) == "year" {
				hasGeoCode = 1
			}

			continue
		}

		if !inData {
			continue
		}

		var (
			geo       string
			year, qtr int64
			index     float64
			e         error
		)
		geo = row[0]

		if year, e = strconv.ParseInt(row[1+hasGeoCode], 10, 64); e != nil {
			return nil, e
		}

		if qtr, e = strconv.ParseInt(row[2+hasGeoCode], 10, 64); e != nil {
			return nil, e
		}

		if index, e = strconv.ParseFloat(row[3+hasGeoCode], 64); e != nil {
			// some rows have no index value
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

			hd[key] = series
		}

		series.dates = append(series.dates, 10*int(year)+int(qtr))
		series.hpi = append(series.hpi, index)
	}

	return hd, nil
}

// Save saves the XLSX to a file.
//
// - data -- string respresentation of the FHFA XLSX as pulled by Fetch()
//
// - localFile -- file to create.
func Save(data, localFile string) error {
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

// ToYrQTR converts a date to a CCYYQ int
func ToYrQtr(dt time.Time) int {
	yr := dt.Year()
	mon := int(dt.Month())
	qtr := 1 + (mon-1)/3

	return 10*yr + qtr
}

// URLs returns the url of the house price index at the requested geo.
//
// series - series requested. Options are "us", "state", "msa", "nonmsa", "pr" and "mg"
func URLs(series string) (string, error) {
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
