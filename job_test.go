package ggsel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBatchOffersLimit(t *testing.T) {
	t.Parallel()
	var path, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		body = string(buf)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"job_id":"b"}`)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL, Options{APIKey: "k"})
	ids := make([]int64, maxOfferBatch)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	job, err := c.PauseOffers(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api_sellers/v2/offers/batch_pause" || !strings.Contains(body, `"offer_ids":[1,2`) || job.ID != "b" {
		t.Fatalf("path=%s body=%s job=%+v", path, body[:40], job)
	}
	for _, call := range []struct {
		path string
		fn   func([]int64) (Job, error)
	}{
		{"/api_sellers/v2/offers/batch_activate", func(ids []int64) (Job, error) {
			return c.ActivateOffers(context.Background(), ids)
		}},
		{"/api_sellers/v2/offers/batch_delete", func(ids []int64) (Job, error) {
			return c.DeleteOffers(context.Background(), ids)
		}},
	} {
		if _, err := call.fn([]int64{1}); err != nil {
			t.Fatal(err)
		}
		if path != call.path {
			t.Fatalf("path = %s, want %s", path, call.path)
		}
	}
	ids = append(ids, 101)
	if _, err := c.ActivateOffers(context.Background(), ids); err == nil {
		t.Fatal("101 ids passed")
	}
	if _, err := c.DeleteOffers(context.Background(), nil); err == nil {
		t.Fatal("empty ids passed")
	}
}

func TestJobStatusShapes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api_sellers/api/apilogin":
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
		case strings.Contains(r.RequestURI, "/api_sellers/v2/async_job_results/a%2Fb"):
			if r.Header.Get("Authorization") == "" {
				t.Error("missing v2 key")
			}
			_, _ = io.WriteString(w, `{"data":{"job_id":"a/b","status":"completed","results":{"price":19.99,"ok":true}}}`)
		case r.URL.Path == "/api_sellers/api/product/edit/UpdateProductsTaskStatus":
			if r.Header.Get("Authorization") != "" || r.URL.Query().Get("taskId") != "task-9" {
				t.Errorf("auth=%q query=%s", r.Header.Get("Authorization"), r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"TaskId":"task-9","Status":2,"SuccessCount":1,"ErrorCount":0,"TotalCount":1,"ErrorsDescriptions":[{"Key":"1","Value":"ok"}]}`)
		default:
			t.Errorf("path = %s", r.RequestURI)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Clock: func() time.Time { return now }})
	job, err := c.GetAsyncJob(context.Background(), "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != AsyncJobCompleted || job.ID != "a/b" || !strings.Contains(string(job.Results), "19.99") {
		t.Fatalf("job = %+v results=%s", job, job.Results)
	}
	task, err := c.GetProductTask(context.Background(), "task-9")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != ProductTaskStatus2 || task.ID != "task-9" || len(task.Errors) != 1 {
		t.Fatalf("task = %+v", task)
	}
	if _, err := c.GetAsyncJob(context.Background(), " .. "); err == nil {
		t.Fatal("blank job id passed")
	}
}

func TestEmptyJobIDIsAnError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api_sellers/api/apilogin":
			_, _ = io.WriteString(w, `{"retval":0,"token":"tok","valid_thru":"2026-01-02T05:04:05Z"}`)
		case "/api_sellers/v2/async_job_results/job":
			_, _ = io.WriteString(w, `{"data":{"job_id":" ","status":"pending","results":{}}}`)
		case "/api_sellers/api/product/edit/UpdateProductsTaskStatus":
			_, _ = io.WriteString(w, `{"TaskId":"","Status":1,"SuccessCount":0,"ErrorCount":0,"TotalCount":0,"ErrorsDescriptions":[]}`)
		default:
			t.Errorf("path = %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := newTestClient(t, srv.URL, Options{APIKey: "secret", SellerID: 4, Clock: func() time.Time { return now }})
	if _, err := c.GetAsyncJob(context.Background(), "job"); err == nil {
		t.Fatal("empty async job id")
	}
	if _, err := c.GetProductTask(context.Background(), "task"); err == nil {
		t.Fatal("empty product task id")
	}
}
