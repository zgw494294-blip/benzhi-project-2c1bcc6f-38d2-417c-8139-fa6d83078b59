package main

import (
	"archive-release/internal/application"
	"archive-release/internal/persistence"
	"archive-release/internal/transport"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19081", "监听地址")
	dataDir := flag.String("data-dir", "./data", "数据目录")
	self := flag.Bool("selfcheck", false, "执行自检后退出")
	flag.Parse()
	if p := os.Getenv("PORT"); p != "" && !flag.CommandLine.Parsed() {
		*addr = "127.0.0.1:" + p
	}
	if p := os.Getenv("PORT"); p != "" && *addr == "127.0.0.1:19081" {
		*addr = "127.0.0.1:" + p
	}
	key := os.Getenv("RELEASE_HMAC_KEY")
	if key == "" {
		key = "development-key"
	}
	var repo persistence.Repository
	if *self {
		repo = persistence.NewMemory()
	} else {
		fileRepo, err := persistence.NewFile(*dataDir)
		if err != nil {
			panic(err)
		}
		repo = fileRepo
	}
	app := application.New(repo, key)
	srv := transport.New(app)
	if *self {
		if err := runSelfCheck(*addr, srv.Handler()); err != nil {
			panic(err)
		}
		fmt.Println("自检通过")
		return
	}
	server := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second}
	fmt.Printf("典藏扫描品发布核准台监听 %s\n", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
func runSelfCheck(addr string, h http.Handler) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ts := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 2 * time.Second}
	ln, err := net.Listen("tcp", ts.Addr)
	if err != nil {
		return err
	}
	go ts.Serve(ln)
	defer ts.Shutdown(context.Background())
	base := "http://" + ln.Addr().String()
	client := &http.Client{Timeout: 3 * time.Second}
	do := func(method, path string, body []byte) (*http.Response, error) {
		req, requestErr := http.NewRequestWithContext(ctx, method, base+path, bytes.NewReader(body))
		if requestErr != nil {
			return nil, requestErr
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return client.Do(req)
	}
	res, err := do(http.MethodGet, "/", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("首页状态 %d", res.StatusCode)
	}
	for _, asset := range []string{"/static/style.css", "/static/app.js"} {
		assetResponse, assetErr := do(http.MethodGet, asset, nil)
		if assetErr != nil {
			return assetErr
		}
		assetResponse.Body.Close()
		if assetResponse.StatusCode != http.StatusOK {
			return fmt.Errorf("前端资源 %s 状态 %d", asset, assetResponse.StatusCode)
		}
	}
	payload := map[string]string{"title": "自检案卷", "archiveUnit": "SELF", "creator": "selfcheck", "actor": "selfcheck", "role": "archivist", "idempotencyKey": "selfcheck-case"}
	body, _ := json.Marshal(payload)
	post := func() (*http.Response, error) {
		return do(http.MethodPost, "/api/cases", body)
	}
	r1, err := post()
	if err != nil {
		return err
	}
	defer r1.Body.Close()
	if r1.StatusCode != 200 {
		return fmt.Errorf("创建状态 %d", r1.StatusCode)
	}
	var created struct {
		CaseID  string `json:"caseID"`
		Version int    `json:"version"`
	}
	if err = json.NewDecoder(r1.Body).Decode(&created); err != nil {
		return fmt.Errorf("解析创建响应: %w", err)
	}
	r2, err := post()
	if err != nil {
		return err
	}
	defer r2.Body.Close()
	if r2.StatusCode != 200 {
		return fmt.Errorf("幂等状态 %d", r2.StatusCode)
	}
	var repeated struct {
		CaseID string `json:"caseID"`
	}
	if err = json.NewDecoder(r2.Body).Decode(&repeated); err != nil || repeated.CaseID != created.CaseID {
		return errors.New("幂等请求未重放原案卷")
	}
	conflict := map[string]any{
		"actor": "selfcheck-conservator", "role": "conservator", "expectedVersion": created.Version + 1,
		"idempotencyKey": "selfcheck-conflict", "metadata": map[string]string{"period": "自检"},
		"pages": []map[string]any{{"pageNumber": 1, "clarity": 95, "skew": 0.5, "cropRatio": 0.99, "colorTarget": true}},
	}
	cb, _ := json.Marshal(conflict)
	cr, err := do(http.MethodPost, "/api/cases/"+created.CaseID+"/revisions", cb)
	if err != nil {
		return err
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusConflict {
		return fmt.Errorf("版本冲突状态 %d", cr.StatusCode)
	}
	if contentType := cr.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		return fmt.Errorf("冲突响应类型 %s", contentType)
	}
	return nil
}
