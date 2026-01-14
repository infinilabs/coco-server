/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package incoming

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	url2 "net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	core2 "infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/service"
	"infini.sh/coco/plugins/webhooks/gitlab/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

func init() {
	//fetch related files and save to db?
	//prepare context for LLM process
	//send event to Gitlab?

}

type Config struct {
	Token                string   `json:"token" config:"token"`
	Endpoint             string   `json:"endpoint" config:"endpoint"`
	FinalReviewAssistant string   `json:"report_assistant" config:"report_assistant"`
	SummaryAssistant     string   `json:"summary_assistant" config:"summary_assistant"`
	PageSize             int      `json:"page_size" config:"page_size"`
	MaxInputLength       int      `json:"max_input_length" config:"max_input_length"`
	MaxBatchSize         int      `json:"max_batch_size" config:"max_batch_size"`
	IncludeOldFile       bool     `json:"include_old_file" config:"include_old_file"`
	ViaCurl              bool     `json:"via_curl" config:"via_curl"`
	OnEvents             []string `json:"on_events" config:"on_events"`
	HttpClient           string   `json:"http_client" config:"http_client"`
}

type Processor struct {
	api.Handler
	Queue *queue.QueueConfig `config:"queue"`

	config     *Config
	version    *util.Version
	httpClient *http.Client
}

const processorName = "gitlab_incoming_message"

func init() {
	pipeline.RegisterProcessorPlugin(processorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{HttpClient: "default", PageSize: 10, IncludeOldFile: true, ViaCurl: true, MaxBatchSize: 10, MaxInputLength: 100 * 1024, OnEvents: []string{"open", "update", "reopen"}}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}

	runner := Processor{config: &cfg}
	log.Debug("load config:", util.MustToJSON(cfg))
	log.Debug("http config:", util.MustToJSON(global.Env().SystemConfig.HTTPClientConfig))

	runner.httpClient = api.GetHttpClient(cfg.HttpClient)

	ver, err := runner.getVersion()
	if err != nil {
		panic(err)
	}
	runner.version = ver

	return &runner, nil
}

func (processor *Processor) Name() string {
	return processorName
}

func (processor *Processor) Process(ctx *pipeline.Context) error {

	bodyBytes, ok := ctx.GetBytes("body_bytes")
	if ok {
		docV := ctx.Get("document")
		doc, ok := docV.(core2.Document)
		if ok {
			event := core.MergeRequestEvent{}
			util.MustFromJSONBytes(bodyBytes, &event)
			log.Debug(util.MustToJSON(event))

			switch event.ObjectAttributes.Action {
			case "open":
				doc.Title = fmt.Sprintf("[%v] Created MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			case "update":
				doc.Title = fmt.Sprintf("[%v] Updated MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			case "close":
				doc.Title = fmt.Sprintf("[%v] Closed MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			case "reopen":
				doc.Title = fmt.Sprintf("[%v] Reopened MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			default:
				//save to store
				doc.Title = fmt.Sprintf("[%v] Perform [%v] on MR: %v", event.User.Name, event.ObjectAttributes.Action, event.ObjectAttributes.Title)
			}

			if util.ContainsAnyInArray(event.ObjectAttributes.Action, processor.config.OnEvents) {
				if processor.version.Major() < 16 {
					err := processor.onOpenMRV12(&doc, &event)
					if err != nil {
						panic(err)
					}
				} else {
					err := processor.onOpenMR(&doc, &event)
					if err != nil {
						panic(err)
					}
				}
			}

			doc.ID = util.GetUUID()
			doc.Category = event.Repository.Name
			doc.Metadata["project_id"] = event.Project.ID
			doc.Metadata["action"] = event.ObjectAttributes.Action
			doc.Metadata["mr_id"] = event.ObjectAttributes.IID
			doc.Metadata["target_branch"] = event.ObjectAttributes.TargetBranch
			doc.Payload["raw_event"] = string(bodyBytes)

			ctx.Set("document", doc)
		} else {
			log.Debug("document not found, skip processing")
		}

	}

	return nil
}

func (processor *Processor) onOpenMR(doc *core2.Document, event *core.MergeRequestEvent) error {
	details, _ := processor.getMRDetail(event.Project.ID, event.ObjectAttributes.IID)

	var (
		pageNo       = 1
		allSummaries []string
	)

	log.Debug(util.ToJson(processor.config, true))

	for {
		log.Infof("Processing MR diffs, page: %d", pageNo)
		diffs, err := processor.getMRDiffs(event.Project.ID, event.ObjectAttributes.IID, pageNo)
		if err != nil {
			return err
		}
		if len(diffs) == 0 {
			log.Debug("skip processing for empty diff")
			break
		}
		pageNo++

		total := len(diffs)
		batchSize := processor.config.MaxBatchSize
		totalBatches := (total + batchSize - 1) / batchSize

		for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
			start := batchIndex * batchSize
			end := start + batchSize
			if end > total {
				end = total
			}

			batch := diffs[start:end]
			previousFiles := make(map[string]*core.FileContent, len(batch))
			for _, diff := range batch {
				if v, _ := processor.getPreviousFile(event.Project.ID, diff.OldPath, event.ObjectAttributes.TargetBranch); v != nil {
					previousFiles[diff.OldPath] = v
				}
			}

			localVars := map[string]interface{}{
				"details":            util.SubStringWithSuffix(details, processor.config.MaxInputLength, "..."),
				"diffs":              util.SubStringWithSuffix(util.MustToJSON(batch), processor.config.MaxInputLength, "..."),
				"old_files":          util.SubStringWithSuffix(util.MustToJSON(previousFiles), processor.config.MaxInputLength, "..."),
				"is_batch":           true,
				"review_hits":        batchIndex + 1,
				"batch_total":        totalBatches,
				"batch_size":         len(batch),
				"batch_context_note": "This review focuses only on the current batch. Please do not include any previous batches.",
			}

			// --- Single AI call per batch/page to produce incremental summary ---
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			log.Infof("Calling AI summary assistant for page %d, batch %d/%d...", pageNo-1, batchIndex+1, totalBatches)

			pageSummary, err := service.AskAssistantSync(ctx, doc.GetOwnerID(), processor.config.SummaryAssistant, doc.Title, localVars)
			cancel()

			if err != nil {
				log.Errorf("Summary assistant error (page %d, batch %d): %v", pageNo-1, batchIndex+1, err)
				continue
			}

			if pageSummary != "" {
				log.Infof("Incremental summary collected for page %d, batch %d,\n%v", pageNo-1, batchIndex+1, pageSummary)
				allSummaries = append(allSummaries, pageSummary)
				log.Infof("Incremental summary collected for page %d, batch %d", pageNo-1, batchIndex+1)
			}
		}
	}

	// ---- Final MR Review ----
	if len(allSummaries) > 0 {
		log.Info("Generating final MR review report...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		finalVars := map[string]interface{}{
			"merge_request_details": util.SubStringWithSuffix(details, processor.config.MaxInputLength, "..."),
			"all_page_summaries":    strings.Join(allSummaries, "\n\n\n"),
			"summary_count":         len(allSummaries),
		}

		finalReport, err := service.AskAssistantSync(ctx, doc.GetOwnerID(), processor.config.FinalReviewAssistant, "Final MR Review Report", finalVars)
		if err != nil {
			log.Errorf("Final report generation error: %v", err)
			return err
		}

		if finalReport != "" {
			log.Info("Posting final AI review to GitLab...")
			res, err := processor.postReply(event.Project.ID, event.ObjectAttributes.IID, finalReport)
			if err != nil {
				log.Error(err, ",", util.MustToJSON(res))
				return err
			}
		}
	} else {
		log.Info("no summary for this MR")
	}
	return nil
}

func (processor *Processor) onOpenMRV12(doc *core2.Document, event *core.MergeRequestEvent) error {
	details, _ := processor.getMRDetail(event.Project.ID, event.ObjectAttributes.IID)

	var (
		pageNo       = 1
		allSummaries []string
	)

	log.Debug(util.ToJson(processor.config, true))

	log.Infof("Processing MR diffs, page: %d", pageNo)
	mrDiff, err := processor.getMRDiffsV12(event.Project.ID, event.ObjectAttributes.IID)
	if err != nil {
		return err
	}
	if mrDiff == nil || len(mrDiff.Changes) == 0 {
		return errors.Error("skip processing for empty diff")
	}

	total := len(mrDiff.Changes)
	batchSize := processor.config.MaxBatchSize
	totalBatches := (total + batchSize - 1) / batchSize

	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		start := batchIndex * batchSize
		end := start + batchSize
		if end > total {
			end = total
		}

		batch := mrDiff.Changes[start:end]
		previousFiles := make(map[string]*core.FileContent, len(batch))
		for _, diff := range batch {
			if v, _ := processor.getPreviousFile(event.Project.ID, diff.OldPath, event.ObjectAttributes.TargetBranch); v != nil {
				previousFiles[diff.OldPath] = v
			}
		}

		localVars := map[string]interface{}{
			"details":            util.SubStringWithSuffix(details, processor.config.MaxInputLength, "..."),
			"diffs":              util.SubStringWithSuffix(util.MustToJSON(batch), processor.config.MaxInputLength, "..."),
			"old_files":          util.SubStringWithSuffix(util.MustToJSON(previousFiles), processor.config.MaxInputLength, "..."),
			"is_batch":           true,
			"review_hits":        batchIndex + 1,
			"batch_total":        totalBatches,
			"batch_size":         len(batch),
			"batch_context_note": "This review focuses only on the current batch. Please do not include any previous batches.",
		}

		// --- Single AI call per batch/page to produce incremental summary ---
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		log.Infof("Calling AI summary assistant for page %d, batch %d/%d...", pageNo-1, batchIndex+1, totalBatches)

		pageSummary, err := service.AskAssistantSync(ctx, doc.GetOwnerID(), processor.config.SummaryAssistant, doc.Title, localVars)
		cancel()

		if err != nil {
			log.Errorf("Summary assistant error (page %d, batch %d): %v", pageNo-1, batchIndex+1, err)
			continue
		}

		if pageSummary != "" {
			log.Infof("Incremental summary collected for page %d, batch %d,\n%v", pageNo-1, batchIndex+1, pageSummary)
			allSummaries = append(allSummaries, pageSummary)
			log.Infof("Incremental summary collected for page %d, batch %d", pageNo-1, batchIndex+1)
		}
	}

	// ---- Final MR Review ----
	if len(allSummaries) > 0 {
		log.Info("Generating final MR review report...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		finalVars := map[string]interface{}{
			"merge_request_details": util.SubStringWithSuffix(details, processor.config.MaxInputLength, "..."),
			"all_page_summaries":    strings.Join(allSummaries, "\n\n\n"),
			"summary_count":         len(allSummaries),
		}

		finalReport, err := service.AskAssistantSync(ctx, doc.GetOwnerID(), processor.config.FinalReviewAssistant, "Final MR Review Report", finalVars)
		if err != nil {
			log.Errorf("Final report generation error: %v", err)
			return err
		}

		if finalReport != "" {
			log.Info("Posting final AI review to GitLab...")
			res, err := processor.postReply(event.Project.ID, event.ObjectAttributes.IID, finalReport)
			if err != nil {
				log.Error(err, ",", util.MustToJSON(res))
			}
		}
	} else {
		log.Info("no summary for this MR")
	}
	return nil
}

func JoinAPIURL(base string, apiPath string, query map[string]string) string {
	base = strings.TrimRight(base, "/")
	apiPath = strings.TrimLeft(apiPath, "/")

	fullURL := base + "/" + apiPath

	if len(query) > 0 {
		values := url2.Values{}
		for k, v := range query {
			values.Set(k, v)
		}
		fullURL += "?" + values.Encode()
	}

	log.Debug(fullURL)
	return fullURL
}

func (processor *Processor) getMRDetail(projectID, mrID int64) (string, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v", projectID, mrID),
		nil, // no query params
	)

	req1 := util.NewGetRequest(url, nil)
	res, err := processor.ExecuteRequest(req1)
	if err != nil {
		return "", err
	}
	if res != nil {
		return string(res.Body), nil
	}
	return "", err
}
func (processor *Processor) getMRCommits(projectID, mrID int64) ([]core.MRCommit, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v/commits", projectID, mrID),
		nil, // no query params
	)
	req1 := util.NewGetRequest(url, nil)
	res, err := processor.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		commits := []core.MRCommit{}
		log.Debug(string(res.Body))
		util.MustFromJSONBytes(res.Body, &commits)
		log.Debug(util.MustToJSON(commits))
		return commits, nil
	}

	return nil, err
}

func (processor *Processor) getMRDiffs(projectID, mrID int64, pageNo int) ([]core.MRDiff, error) {
	args := map[string]string{"per_page": util.IntToString(processor.config.PageSize)}
	args["page"] = fmt.Sprintf("%v", pageNo)
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v/diffs", projectID, mrID),
		args,
	)
	req1 := util.NewGetRequest(url, nil)
	res, err := processor.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		log.Debug(string(res.Body))
		commits := []core.MRDiff{}
		util.MustFromJSONBytes(res.Body, &commits)
		log.Debug(util.MustToJSON(commits))
		return commits, nil
	}

	return nil, err
}

func (processor *Processor) getMRDiffsV12(projectID, mrID int64) (*core.MergeRequestV12, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v/changes", projectID, mrID),
		nil,
	)
	req1 := util.NewGetRequest(url, nil)
	res, err := processor.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		log.Debug(string(res.Body))
		commits := core.MergeRequestV12{}
		util.MustFromJSONBytes(res.Body, &commits)
		log.Debug(util.MustToJSON(commits))
		return &commits, nil
	}

	return nil, err
}

func (processor *Processor) getPreviousFile(projectID int64, file string, branch string) (*core.FileContent, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/repository/files/%v", projectID, util.UrlEncode(file)),
		map[string]string{"ref": branch},
	)
	req1 := util.NewGetRequest(url, nil)
	res, err := processor.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		file := core.FileContent{}
		log.Debug(string(res.Body))
		util.MustFromJSONBytes(res.Body, &file)
		log.Debug(util.MustToJSON(file))
		return &file, nil
	}

	return nil, err
}

func (processor *Processor) postReply(projectID, mrID int64, msg string) (*core.MergeRequestNote, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v/notes", projectID, mrID),
		nil,
	)

	bytes := util.MustToJSONBytes(util.MapStr{
		"body": msg,
	})
	req1 := util.NewPostRequest(url, bytes)
	req1.AddHeader("Content-Type", "application/json")

	res, err := processor.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		log.Debug(string(res.Body))
		commits := core.MergeRequestNote{}
		util.MustFromJSONBytes(res.Body, &commits)
		log.Debug(util.MustToJSON(commits))
		return &commits, nil
	}

	return nil, err
}

func (processor *Processor) ExecuteRequest(req *util.Request) (*util.Result, error) {
	req.AddHeader("PRIVATE-TOKEN", processor.config.Token)

	if processor.config.ViaCurl {
		cfg, ok := global.Env().SystemConfig.HTTPClientConfig[processor.config.HttpClient]
		if !ok {
			panic("http client config was missing")
		}
		return ExecuteRequestViaCurl(req, cfg.TLSConfig.TLSCertFile, cfg.TLSConfig.TLSKeyFile, cfg.TLSConfig.TLSCertPassword)
	}
	return util.ExecuteRequestWithCatchFlag(processor.httpClient, req, true)
}

// ExecuteRequestViaCurl executes a request using curl, automatically handling headers and parameters
func ExecuteRequestViaCurl(req *util.Request, cert, key, pass string) (*util.Result, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	// Build curl command arguments
	args := []string{"-k", "-s", "--connect-timeout", "30"}

	// Add method
	method := strings.ToUpper(req.Method)
	if method != "" && method != "GET" {
		args = append(args, "-X", method)
	}

	// Add headers
	headers := req.AllHeaders()
	if headers != nil {
		for key, value := range headers {
			args = append(args, "-H", fmt.Sprintf("%s: %s", key, value))
		}
	}

	// Handle Content-Type if set
	if req.ContentType != "" {
		args = append(args, "-H", fmt.Sprintf("Content-Type: %s", req.ContentType))
	}

	// Add User-Agent if set
	if req.Agent != "" {
		args = append(args, "-H", fmt.Sprintf("User-Agent: %s", req.Agent))
	}

	// Add client certificate
	if cert != "" {
		args = append(args, "--cert", cert)
	}
	if key != "" {
		args = append(args, "--key", key)
	}
	if pass != "" {
		args = append(args, "--pass", pass)
	}

	// Add body data
	if len(req.Body) > 0 {
		// Create temporary file for body content
		tmpFile, err := os.CreateTemp("", "curl_body_*.tmp")
		if err != nil {
			return nil, errors.New("failed to create temp file for curl body")
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		// Write body to temp file
		if _, err := tmpFile.Write(req.Body); err != nil {
			return nil, errors.New("failed to write body to temp file")
		}

		// Use the temp file as data source
		args = append(args, "--data-binary", "@"+tmpFile.Name())
	}

	// Add the URL
	args = append(args, req.Url)

	// Add response headers and status code to output
	args = append(args, "-w", "\nCURL_STATUS_CODE:%{http_code}\nCURL_RESPONSE_TIME:%{time_total}")

	// Execute curl command
	cmd := exec.Command("curl", args...)

	log.Debug(args)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		log.Errorf("curl command failed: %v, stderr: %s", err, stderr.String())
		return nil, errors.Errorf("curl request failed: %v, stderr: %s", err, stderr.String())
	}

	// Parse curl output to separate body, headers, and status code
	output := stdout.String()

	// Find and extract status code
	statusCode := 200 // Default status
	if strings.Contains(output, "CURL_STATUS_CODE:") {
		parts := strings.Split(output, "CURL_STATUS_CODE:")
		if len(parts) > 1 {
			statusParts := strings.Split(strings.TrimSpace(parts[1]), "\n")
			if len(statusParts) > 0 {
				if code, err := strconv.Atoi(strings.TrimSpace(statusParts[0])); err == nil {
					statusCode = code
				}
			}
		}
	}

	// Extract actual body (everything before the CURL_ markers)
	body := output
	if curlIndex := strings.LastIndex(output, "\nCURL_STATUS_CODE:"); curlIndex != -1 {
		body = output[:curlIndex]
	}

	result := &util.Result{
		Body:       []byte(body),
		Headers:    map[string][]string{}, // curl output includes headers mixed with body
		StatusCode: statusCode,
		// ResponseTime is not part of Result struct, could log if needed
	}

	return result, nil
}

func (processor *Processor) getVersion() (*util.Version, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		"/api/v4/version",
		nil,
	)
	req1 := util.NewGetRequest(url, nil)

	res, err := processor.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		log.Debug("gitlab version: ", string(res.Body))
		ver := core.GitlabVersion{}
		util.MustFromJSONBytes(res.Body, &ver)
		log.Debug(util.MustToJSON(ver))
		return util.ParseSemantic(ver.Version)
	}

	return nil, err
}
