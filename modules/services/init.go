// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package services

import (
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
	"net/http"
)

type APIHandler struct {
	api.Handler
}

type TranscriptionRequest struct {
	AudioType    string `json:"type"`    //wav,mp3, etc
	AudioContent string `json:"content"` //in base64
}

type TranscriptionResponse struct {
}

func (h APIHandler) transcription(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//url := "https://dashscope.aliyuncs.com/api/v1/services/audio/asr/transcription"
	//
	//request := &util.Request{
	//	Method:  http.MethodPost,
	//	Url:     url,
	//	Context: context.Background(),
	//}
	//
	//apiToken := ""
	//request.AddHeader("Authorization", fmt.Sprintf("Bearer %v", apiToken))
	//request.AddHeader("Content-Type", "application/json")
	//request.AddHeader("X-DashScope-Async", "enable")
	//
	//data := util.MapStr{}
	//data["model"] = "sensevoice-v1"
	//data["input"] = util.MapStr{
	//	"file_urls": []string{
	//		"https://dashscope.oss-cn-beijing.aliyuncs.com/samples/audio/sensevoice/rich_text_example_1.wav",
	//	},
	//}
	//data["parameters"] = util.MapStr{
	//	"channel_id": []int{
	//		0,
	//	},
	//	"disfluency_removal_enabled": false,
	//	"language_hints":             []string{"auto"},
	//}
	//request.Body = util.MustToJSONBytes(data)
	//
	//res, err := util.ExecuteRequest(request)
	//if err != nil {
	//	panic(err)
	//}
	//
	result := util.MapStr{}
	//result["task_id"] = "my_task_id"
	//result["task_status"] = "PENDING"

	result["text"] = "COMING SOON"

	h.WriteAckJSON(w, true, 200, result)
}

func (h APIHandler) getTasks(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

}

func (h APIHandler) getTask(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//taskID := ps.MustGetParameter("task_id")
	//url := fmt.Sprintf("https://dashscope.aliyuncs.com/api/v1/tasks/%v", taskID)
	//
	//request := &util.Request{
	//	Method:  http.MethodPost,
	//	Url:     url,
	//	Context: context.Background(),
	//}
	//
	//apiToken := ""
	//request.AddHeader("Authorization", fmt.Sprintf("Bearer %v", apiToken))

	//res, err := util.ExecuteRequest(request)
	//if err != nil {
	//	panic(err)
	//}
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/services/audio/transcription", handler.transcription, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/services/tasks/", handler.getTasks, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/services/task/:task_id", handler.getTask, api.RequireLogin())

}
