
## package fhfa

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/fhfa)](https://goreportcard.com/report/github.com/invertedv/fhfw)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/fhfa?tab=overview)

### Purpose

The purpose of this package is to make the FHFA house price indices available in a format that is fast and
readily usable within Go. 

### Details

This package will pull the data from the FHFA web site and will find the HPI for individual geos and dates. The data can also be read from the FHFA XLSX files directly if they have already been downloaded. The data is loaded into a map of type HPIdata.  The key to the map is the geo name (or code, in the case of MSAs). The value of the map is the struct HPIgeo.  The HPI method of HPIgeo returns the house price index for a given date (specified as an int in the format CCYYQ).


The normal flow is:

1. If pulling the data from the web:

    a. Get the URL of the series you want by calling URLs().

    b. Use Fetch() to pull the data.

    c. Use Save() to save as an XLSX file
2. Use Parse() to process the XLSX into an HPIdata map.
3. Use the HPI() of any HPIdata series to find the HPI for any date.


The indices are available [here](https://www.fhfa.gov/data/hpi/datasets?tab=quarterly-data).
