/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package incoming

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	core2 "infini.sh/coco/core"
	"infini.sh/coco/modules/assistant"
	"infini.sh/coco/plugins/webhooks/gitlab/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"net/http"
	url2 "net/url"
	"strings"
	"time"
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
	OnEvents             []string `json:"on_events" config:"on_events"`
	HttpClient           string   `json:"http_client" config:"http_client"`
}

type Processor struct {
	api.Handler
	Queue *queue.QueueConfig `config:"queue"`

	config     *Config
	httpClient *http.Client
}

const processorName = "gitlab_incoming_message"

func init() {
	pipeline.RegisterProcessorPlugin(processorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{HttpClient: "default", PageSize: 10, MaxBatchSize: 10, MaxInputLength: 10 * 1024, OnEvents: []string{"open", "update", "reopen"}}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}

	runner := Processor{config: &cfg}
	log.Info("load config:", util.MustToJSON(cfg))
	log.Info("http config:", util.MustToJSON(global.Env().SystemConfig.HTTPClientConfig))
	runner.httpClient = api.GetHttpClient(cfg.HttpClient)
	return &runner, nil
}

func (processor *Processor) Name() string {
	return processorName
}

func (processor *Processor) Process(ctx *pipeline.Context) error {

	bodyBytes, ok := ctx.GetBytes("body_bytes")
	if ok {
		event := core.MergeRequestEvent{}
		util.MustFromJSONBytes(bodyBytes, &event)
		log.Debug(util.MustToJSON(event))

		docV := ctx.Get("document")
		doc, ok := docV.(core2.Document)
		if ok {
			switch event.ObjectAttributes.Action {
			case "open":
				doc.Title = fmt.Sprintf("[%v] 创建了 MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			case "update":
				doc.Title = fmt.Sprintf("[%v] 更新了 MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			case "close":
				doc.Title = fmt.Sprintf("[%v] 关闭了 MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			case "reopen":
				doc.Title = fmt.Sprintf("[%v] 重新打开了 MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			default:
				//save to store
				doc.Title = fmt.Sprintf("[%v] [%v] on MR: %v", event.User.Name, event.ObjectAttributes.Action, event.ObjectAttributes.Title)
			}

			if util.ContainsAnyInArray(event.ObjectAttributes.Action, processor.config.OnEvents) {
				processor.onOpenMR(&doc, &event)
			}

			doc.ID = util.GetUUID()
			doc.Category = event.Repository.Name
			doc.Metadata["project_id"] = event.Project.ID
			doc.Metadata["action"] = event.ObjectAttributes.Action
			doc.Metadata["mr_id"] = event.ObjectAttributes.IID
			doc.Metadata["target_branch"] = event.ObjectAttributes.TargetBranch
			doc.Payload["raw_event"] = event

			ctx.Set("document", doc)
		}

	}

	return nil
}

func (processor *Processor) onOpenMR(doc *core2.Document, event *core.MergeRequestEvent) {
	details, _ := processor.getMRDetail(event.Project.ID, event.ObjectAttributes.IID)

	var (
		pageNo       = 1
		allSummaries []string
	)

	log.Debug(util.ToJson(processor.config, true))

	for {
		log.Infof("Processing MR diffs, page: %d", pageNo)
		diffs, _ := processor.getMRDiffs(event.Project.ID, event.ObjectAttributes.IID, pageNo)
		if len(diffs) == 0 {
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
				"details":            util.SubStringWithSuffix(util.MustToJSON(details), processor.config.MaxInputLength, "..."),
				"diffs":              util.SubStringWithSuffix(util.MustToJSON(batch), processor.config.MaxInputLength, "..."),
				"old_files":          util.SubStringWithSuffix(util.MustToJSON(previousFiles), processor.config.MaxInputLength, "..."),
				"is_batch":           true,
				"review_hits":        batchIndex + 1,
				"batch_total":        totalBatches,
				"batch_size":         len(batch),
				"batch_context_note": "本次审查只关注当前批次文件，请不要包含前批次内容。",
			}

			// --- Single AI call per batch/page to produce incremental summary ---
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			log.Infof("Calling AI summary assistant for page %d, batch %d/%d...", pageNo-1, batchIndex+1, totalBatches)

			pageSummary, err := assistant.AskAssistantSync(ctx, doc.GetOwnerID(), processor.config.SummaryAssistant, doc.Title, localVars)
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
			"merge_request_details": util.SubStringWithSuffix(util.MustToJSON(details), processor.config.MaxInputLength, "..."),
			"all_page_summaries":    strings.Join(allSummaries, "\n\n"),
			"summary_count":         len(allSummaries),
		}

		finalReport, err := assistant.AskAssistantSync(ctx, doc.GetOwnerID(), processor.config.FinalReviewAssistant, "Final MR Review Report", finalVars)
		if err != nil {
			log.Errorf("Final report generation error: %v", err)
			return
		}

		if finalReport != "" {
			log.Info("Posting final AI review to GitLab...")
			res, err := processor.postReply(event.Project.ID, event.ObjectAttributes.IID, finalReport)
			if err != nil {
				log.Error(err, ",", util.MustToJSON(res))
			}
		}
	}
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

func (processor *Processor) getMRDetail(projectID, mrID int64) (*core.MergeRequestDetail, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v", projectID, mrID),
		nil, // no query params
	)

	req1 := util.NewGetRequest(url, nil)
	req1.AddHeader("PRIVATE-TOKEN", processor.config.Token)
	res, err := util.ExecuteRequestWithCatchFlag(processor.httpClient, req1, true)
	if err != nil {
		return nil, err
	}

	if res != nil {
		detail := core.MergeRequestDetail{}
		log.Debug(string(res.Body))
		util.MustFromJSONBytes(res.Body, &detail)
		return &detail, nil
	}

	return nil, err
}
func (processor *Processor) getMRCommits(projectID, mrID int64) ([]core.MRCommit, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v/commits", projectID, mrID),
		nil, // no query params
	)
	req1 := util.NewGetRequest(url, nil)
	req1.AddHeader("PRIVATE-TOKEN", processor.config.Token)
	res, err := util.ExecuteRequestWithCatchFlag(processor.httpClient, req1, true)
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
	req1.AddHeader("PRIVATE-TOKEN", processor.config.Token)
	res, err := util.ExecuteRequestWithCatchFlag(processor.httpClient, req1, true)
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

func (processor *Processor) getPreviousFile(projectID int64, file string, branch string) (*core.FileContent, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/repository/files/%v", projectID, util.UrlEncode(file)),
		map[string]string{"ref": branch},
	)
	req1 := util.NewGetRequest(url, nil)
	req1.AddHeader("PRIVATE-TOKEN", processor.config.Token)
	res, err := util.ExecuteRequestWithCatchFlag(processor.httpClient, req1, true)
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
	req1.AddHeader("PRIVATE-TOKEN", processor.config.Token)
	req1.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequestWithCatchFlag(processor.httpClient, req1, true)
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
