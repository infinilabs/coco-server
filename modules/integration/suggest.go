/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/util"
	"math/rand"
	"net/http"
	"time"
)

const SuggestTopicPerIntegrationKy = "suggest_topic_integration"

func (h APIHandler) updateSuggestTopic(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	appIntegrationID := ps.MustGetParameter("id")

	topics := []string{}
	body, err := h.GetRawBody(req)
	if err != nil {
		panic(err)
	}
	err = util.FromJSONBytes(body, &topics)
	if err != nil {
		panic(err)
	}

	err = kv.AddValue(SuggestTopicPerIntegrationKy, []byte(appIntegrationID), util.MustToJSONBytes(topics))
	if err != nil {
		panic(err)
	}

	h.WriteAckOKJSON(w)
}

func (h APIHandler) viewSuggestTopic(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	appIntegrationID := ps.MustGetParameter("id")

	topics := []string{}
	v, err := kv.GetValue(SuggestTopicPerIntegrationKy, []byte(appIntegrationID))
	if err != nil {
		panic(err)
	}

	err = util.FromJSONBytes(v, &topics)
	if err != nil {
		h.Error(w, err)
	}

	if len(v) > 0 {
		if len(topics) > 3 {
			// Shuffle topics randomly and pick 3
			rand.Seed(time.Now().UnixNano()) // Ensure randomness
			rand.Shuffle(len(topics), func(i, j int) { topics[i], topics[j] = topics[j], topics[i] })
			topics = topics[:3] // Keep only the first 3 topics
		}
	}

	h.WriteJSON(w, topics, 200)
}
