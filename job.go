package ggsel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const maxOfferBatch = 100

// Job — принятая асинхронная операция v2. Клиент её не опрашивает.
type Job struct {
	Success bool   `json:"success"`
	ID      string `json:"job_id"`
}

// AsyncJobStatus — состояние результата v2.
type AsyncJobStatus string

const (
	AsyncJobPending   AsyncJobStatus = "pending"
	AsyncJobCompleted AsyncJobStatus = "completed"
	AsyncJobFailed    AsyncJobStatus = "failed"
)

// AsyncJob — результат GET /api_sellers/v2/async_job_results/{job_id}.
// Results остаётся сырым JSON: у разных операций разная форма, и числа не становятся float64.
type AsyncJob struct {
	ID      string          `json:"job_id"`
	Status  AsyncJobStatus  `json:"status"`
	Results json.RawMessage `json:"results"`
}

// ProductTaskStatus — статус задачи правки цен v1. В снимке это 1, 2 и 3.
type ProductTaskStatus int

const (
	ProductTaskStatus1 ProductTaskStatus = 1
	ProductTaskStatus2 ProductTaskStatus = 2
	ProductTaskStatus3 ProductTaskStatus = 3
)

// ProductTaskError — одна ошибка массовой правки цен.
type ProductTaskError struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ProductTask — статус задачи v1. Это не AsyncJob: другой путь и другие поля.
type ProductTask struct {
	ID           string             `json:"TaskId"`
	Status       ProductTaskStatus  `json:"Status"`
	SuccessCount int64              `json:"SuccessCount"`
	ErrorCount   int64              `json:"ErrorCount"`
	TotalCount   int64              `json:"TotalCount"`
	Errors       []ProductTaskError `json:"ErrorsDescriptions"`
}

// ActivateOffers ставит офферы в продажу. В пачке не больше 100 id.
func (c *Client) ActivateOffers(ctx context.Context, offerIDs []int64) (Job, error) {
	return c.batchOffers(ctx, "activate offers", "/api_sellers/v2/offers/batch_activate", offerIDs)
}

// PauseOffers снимает офферы с продажи. В пачке не больше 100 id.
func (c *Client) PauseOffers(ctx context.Context, offerIDs []int64) (Job, error) {
	return c.batchOffers(ctx, "pause offers", "/api_sellers/v2/offers/batch_pause", offerIDs)
}

// DeleteOffers архивирует офферы. В пачке не больше 100 id.
func (c *Client) DeleteOffers(ctx context.Context, offerIDs []int64) (Job, error) {
	return c.batchOffers(ctx, "delete offers", "/api_sellers/v2/offers/batch_delete", offerIDs)
}

// GetAsyncJob возвращает состояние задачи v2. Повтор безопасен. Цикл опроса остаётся у вызывающего кода.
func (c *Client) GetAsyncJob(ctx context.Context, jobID string) (AsyncJob, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" || jobID == "." || jobID == ".." {
		return AsyncJob{}, fmt.Errorf("ggsel: get async job: id is required")
	}
	var env objectEnvelope
	if err := c.do(ctx, call{
		method:    http.MethodGet,
		path:      "/api_sellers/v2/async_job_results/" + url.PathEscape(jobID),
		errorPath: "/api_sellers/v2/async_job_results/{job_id}",
		header:    c.localeHeader(),
		dest:      &env,
		auth:      authV2,
		retry:     true,
	}); err != nil {
		return AsyncJob{}, err
	}
	var job AsyncJob
	if err := decodeData(env.Data, &job); err != nil {
		return AsyncJob{}, fmt.Errorf("ggsel: get async job: %w", err)
	}
	if strings.TrimSpace(job.ID) == "" {
		return AsyncJob{}, fmt.Errorf("ggsel: get async job: empty job id")
	}
	return job, nil
}

// GetProductTask возвращает статус массовой правки цен v1.
func (c *Client) GetProductTask(ctx context.Context, taskID string) (ProductTask, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ProductTask{}, fmt.Errorf("ggsel: get product task: id is required")
	}
	q := url.Values{}
	q.Set("taskId", taskID)
	var task ProductTask
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/product/edit/UpdateProductsTaskStatus",
		query:  q,
		dest:   &task,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return ProductTask{}, err
	}
	if strings.TrimSpace(task.ID) == "" {
		return ProductTask{}, fmt.Errorf("ggsel: get product task: empty task id")
	}
	if task.Errors == nil {
		task.Errors = []ProductTaskError{}
	}
	return task, nil
}

func (c *Client) batchOffers(ctx context.Context, op, path string, offerIDs []int64) (Job, error) {
	if err := requireIDs(op, offerIDs, maxOfferBatch); err != nil {
		return Job{}, err
	}
	var job Job
	if err := c.do(ctx, call{
		method: http.MethodPost,
		path:   path,
		header: c.localeHeader(),
		body: struct {
			OfferIDs []int64 `json:"offer_ids"`
		}{OfferIDs: offerIDs},
		dest: &job,
		auth: authV2,
	}); err != nil {
		return Job{}, err
	}
	return job, nil
}
