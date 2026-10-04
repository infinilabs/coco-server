#// Scenario 1: knowledge hub end-to-end — the live walkthrough as code.
#//
#// The unit suite covers the pure math (RRF, scoring, rewrite guards); this
#// scenario locks the full-stack wiring the walkthrough kept catching bugs
#// in: simhash persistence against a real long mapping, the published-layer
#// retrieval, the wiki/graph recall legs, the company map, the golden-query
#// evaluation flow (scoring + zero-recall gap reflux + idempotency), the
#// correction idempotency anchor, and the rewrite leg's degrade path.
#//
#// Deterministic by construction: the eval probes use scalar asserts only
#// (per-case arrays are not assertable), expectations name the two content
#// documents (exact-title BM25 wins rank 1 on any corpus), and the
#// forced-zero cases run while no page passes the wiki noise gate — every
#// recall leg is empty for them, pinning exact top4_rate/MRR values.


#//----------------------------------------------------------------------------
#// Login
#//----------------------------------------------------------------------------

POST /account/login
{
  "login": "$[[env.ADMIN_MAIL]]",
  "password": "$[[env.ADMIN_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
#
# register: [
#   { admin_token: "_ctx.response.body_json.access_token" },
# ]


#//----------------------------------------------------------------------------
#// dedicated datasource for the eval probes: the text leg scopes documents
#// by source.id, so the expected documents must live in a datasource the
#// admin can see — and a dedicated one makes the scoped recall deterministic
#//----------------------------------------------------------------------------

#// grab any builtin connector to hang a fresh, empty, enabled datasource
#// on — owned by the admin, so the scoped text leg below sees exactly the
#// two probe documents and nothing else
GET /connector/_search?size=1
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# register: [
#   { conn_id: "_ctx.response.body_json.hits.hits.0._id" },
# ]

POST /datasource/
{
  "type": "connector",
  "name": "knowledge-hub-it",
  "connector": {"id": "$[[conn_id]]"},
  "enabled": true,
  "sync": {"enabled": false}
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { ds_id: "_ctx.response.body_json._id" },
# ]


#//----------------------------------------------------------------------------
#// simhash regression: high-bit simhash values must persist as signed long
#// (the exact content that failed before the fix — 200, not a parse error)
#//----------------------------------------------------------------------------

POST /document/
{
  "title": "年假申请流程",
  "summary": "员工申请年假的步骤与审批链",
  "content": "年假申请需提前三个工作日在系统提交，由直属上级审批。工龄满五年者年假天数增加两天。"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),

POST /document/
{
  "title": "年终双薪发放政策",
  "summary": "员工年终双薪的发放条件、计算方式与发放时间",
  "content": "公司为考核合格的员工提供年终双薪。年终双薪按当年度实际在职月份折算，于次年一月随工资发放。试用期员工不享受年终双薪。"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { pay_doc_id: "_ctx.response.body_json._id" },
# ]


#//----------------------------------------------------------------------------
#// the two content documents the evaluation probes expect: exact-title BM25
#// puts them at rank 1 for both probe queries regardless of corpus noise
#//----------------------------------------------------------------------------

POST /document/
{
  "title": "支付服务运维手册",
  "summary": "支付服务的职责、依赖与运维要点",
  "content": "支付服务承接订单服务的支付请求，负责收款与对账。上游依赖：订单服务。运维要点：监控支付成功率与对账差异。",
  "source": {"id": "$[[ds_id]]", "name": "knowledge-hub-it"}
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { manual_doc: "_ctx.response.body_json._id" },
# ]

POST /document/
{
  "title": "支付链路故障排查",
  "summary": "支付失败超时的定位路径",
  "content": "当支付服务不可用时，订单服务订单滞留，用户无法完成付款。排查顺序：网关日志、支付服务状态、订单补偿队列。",
  "source": {"id": "$[[ds_id]]", "name": "knowledge-hub-it"}
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { trouble_doc: "_ctx.response.body_json._id" },
# ]


#//----------------------------------------------------------------------------
#// ontology vocabulary: declare product.depends_on so entity writes validate
#//----------------------------------------------------------------------------

PUT /wiki/ontology/schema
{
  "schema": {
    "entity_types": [
      {
        "name": "product",
        "label": "产品与服务",
        "icon": "📦",
        "properties": [
          {"key": "owner", "label": "负责人", "type": "string"}
        ],
        "relations": [
          {"name": "depends_on", "label": "依赖于", "target_type": "product", "inverse": "depended_by"}
        ]
      }
    ]
  }
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "updated"}),


#//----------------------------------------------------------------------------
#// knowledge base + entities + depends_on chain
#//----------------------------------------------------------------------------

POST /wiki/kb/
{
  "name": "知识中枢测试库",
  "description": "integration scenario",
  "visibility": "public"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { kb_id: "_ctx.response.body_json._id" },
# ]

POST /wiki/entity/
{
  "kb_id": "$[[kb_id]]",
  "name": "支付服务",
  "type": "product",
  "aliases": ["支付"],
  "status": "reviewed",
  "confidence": 0.9
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { e_pay: "_ctx.response.body_json._id" },
# ]

POST /wiki/entity/
{
  "kb_id": "$[[kb_id]]",
  "name": "订单服务",
  "type": "product",
  "status": "reviewed",
  "confidence": 0.9,
  "relations": [
    {"target_id": "$[[e_pay]]", "relation": "depends_on"}
  ]
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { e_order: "_ctx.response.body_json._id" },
# ]


#//----------------------------------------------------------------------------
#// company map: the only page published before the eval section. Map pages
#// never pass the wiki noise gate, so the wiki recall leg stays empty for
#// the forced-zero eval cases below — that is what makes them deterministic
#//----------------------------------------------------------------------------

POST /wiki/article/
{
  "kb_id": "$[[kb_id]]",
  "title": "公司地图",
  "summary": "scenario company map",
  "page_type": "map",
  "content": ""
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { a_map: "_ctx.response.body_json._id" },
# ]

PUT /wiki/article/$[[a_map]]/status
{
  "status": "reviewed"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "updated"}),

PUT /wiki/article/$[[a_map]]/status
{
  "status": "published"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "updated"}),

#// let the published layer settle before retrieval asserts
# sleep: {
#   sleep_in_milli_seconds: 3000,
# },

GET /wiki/company-map
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.found: true,
# },


#//----------------------------------------------------------------------------
#// hybrid retrieval through the production search endpoint: /query/_search
#// is the app/widget search (search.go handler); with search_type=hybrid_rrf
#// it runs the full RRF pipeline end-to-end and records one search log (so
#// the search_logs store exists for the index-health assert below).
#// /document/_search is the documents CRUD page's BM25 list search — it
#// ignores search_type and records nothing, don't seed through it.
#//----------------------------------------------------------------------------

GET /query/_search?query=%E5%B9%B4%E7%BB%88%E5%8F%8C%E8%96%AA&search_type=hybrid_rrf
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#


#//----------------------------------------------------------------------------
#// golden-query evaluation set: two recall probes naming the two content
#// documents + two forced-zero cases (unmatchable query + empty datasource
#// scope) that must reflux knowledge-gap proposals
#//----------------------------------------------------------------------------

POST /search/studio/eval/_cases
{
  "query": "支付服务运维手册要点",
  "expected_ids": ["$[[manual_doc]]", "$[[trouble_doc]]"],
  "expected_titles": ["支付服务运维手册", "支付链路故障排查"],
  "datasource": "$[[ds_id]]"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {query: "支付服务运维手册要点"}),

POST /search/studio/eval/_cases
{
  "query": "支付服务挂了影响什么",
  "expected_ids": ["$[[manual_doc]]", "$[[trouble_doc]]"],
  "expected_titles": ["支付服务运维手册", "支付链路故障排查"],
  "datasource": "$[[ds_id]]"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {query: "支付服务挂了影响什么"}),

POST /search/studio/eval/_cases
{
  "query": "xqzwjvqz",
  "expected_ids": ["$[[manual_doc]]"],
  "datasource": "no-such-datasource"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {query: "xqzwjvqz"}),

POST /search/studio/eval/_cases
{
  "query": "xqzwjvkqz",
  "expected_ids": ["$[[manual_doc]]"],
  "datasource": "no-such-datasource"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {query: "xqzwjvkqz"}),

POST /search/studio/eval/_run
{}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.run.total_cases: 4,
#   _ctx.response.body_json.run.top4_hits: 2,
#   _ctx.response.body_json.run.top4_rate: 0.5,
#   _ctx.response.body_json.run.mrr: 0.5,
# },

GET /wiki/governance/_search?filter=type%3Aknowledge_gap
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.hits.total.value: 2,
# },

#// rerun: same zero-recall queries must refresh, never duplicate
POST /search/studio/eval/_run
{}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.run.total_cases: 4,
#   _ctx.response.body_json.run.top4_rate: 0.5,
# },

GET /wiki/governance/_search?filter=type%3Aknowledge_gap
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.hits.total.value: 2,
# },


#//----------------------------------------------------------------------------
#// correction idempotency: same report twice files one proposal and bumps
#// its report_count (D7)
#//----------------------------------------------------------------------------

POST /wiki/governance/_correction
{
  "query": "支付服务挂了影响什么",
  "answer_excerpt": "支付服务只影响退款流程",
  "route_hint": "fact_missing",
  "comment": "支付服务故障也阻断下单，答案不完整"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {report_count: 1}),

POST /wiki/governance/_correction
{
  "query": "支付服务挂了影响什么",
  "answer_excerpt": "支付服务只影响退款流程",
  "route_hint": "fact_missing",
  "comment": "支付服务故障也阻断下单，答案不完整"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {report_count: 2}),

GET /wiki/governance/_search?filter=type%3Acorrection
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.hits.total.value: 1,
# },


#//----------------------------------------------------------------------------
#// now publish the dual-source curated pages — the full wiring: create →
#// reviewed → published, sources attached, entity page links the ontology.
#// The eval ran before this on purpose: its numbers must not depend on the
#// wiki layer
#//----------------------------------------------------------------------------

POST /wiki/article/
{
  "kb_id": "$[[kb_id]]",
  "title": "支付服务运维手册",
  "summary": "支付服务的职责、依赖与运维要点",
  "page_type": "entity",
  "entity_id": "$[[e_pay]]",
  "content": "支付服务承接订单服务的支付请求。\n\n- 上游：订单服务\n",
  "tags": ["支付", "运维"],
  "confidence": "high",
  "sources": [
    {"doc_id": "$[[pay_doc_id]]", "source_type": "document", "title": "年终双薪发放政策", "locator": "section:1"},
    {"doc_id": "simulated-doc", "source_type": "document", "title": "支付服务架构说明", "locator": "section:2"}
  ]
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { a_pay: "_ctx.response.body_json._id" },
# ]

PUT /wiki/article/$[[a_pay]]/status
{
  "status": "reviewed"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "updated"}),

PUT /wiki/article/$[[a_pay]]/status
{
  "status": "published"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "updated"}),

POST /wiki/article/
{
  "kb_id": "$[[kb_id]]",
  "title": "支付链路故障排查",
  "summary": "支付失败超时的定位路径",
  "page_type": "concept",
  "content": "当支付服务不可用时，订单服务订单滞留。\n",
  "tags": ["故障排查"],
  "confidence": "high",
  "sources": [
    {"doc_id": "$[[pay_doc_id]]", "source_type": "document", "title": "年终双薪发放政策", "locator": "section:1"},
    {"doc_id": "simulated-doc-2", "source_type": "document", "title": "支付链路事故复盘", "locator": "section:3"}
  ]
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "created"}),
# register: [
#   { a_trouble: "_ctx.response.body_json._id" },
# ]

PUT /wiki/article/$[[a_trouble]]/status
{
  "status": "reviewed"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "updated"}),

PUT /wiki/article/$[[a_trouble]]/status
{
  "status": "published"
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: (200, {result: "updated"}),


#//----------------------------------------------------------------------------
#// rewrite leg degrade path: no reachable language model -> applied=false,
#// the search itself must not fail
#//----------------------------------------------------------------------------

POST /search/studio/test
{
  "query": "支付服务挂了影响什么",
  "size": 5,
  "rrf": {
    "k": 60,
    "text_weight": 1,
    "semantic_weight": 1,
    "wiki_weight": 1,
    "graph_weight": 1,
    "rewrite_weight": 1
  }
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.rewrite.applied: false,
# },


#//----------------------------------------------------------------------------
#// index health: every knowledge-hub store answers, including the two
#// evaluation stores
#//----------------------------------------------------------------------------

GET /search/ops/index-health
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
# },
#
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.healthy: 8,
#},
