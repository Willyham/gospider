package parser

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestByTokenRealistic(t *testing.T) {
	body, err := ioutil.ReadFile("./testdata/willdemaine.ghost.io.html")
	require.NoError(t, err)

	_, err = ByToken(body)
	assert.NoError(t, err)
}

func TestByTokenInvalidInputs(t *testing.T) {
	inputs := [][]byte{[]byte("><><><"), []byte(nil)}
	for _, input := range inputs {
		t.Run(string(input), func(t *testing.T) {
			_, err := ByToken(input)
			assert.NoError(t, err)
		})
	}
}

func TestMissingAttrs(t *testing.T) {
	body, err := ioutil.ReadFile("./testdata/missingAttrs.html")
	require.NoError(t, err)

	results, err := ByToken(body)
	assert.NoError(t, err)
	assert.Len(t, results.Assets, 0)
	assert.Len(t, results.Links, 0)
}
