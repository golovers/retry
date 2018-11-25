package retry_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golovers/retry"
)

func TestRetryFailed(t *testing.T) {
	t.Parallel()
	cnt := uint64(0)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			cnt++
		}()
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer ts.Close()
	c := retry.New()
	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Errorf("failed to create request, err: %v", err)
	}
	if _, err := c.Do(req); err == nil {
		t.Errorf("Do(req) got error=nil, want error != nil")
	}
	// first time + DefaultMaxRetry :)
	if cnt != retry.DefaultMaxRetry+1 {
		t.Errorf("Do(req) executed %d times, want %d times", cnt, retry.DefaultMaxRetry)
	}
}

func TestRetrySuccessAtSecondTime(t *testing.T) {
	t.Parallel()
	cnt := uint64(0)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			cnt++
		}()
		if cnt == 0 {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()
	c := retry.New()
	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Errorf("failed to create request, err: %v", err)
	}
	rs, err := c.Do(req)
	if err != nil {
		t.Errorf("Do(req) got error=%v, want error=nil", err)
	}
	b, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		t.Errorf("failed to read response body, err: %v", err)
	}
	if "ok" != string(b) {
		t.Errorf("got response body: '%v', want '%v'", string(b), "ok")
	}
	if cnt != 2 {
		t.Errorf("Do(req) executed %d times, want %d times", cnt, 2)
	}
}
