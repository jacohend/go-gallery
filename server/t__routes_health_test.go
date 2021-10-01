package server

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/mikeydub/go-gallery/util"
)

func TestHealthcheck(t *testing.T) {
	assert := setupTest(t)

	resp, err := http.Get(fmt.Sprintf("%s/health", tc.serverURL))
	assert.Nil(err)
	assertValidJSONResponse(assert, resp)

	body := healthcheckResponse{}
	util.UnmarshallBody(&body, resp.Body)
	assert.Equal("gallery operational", body.Message)
	assert.Equal("local", body.Env)
}
