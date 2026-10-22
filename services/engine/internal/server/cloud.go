package server

import (
	"bytes"
	"cloud.google.com/go/compute/metadata"
	"context"
	"fmt"
	"github.com/optitrace/optitrace/services/engine/internal/evaluation"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type identityTokenSource struct{ audience string }

func (t identityTokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, e := metadata.GetWithContext(ctx, "instance/service-accounts/default/identity?audience="+url.QueryEscape(t.audience)+"&format=full")
	if e != nil {
		return nil, e
	}
	return &oauth2.Token{AccessToken: raw, TokenType: "Bearer", Expiry: time.Now().Add(50 * time.Minute)}, nil
}
func NewEngine(ctx context.Context) (*evaluation.Engine, error) {
	audience := os.Getenv("SIMULATOR_AUDIENCE")
	if audience == "" {
		if os.Getenv("APP_ENV") == "production" {
			return nil, fmt.Errorf("SIMULATOR_AUDIENCE required in production")
		}
		return evaluation.New(os.Getenv("SIMULATOR_URL"))
	}
	client := oauth2.NewClient(ctx, oauth2.ReuseTokenSource(nil, identityTokenSource{audience}))
	return evaluation.NewWithHTTPClient(os.Getenv("SIMULATOR_URL"), client)
}
func launchBenchmark() {
	name := os.Getenv("BENCHMARK_JOB_NAME")
	if name == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, e := google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if e != nil {
		slog.Error("benchmark launch credentials", "error", e)
		return
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://run.googleapis.com/v2/"+name+":run", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	res, e := client.Do(req)
	if e != nil {
		slog.Error("benchmark launch deferred to scheduler", "error", e)
		return
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		slog.Error("benchmark launch deferred to scheduler", "status", res.StatusCode)
	}
}
func uploadReport(ctx context.Context, object string, raw []byte) error {
	bucket := os.Getenv("REPORT_BUCKET")
	if bucket == "" {
		return nil
	}
	client, e := google.DefaultClient(ctx, "https://www.googleapis.com/auth/devstorage.read_write")
	if e != nil {
		return e
	}
	endpoint := "https://storage.googleapis.com/upload/storage/v1/b/" + url.PathEscape(bucket) + "/o?uploadType=media&ifGenerationMatch=0&name=" + url.QueryEscape(object)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	res, e := client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return fmt.Errorf("report upload HTTP %d", res.StatusCode)
	}
	return nil
}
