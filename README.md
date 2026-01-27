
## package fhfa

[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/fhfa)](https://goreportcard.com/report/github.com/invertedv/fhfw)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/fhfa?tab=overview)

### Purpose

The purpose of this package is to make the FHFA house price indices available in a format that is fast and
readily usable within Go. 

### Details

This package will pull the data from the FHFA web site and will find the HPI for individual geos and dates. The data can also be read from the FHFA XLSX files directly if they have already been downloaded. The data is loaded into a stuct of type HPIdata. The HPI method of HPIdata returns the house price index for a given date (specified as an int in the format CCYYQ) and geo location.


The indices are available [here](https://www.fhfa.gov/data/hpi/datasets?tab=quarterly-data).


The XLSX format is chosen since not all of the data is available as a CSV but all are as XLSX.
