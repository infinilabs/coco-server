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
