package server

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudflare/gokey"
	"github.com/kyleterry/jot/pkg/auth"
	"github.com/kyleterry/jot/pkg/config"
	"github.com/kyleterry/jot/pkg/id"
	pkgimage "github.com/kyleterry/jot/pkg/image"
	imagefs "github.com/kyleterry/jot/pkg/image/backend/filesystem"
	"github.com/kyleterry/jot/pkg/jot"
	"github.com/kyleterry/jot/pkg/store"
	"github.com/kyleterry/jot/pkg/testutil"
	"github.com/stretchr/testify/require"
)

const TestMasterPassword = "test password"

func WithTestServer(t *testing.T, fn func(*httptest.Server)) {
	tmp, fs, cleanup := testutil.NewTextFilesystem(t)
	defer cleanup()

	seedPath := filepath.Join(tmp, "seed")
	seedBytes, err := gokey.GenerateEncryptedKeySeed(TestMasterPassword)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(seedPath, seedBytes, 0o600))

	cfg := &config.Config{
		SeedFileLocation: auth.SeedFileLocation(seedPath),
		MasterPassword:   auth.MasterPassword(TestMasterPassword),
		DataDir:          config.DataDir(tmp),
	}

	spec := auth.DefaultSpec()
	sf, err := auth.NewSeedFile(cfg.MasterPassword, cfg.SeedFileLocation, spec)
	require.NoError(t, err)
	pm := auth.NewPasswordManager(sf)

	idm, err := id.NewIDManager()
	require.NoError(t, err)

	jotOpts := &store.Options{
		PasswordManager: pm,
		IDManager:       idm,
	}
	textStore := jot.NewStore(fs, jotOpts)

	jr := NewJotHandler(cfg, textStore, pm)
	ir := NewImageHandler(cfg, nil, pm)

	srv := New(cfg, jr, ir)

	ts := httptest.NewServer(srv)
	defer ts.Close()

	fn(ts)
}

func TestJotServer(t *testing.T) {
	cases := []struct {
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
				jotURL          *url.URL
				jotPassword     string
				jotETag         string
				jotLastModified string
			)

			t.Run("POST", func(t *testing.T) {
				resp, err := client.Post(ts.URL+"/txt", "text/plain", bufPayload)
				require.NoError(t, err)
				require.Equal(t, http.StatusCreated, resp.StatusCode)

				require.NotEmpty(t, resp.Header.Get("Jot-Password"))
				jotPassword = resp.Header.Get("Jot-Password")

				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				u := strings.TrimSuffix(string(b), "\n")

				jotURL, err = url.Parse(u)
				require.NoError(t, err)
			})

			t.Run("GET after POST", func(t *testing.T) {
				resp, err := client.Get(jotURL.String())
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, DefaultContentType, resp.Header.Get("Content-Type"))

				require.NotEmpty(t, resp.Header.Get("ETag"), "etag is missing")
				jotETag = resp.Header.Get("ETag")
				jotLastModified = resp.Header.Get("Last-Modified")

				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, c.payload, string(b))
			})

			t.Run("GET with modified check", func(t *testing.T) {
				req, err := http.NewRequest("GET", jotURL.String(), nil)
				require.NoError(t, err)

				req.Header.Set("if-modified-since", jotLastModified)

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusNotModified, resp.StatusCode)
			})

			t.Run("GET with old last modified", func(t *testing.T) {
				req, err := http.NewRequest("GET", jotURL.String(), nil)
				require.NoError(t, err)

				req.Header.Set("if-modified-since", "Thu, 17 Sep 2020 20:27:38 GMT")

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)

				b, err := io.ReadAll(resp.Body)
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

				b, err := io.ReadAll(resp.Body)
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

				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, c.payload, string(b))
			})

			t.Run("PUT", func(t *testing.T) {
				req, err := http.NewRequest("PUT", jotURL.String(), bufUpdatePayload)
				require.NoError(t, err)

				// make sure we can update if the precondition modified date matches
				req.Header.Set("if-match", jotETag)

				req.SetBasicAuth("", jotPassword)

				// we want to make sure the correct reponse redirect happens, so we
				// tell the client not to follow redirects so we can check the actual
				// response received.
				client = ts.Client()
				client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				}

				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Equal(t, http.StatusSeeOther, resp.StatusCode, string(body), jotURL.String())
			})

			t.Run("GET after PUT", func(t *testing.T) {
				resp, err := client.Get(jotURL.String())
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode)
				require.Equal(t, DefaultContentType, resp.Header.Get("Content-Type"))

				require.NotEmpty(t, resp.Header.Get("ETag"), "etag is missing")
				// then set the updated etag
				jotETag = resp.Header.Get("ETag")

				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, c.updatePayload, string(b))
			})

			t.Run("PUT failed with old precondition", func(t *testing.T) {
				req, err := http.NewRequest("PUT", jotURL.String(), bufUpdatePayload)
				require.NoError(t, err)

				// make sure we can update if the precondition modified date matches
				req.Header.Set("if-match", "2001-01-01T00:00:00Z")

				req.SetBasicAuth("", jotPassword)

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusPreconditionFailed, resp.StatusCode)
			})

			t.Run("PUT with the wrong password", func(t *testing.T) {
				req, err := http.NewRequest("PUT", jotURL.String(), bufUpdatePayload)
				require.NoError(t, err)

				// make sure we can update if the precondition modified date matches
				req.Header.Set("if-match", jotETag)

				req.SetBasicAuth("", "wrongpassword")

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
			})

			t.Run("DELETE", func(t *testing.T) {
				req, err := http.NewRequest("DELETE", jotURL.String(), nil)
				require.NoError(t, err)

				req.SetBasicAuth("", jotPassword)

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

func WithImageTestServer(t *testing.T, fn func(*httptest.Server)) {
	t.Helper()

	tmp, err := os.MkdirTemp("", "github.com-kyleterry-jot-img")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	seedPath := filepath.Join(tmp, "seed")
	seedBytes, err := gokey.GenerateEncryptedKeySeed(TestMasterPassword)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(seedPath, seedBytes, 0o600))

	cfg := &config.Config{
		SeedFileLocation: auth.SeedFileLocation(seedPath),
		MasterPassword:   auth.MasterPassword(TestMasterPassword),
		DataDir:          config.DataDir(tmp),
	}

	spec := auth.DefaultSpec()
	sf, err := auth.NewSeedFile(cfg.MasterPassword, cfg.SeedFileLocation, spec)
	require.NoError(t, err)
	pm := auth.NewPasswordManager(sf)

	idm, err := id.NewIDManager()
	require.NoError(t, err)

	storeOpts := &store.Options{
		PasswordManager: pm,
		IDManager:       idm,
	}

	imgBackend, err := imagefs.New(&imagefs.Options{StorageDir: cfg.DataDir})
	require.NoError(t, err)

	imgStore := pkgimage.NewStore(imgBackend, storeOpts)

	// txt is not exercised in image tests; lazy closures in NewJotHandler won't panic
	jr := NewJotHandler(cfg, nil, pm)
	ir := NewImageHandler(cfg, imgStore, pm)

	srv := New(cfg, jr, ir)

	ts := httptest.NewServer(srv)
	defer ts.Close()

	fn(ts)
}

// minimalPNG returns the bytes of a 1×1 red PNG image.
func minimalPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	return buf.Bytes()
}

// imageMultipart builds a multipart/form-data body with a single "images" file field.
func imageMultipart(t *testing.T, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("images", filename)
	require.NoError(t, err)

	_, err = fw.Write(data)
	require.NoError(t, err)

	require.NoError(t, w.Close())

	return &buf, w.FormDataContentType()
}

func TestImageServer(t *testing.T) {
	WithImageTestServer(t, func(ts *httptest.Server) {
		client := ts.Client()
		pngData := minimalPNG(t)

		var (
			galleryURL *url.URL
			password   string
			etag       string
		)

		t.Run("POST", func(t *testing.T) {
			body, ct := imageMultipart(t, "red.png", pngData)

			resp, err := client.Post(ts.URL+"/img", ct, body)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode)

			password = resp.Header.Get("Jot-Password")
			require.NotEmpty(t, password)

			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			u, err := url.Parse(strings.TrimSpace(string(raw)))
			require.NoError(t, err)
			galleryURL = u
		})

		t.Run("GET gallery", func(t *testing.T) {
			resp, err := client.Get(galleryURL.String())
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Contains(t, resp.Header.Get("Content-Type"), "text/html")

			etag = resp.Header.Get("Etag")
			require.NotEmpty(t, etag)
		})

		t.Run("GET gallery with current If-None-Match", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, galleryURL.String(), nil)
			require.NoError(t, err)
			req.Header.Set("if-none-match", etag)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNotModified, resp.StatusCode)
		})

		t.Run("GET gallery with expired If-None-Match", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, galleryURL.String(), nil)
			require.NoError(t, err)
			req.Header.Set("if-none-match", "2001-01-01T00:00:00Z")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
		})

		t.Run("GET image file", func(t *testing.T) {
			fileURL := galleryURL.JoinPath("red.png")

			resp, err := client.Get(fileURL.String())
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, "image/png", resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NotEmpty(t, body)
		})

		t.Run("GET unknown image file returns 404", func(t *testing.T) {
			fileURL := galleryURL.JoinPath("nope.png")

			resp, err := client.Get(fileURL.String())
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("DELETE with wrong password", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodDelete, galleryURL.String(), nil)
			require.NoError(t, err)
			req.SetBasicAuth("", "wrongpassword")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})

		t.Run("DELETE", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodDelete, galleryURL.String(), nil)
			require.NoError(t, err)
			req.SetBasicAuth("", password)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("GET after DELETE", func(t *testing.T) {
			resp, err := client.Get(galleryURL.String())
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("GET nonexistent gallery", func(t *testing.T) {
			resp, err := client.Get(ts.URL + "/img/doesnotexist")
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})
}
