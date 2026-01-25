package fhfa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {

	//	data, e := Fetch("")
	//	assert.Nil(t, e)

	//	e = Save(data, "/home/will/tmp/test.xlsx")
	//	assert.Nil(t, e)

	hd, e := Parse("/home/will/Downloads/hpi_at_metro.xlsx")
	_=hd
	assert.Nil(t, e)
}
