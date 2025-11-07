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
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
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
	Token          string `json:"token"`
	Endpoint       string `json:"endpoint"`
	Prompt         string `json:"prompt"`
	Assistant      string `json:"assistant"`
	PageSize       int    `json:"page_size"`
	IncludeOldFile bool   `json:"include_old_file"`
}

type Processor struct {
	api.Handler
	Queue *queue.QueueConfig `config:"queue"`

	config *Config
}

const processorName = "gitlab_incoming_message"

func init() {
	pipeline.RegisterProcessorPlugin(processorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{PageSize: 10}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}
	runner := Processor{config: &cfg}
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
				processor.onOpenMR(&doc, &event)
				break
			case "update":
				doc.Title = fmt.Sprintf("[%v] 更新了 MR: %v", event.User.Name, event.ObjectAttributes.Title)
				break
			case "close":
				doc.Title = fmt.Sprintf("[%v] 关闭了 MR: %v", event.User.Name, event.ObjectAttributes.Title)
				processor.onOpenMR(&doc, &event)
				break
			case "reopen":
				doc.Title = fmt.Sprintf("[%v] 重新打开了 MR: %v", event.User.Name, event.ObjectAttributes.Title)
				processor.onOpenMR(&doc, &event)
				break
			default:
				//save to store
				doc.Title = fmt.Sprintf("[%v] [%v] on MR: %v", event.User.Name, event.ObjectAttributes.Action, event.ObjectAttributes.Title)
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
	vars := map[string]any{}

	details, _ := processor.getMRDetail(event.Project.ID, event.ObjectAttributes.IID)
	diffs, _ := processor.getMRDiffs(event.Project.ID, event.ObjectAttributes.IID)

	if len(diffs) > 0 {
		previousFile := map[string]*core.FileContent{}
		for _, diff := range diffs {
			v, _ := processor.getPreviousFile(event.Project.ID, diff.OldPath, event.ObjectAttributes.TargetBranch)
			if v != nil {
				previousFile[diff.OldPath] = v
			}
		}
		vars["details"] = util.MustToJSON(details)
		vars["diffs"] = util.MustToJSON(diffs)
		vars["old_files"] = util.MustToJSON(previousFile)
	}

	//TODO, send to task framework
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(5)*time.Minute)

	finalText, err := assistant.AskAssistantSync(ctx, processor.config.Assistant, doc.Title, vars)
	if err != nil {
		panic(err)
	}

	log.Debug("get final text: ", finalText)
	if finalText != "" {
		processor.postReply(event.Project.ID, event.ObjectAttributes.IID, finalText)
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
	res, err := util.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		//log.Error(string(res.Body))
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
	res, err := util.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		//log.Error(string(res.Body))
		commits := []core.MRCommit{}
		log.Debug(string(res.Body))
		util.MustFromJSONBytes(res.Body, &commits)
		log.Debug(util.MustToJSON(commits))
		return commits, nil
	}

	return nil, err
}
func (processor *Processor) getMRDiffs(projectID, mrID int64) ([]core.MRDiff, error) {
	url := JoinAPIURL(
		processor.config.Endpoint,
		fmt.Sprintf("/api/v4/projects/%v/merge_requests/%v/diffs", projectID, mrID),
		map[string]string{"per_page": util.IntToString(processor.config.PageSize)},
	)
	req1 := util.NewGetRequest(url, nil)
	req1.AddHeader("PRIVATE-TOKEN", processor.config.Token)
	res, err := util.ExecuteRequest(req1)
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
	res, err := util.ExecuteRequest(req1)
	if err != nil {
		return nil, err
	}

	if res != nil {
		//log.Error(string(res.Body))
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
		//map[string]string{"body": util.IntToString(processor.config.PageSize)},
	)

	bytes := util.MustToJSONBytes(util.MapStr{
		"body": msg,
	})
	req1 := util.NewPostRequest(url, bytes)
	req1.AddHeader("PRIVATE-TOKEN", processor.config.Token)
	req1.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req1)
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
