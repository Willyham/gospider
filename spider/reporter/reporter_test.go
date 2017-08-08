package reporter

import (
	"bytes"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport(t *testing.T) {
	root, err := url.Parse("http://willdemaine.co.uk")
	require.NoError(t, err)

	page1, err := url.Parse("http://willdemaine.co.uk/page1")
	require.NoError(t, err)

	page2, err := url.Parse("http://willdemaine.co.uk/page2")
	require.NoError(t, err)

	r := NewHTML()
	r.Add(root, []*url.URL{page1, page2}, []string{"foo.img"})
	r.Add(page1, []*url.URL{page2}, []string{})
	r.Add(page2, []*url.URL{}, []string{"bar.img"})

	buf := bytes.NewBuffer(nil)
	err = r.Report(buf)
	assert.NoError(t, err)

	fmt.Println(buf)
}
