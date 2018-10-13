package server

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/kyleterry/jot/auth"
	"github.com/kyleterry/jot/config"
	"github.com/kyleterry/jot/jot"
	"github.com/kyleterry/jot/testutil"
	"github.com/stretchr/testify/require"
)

const TestMasterPassword = "test password"

func WithTestServer(t *testing.T, fn func(*httptest.Server)) {
	tmp, _, cleanup := testutil.NewTempFilesystem(t)
	defer cleanup()

	cfg := &config.Config{
		SeedFile:       filepath.Join(tmp, "seed"),
		MasterPassword: "test master password",
		DataDir:        tmp,
	}

	seed, err := auth.MakeSeed(TestMasterPassword)
	require.NoError(t, err)

	manager := auth.NewPasswordManager(TestMasterPassword, seed)
	store, err := jot.NewStore(cfg, &manager)
	require.NoError(t, err)
	s := New(cfg, store, &manager)

	ts := httptest.NewServer(s)
	defer ts.Close()

	fn(ts)
}

func TestJotServer(t *testing.T) {
	var cases = []struct {
		payload       string
		updatePayload string
		ensure        func(*testing.T, *http.Response)
	}{
		{"payload", "updated payload", nil},
	}

	WithTestServer(t, func(ts *httptest.Server) {
		client := ts.Client()

		for _, c := range cases {
			bufPayload := bytes.NewBufferString(c.payload)
			bufUpdatePayload := bytes.NewBufferString(c.updatePayload)

			var (
				jotURL      *url.URL
				jotPassword string
				jotETag     string
			)

			t.Run("POST", func(t *testing.T) {
				resp, err := client.Post(ts.URL, "text/plain", bufPayload)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, resp.StatusCode)

				require.NotEmpty(t, resp.Header.Get("Jot-Password"))
				jotPassword = resp.Header.Get("Jot-Password")

				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				jotURL, err = url.Parse(string(b))
				require.NoError(t, err)
			})

			t.Run("GET after POST", func(t *testing.T) {
				resp, err := client.Get(jotURL.String())
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, DefaultContentType, resp.Header.Get("Content-Type"))

				require.NotEmpty(t, resp.Header.Get("ETag"), "etag is missing")
				jotETag = resp.Header.Get("ETag")

				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, c.payload, string(b))
			})

			t.Run("GET with current precondition", func(t *testing.T) {
				req, err := http.NewRequest("GET", jotURL.String(), nil)
				require.NoError(t, err)

				req.Header.Set("if-none-match", jotETag)

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusNotModified, resp.StatusCode)
			})

			t.Run("GET with expired precondition", func(t *testing.T) {
				req, err := http.NewRequest("GET", jotURL.String(), nil)
				require.NoError(t, err)

				// expired
				req.Header.Set("if-none-match", "2001-01-01T00:00:00Z")

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, DefaultContentType, resp.Header.Get("Content-Type"))

				require.NotEmpty(t, resp.Header.Get("ETag"), "etag is missing")
				jotETag = resp.Header.Get("ETag")

				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, c.payload, string(b))
			})

			t.Run("GET with malformed precondition", func(t *testing.T) {
				req, err := http.NewRequest("GET", jotURL.String(), nil)
				require.NoError(t, err)

				// expired
				req.Header.Set("if-none-match", "this is malformed")

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, DefaultContentType, resp.Header.Get("Content-Type"))

				require.NotEmpty(t, resp.Header.Get("ETag"), "etag is missing")
				jotETag = resp.Header.Get("ETag")

				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, c.payload, string(b))
			})

			t.Run("PUT", func(t *testing.T) {
				req, err := http.NewRequest("PUT", jotURL.String(), bufUpdatePayload)
				require.NoError(t, err)

				// make sure we can update if the precondition modified date matches
				req.Header.Set("if-match", jotETag)

				q := req.URL.Query()
				q.Add("password", jotPassword)
				req.URL.RawQuery = q.Encode()

				// we want to make sure the correct reponse redirect happens, so we
				// tell the client not to follow redirects so we can check the actual
				// response received.
				client := ts.Client()
				client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				}

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusSeeOther, resp.StatusCode)
			})

			t.Run("GET after PUT", func(t *testing.T) {
				resp, err := client.Get(jotURL.String())
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, DefaultContentType, resp.Header.Get("Content-Type"))

				require.NotEmpty(t, resp.Header.Get("ETag"), "etag is missing")
				// then set the updated etag
				jotETag = resp.Header.Get("ETag")

				b, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, c.updatePayload, string(b))
			})

			t.Run("PUT failed with old precondition", func(t *testing.T) {
				req, err := http.NewRequest("PUT", jotURL.String(), bufUpdatePayload)
				require.NoError(t, err)

				// make sure we can update if the precondition modified date matches
				req.Header.Set("if-match", "2001-01-01T00:00:00Z")

				q := req.URL.Query()
				q.Add("password", jotPassword)
				req.URL.RawQuery = q.Encode()

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusPreconditionFailed, resp.StatusCode)
			})

			t.Run("PUT with the wrong password", func(t *testing.T) {
				req, err := http.NewRequest("PUT", jotURL.String(), bufUpdatePayload)
				require.NoError(t, err)

				// make sure we can update if the precondition modified date matches
				req.Header.Set("if-match", jotETag)

				q := req.URL.Query()
				q.Add("password", "wrongpassword")
				req.URL.RawQuery = q.Encode()

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})

			t.Run("DELETE", func(t *testing.T) {
				req, err := http.NewRequest("DELETE", jotURL.String(), nil)
				require.NoError(t, err)

				q := req.URL.Query()
				q.Add("password", jotPassword)
				req.URL.RawQuery = q.Encode()

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusNoContent, resp.StatusCode)
			})

			t.Run("GET after DELETE", func(t *testing.T) {
				resp, err := client.Get(jotURL.String())
				require.NoError(t, err)
				require.Equal(t, http.StatusNotFound, resp.StatusCode)
			})

			t.Run("GET general 404", func(t *testing.T) {
				newJotURL := *jotURL
				newJotURL.Path = "wrongwrongwrong"

				resp, err := client.Get(newJotURL.String())
				require.NoError(t, err)
				require.Equal(t, http.StatusNotFound, resp.StatusCode)
			})
		}
	})
}
