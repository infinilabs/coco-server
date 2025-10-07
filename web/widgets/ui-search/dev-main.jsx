import React, { useState, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { FullscreenPage } from "./src/index.jsx";

// 导入 UnoCSS 样式
import "uno.css";

// 简化版的 queryParams hook
function useSimpleQueryParams(defaultParams = {}) {
  const [params, setParams] = useState({
    from: 0,
    size: 10,
    sort: [],
    filter: {},
    ...defaultParams,
  });

  return [params, setParams];
}

// 开发环境组件
function DevApp() {
  const [queryParams, setQueryParams] = useSimpleQueryParams();
  const [queryParamsState, setQueryParamsState] = useState({
    from: 0,
    size: 10,
  });

  const enableQueryParams = true;

  // 模拟搜索 API
  const mockSearch = (query, callback, setLoading, shouldAgg = true) => {
    const res = {
      took: 4,
      hits: {
        total: {
          relation: "eq",
          value: 4,
        },
        max_score: 3.1079693,
        hits: [
          {
            _index: "coco_document-v2",
            _type: "_doc",
            _id: "d2alse8qlqbca26pbju0",
            _score: 3.1079693,
            _source: {
              category: "焊接",
              content:
                '| **专业分类**: | 焊接  | **标准**: | 《QJ1843A-96<br>结构钢、不锈钢熔焊工艺规范》 |\n| :--------| :---- | :---- |:----|\n|**禁用内容**:|禁止使用未充分烘干的焊条进行电弧焊熔焊。|**建议工艺**:|焊条使用前应按规定进行烘干，酸性焊条一般在150℃-200℃、1h-2h烘干;碱性焊条一般在300℃-400℃、1h-2h烘干。|\n禁止图片|<img src="http://coco.infini.cloud/bq/pics/27.jpg" width="260" height="200"> |推荐图片|<img src="http://coco.infini.cloud/bq/pics/28.jpg" width="260" height="200">|',
              created: "2025-08-08T02:17:29.394215628Z",
              icon: "http://coco.infini.cloud/bq/hanjie.png",
              id: "d2alse8qlqbca26pbju0",
              lang: "cn",
              last_updated_by: {
                timestamp: "2025-08-08T02:25:00Z",
                user: {
                  username: "liukj",
                },
              },
              owner: {
                username: "liukj",
              },
              size: 1048576,
              source: {
                id: "d2aloi8qlqbca26pbilg",
                name: "BQ",
                type: "connector",
              },
              summary:
                "禁用内容: 禁止使用未充分烘干的焊条进行电弧焊熔焊。 建议工艺: 焊条使用前应按规定进行烘干，酸性焊条一般在150℃-200℃、1h-2h烘干;碱性焊条一般在300℃-400℃、1h-2h烘干。 专业分类: 焊接 标准: 《QJ1843A-96<br>结构钢、不锈钢熔焊工艺规范》",
              tags: ["焊接"],
              title: "禁止使用未充分烘干的焊条进行电弧焊熔焊",
              type: "pdf",
              updated: "2025-08-08T02:45:38.382266717Z",
              url: "http://coco.infini.cloud/bq/1.GBT 22086-2008《铝及铝合金弧焊推荐工艺》.pdf",
            },
          },
          {
            _index: "coco_document-v2",
            _type: "_doc",
            _id: "d2alse8qlqbca26pbjug",
            _score: 2.9599512,
            _source: {
              category: "焊接",
              content:
                '| **专业分类**: | 焊接  | **标准**: | 《QJ2864B-2018<br>铝及铝合金熔焊工艺规范》；《QJI843A-96<br>结构钢、不锈钢熔焊工艺规范》 |\n| :--------| :---- | :---- |:----|\n|**禁用内容**:|熔焊焊接禁止在焊缝交叉处起弧、收弧:多层熔焊焊接各层处起弧、收弧位置严禁重叠。|**建议工艺**:|起弧和收弧应避开焊缝交叉处:多层或多道焊时起弧和收弧位置应错开。|\n禁止图片|<img src="http://coco.infini.cloud/bq/pics/29.jpg" width="260" height="200"> |推荐图片|<img src="http://coco.infini.cloud/bq/pics/30.jpg" width="260" height="200">|',
              created: "2025-08-08T02:17:29.556803343Z",
              icon: "http://coco.infini.cloud/bq/hanjie.png",
              id: "d2alse8qlqbca26pbjug",
              lang: "cn",
              last_updated_by: {
                timestamp: "2025-08-08T02:25:00Z",
                user: {
                  username: "liukj",
                },
              },
              owner: {
                username: "liukj",
              },
              size: 1048576,
              source: {
                id: "d2aloi8qlqbca26pbilg",
                name: "BQ",
                type: "connector",
              },
              summary:
                "禁用内容: 熔焊焊接禁止在焊缝交叉处起弧、收弧:多层熔焊焊接各层处起弧、收弧位置严禁重叠。 建议工艺: 起弧和收弧应避开焊缝交叉处:多层或多道焊时起弧和收弧位置应错开。 专业分类: 焊接 标准: 《QJ2864B-2018<br>铝及铝合金熔焊工艺规范》；《QJI843A-96<br>结构钢、不锈钢熔焊工艺规范》",
              tags: ["焊接"],
              title:
                "熔焊焊接禁止在焊缝交叉处起弧、收弧:多层熔焊焊接各层处起弧、收弧位置严禁重叠",
              type: "pdf",
              updated: "2025-08-08T02:45:39.149459334Z",
              url: "http://coco.infini.cloud/bq/1.GBT 22086-2008《铝及铝合金弧焊推荐工艺》.pdf",
            },
          },
          {
            _index: "coco_document-v2",
            _type: "_doc",
            _id: "d2alsdgqlqbca26pbjo0",
            _score: 0.8037008,
            _source: {
              category: "热处理",
              content:
                '| **专业分类**: | 热处理  | **标准**: | 《GB/T34883-2017<br>离子渗氦》 |\n| :--------| :---- | :---- |:----|\n|**禁用内容**:|禁止使用热导式电阻真空计测量离子渗氨的工作气压。|**建议工艺**:|一般采用薄膜式真空计测量离子渗氮的工作气压。|\n禁止图片|<img src="http://coco.infini.cloud/bq/pics/3.jpg" width="260" height="200"> |推荐图片|<img src="http://coco.infini.cloud/bq/pics/4.jpg" width="260" height="200">|',
              created: "2025-08-08T02:17:26.372827278Z",
              icon: "http://coco.infini.cloud/bq/jiare.png",
              id: "d2alsdgqlqbca26pbjo0",
              lang: "cn",
              last_updated_by: {
                timestamp: "2025-08-08T02:25:00Z",
                user: {
                  username: "liukj",
                },
              },
              owner: {
                username: "liukj",
              },
              size: 1048576,
              source: {
                id: "d2aloi8qlqbca26pbilg",
                name: "BQ",
                type: "connector",
              },
              summary:
                "禁用内容: 禁止使用热导式电阻真空计测量离子渗氨的工作气压。 建议工艺: 一般采用薄膜式真空计测量离子渗氮的工作气压。 专业分类: 热处理 标准: 《GB/T34883-2017<br>离子渗氦》",
              tags: ["热处理"],
              title: "禁止使用热导式电阻真空计测量离子渗氨的工作气压",
              type: "pdf",
              updated: "2025-08-08T02:45:18.088085437Z",
              url: "http://coco.infini.cloud/bq/2.GB 6514-2023《涂装作业安全规程 涂漆工艺安全及其通风》.pdf",
            },
          },
          {
            _index: "coco_document-v2",
            _type: "_doc",
            _id: "d2alse8qlqbca26pbjv0",
            _score: 0.6860195,
            _source: {
              category: "机械加工",
              content:
                '| **专业分类**: | 机械加工  | **标准**: | 《GB/T12611-2008<br>金属零(部)件镀覆前质量控制技术要求》 |\n| :--------| :---- | :---- |:----|\n|**禁用内容**:|需瓷质阳极化的铝合金零件精加工(表面粗糙度值小于Ra0.4)时，禁止采用乳化液冷却。|**建议工艺**:|采用煤油、珩磨油等无腐蚀性的冷却液。|\n禁止图片|<img src="http://coco.infini.cloud/bq/pics/31.jpg" width="260" height="200"> |推荐图片|<img src="http://coco.infini.cloud/bq/pics/32.jfif" width="260" height="200">|',
              created: "2025-08-08T02:17:29.919289601Z",
              icon: "http://coco.infini.cloud/bq/jixie.png",
              id: "d2alse8qlqbca26pbjv0",
              lang: "cn",
              last_updated_by: {
                timestamp: "2025-08-08T02:25:00Z",
                user: {
                  username: "liukj",
                },
              },
              owner: {
                username: "liukj",
              },
              size: 1048576,
              source: {
                id: "d2aloi8qlqbca26pbilg",
                name: "BQ",
                type: "connector",
              },
              summary:
                "禁用内容: 需瓷质阳极化的铝合金零件精加工(表面粗糙度值小于Ra0.4)时，禁止采用乳化液冷却。 建议工艺: 采用煤油、珩磨油等无腐蚀性的冷却液。 专业分类: 机械加工 标准: 《GB/T12611-2008<br>金属零(部)件镀覆前质量控制技术要求》",
              tags: ["机械加工"],
              title:
                "需瓷质阳极化的铝合金零件精加工(表面粗糙度值小于Ra0.4)时，禁止采用乳化液冷却",
              type: "pdf",
              updated: "2025-08-08T02:45:40.199695999Z",
              url: "http://coco.infini.cloud/bq/3.GBT 12611-2008《金属零（部）件镀覆前质量控制技术要求》.pdf",
            },
          },
        ],
      },
      aggregations: {
        category: {
          buckets: [
            {
              doc_count: 2,
              key: "焊接",
            },
            {
              doc_count: 1,
              key: "机械加工",
            },
            {
              doc_count: 1,
              key: "热处理",
            },
          ],
        },
        lang: {
          buckets: [
            {
              doc_count: 4,
              key: "cn",
            },
          ],
        },
        "source.id": {
          buckets: [
            {
              doc_count: 4,
              key: "d2aloi8qlqbca26pbilg",
              top: {
                hits: {
                  hits: [
                    {
                      _id: "d2alse8qlqbca26pbju0",
                      _index: "coco_document-v2",
                      _score: 3.1079693,
                      _source: {
                        source: {
                          name: "BQ",
                        },
                      },
                      _type: "_doc",
                    },
                  ],
                  max_score: 3.1079693,
                  total: {
                    relation: "eq",
                    value: 4,
                  },
                },
              },
            },
          ],
        },
        type: {
          buckets: [
            {
              doc_count: 4,
              key: "pdf",
            },
          ],
        },
      },
    };
    callback(res);
  };

  // 模拟 AI 助手 API - 参考 Fullscreen.jsx 的实现
  const mockAsk = async (assistantID, message, callback, setLoading) => {
    setLoading(true);

    try {
      // 首先返回初始消息创建响应
      const initialResponse = {
        "_id": "d3b3o50qlqbfo2h3q3bg",
        "_source": {
          "id": "d3b3o50qlqbfo2h3q3bg",
          "created": new Date().toISOString(),
          "updated": new Date().toISOString(),
          "_system": {
            "owner_id": "cvv85fk61mdus565iqig",
            "tenant_id": "cvv85fk61mdus565iqi0"
          },
          "status": "active",
          "title": JSON.stringify(message),
          "visible": false
        },
        "payload": {
          "id": "d3b3o50qlqbfo2h3q3c0",
          "created": new Date().toISOString(),
          "updated": new Date().toISOString(),
          "_system": {
            "owner_id": "cvv85fk61mdus565iqig",
            "tenant_id": "cvv85fk61mdus565iqi0"
          },
          "type": "user",
          "session_id": "d3b3o50qlqbfo2h3q3bg",
          "from": "",
          "message": JSON.stringify(message),
          "details": null,
          "up_vote": 0,
          "down_vote": 0,
          "assistant_id": assistantID
        },
        "result": "created"
      };

      callback(initialResponse);

      // 模拟流式响应的文本块
      const responseText = "The search results contain two PDF documents related to industrial standards and practices:\n\n1. **Thermal Treatment Standard (GB/T34883-2017)**:\n   - Prohibits the use of thermal conductivity resistance vacuum gauges for measuring ion nitriding working gas pressure.\n   - Recommends using thin-film vacuum gauges instead.\n   - Category: Thermal treatment.\n\n2. **Metal Parts Coating Quality Control (GB/T12611-2008)**:\n   - Prohibits the use of emulsion coolant for precision machining of aluminum parts requiring porcelain anodization (surface roughness less than Ra0.4).\n   - Suggests using non-corrosive coolants like kerosene or honing oil.\n   - Category: Mechanical processing.\n\nBoth documents provide specific guidelines on prohibited and recommended practices in their respective fields.";

      const words = responseText.split(' ');
      const sessionId = "d3b3o50qlqbfo2h3q3bg";
      const messageId = "d3b3o50qlqbfo2h3q3cg";
      const replyToMessage = "d3b3o50qlqbfo2h3q3c0";

      // 首先发送空的开始块
      callback({
        "session_id": sessionId,
        "message_id": messageId,
        "message_type": "assistant",
        "reply_to_message": replyToMessage,
        "chunk_sequence": 0,
        "chunk_type": "response",
        "message_chunk": ""
      });

      // 逐个发送单词块，模拟真实的流式响应
      for (let i = 0; i < words.length; i++) {
        await new Promise(resolve => setTimeout(resolve, 50 + Math.random() * 100)); // 随机延迟 50-150ms

        const chunk = i === 0 ? words[i] : ` ${words[i]}`;

        callback({
          "session_id": sessionId,
          "message_id": messageId,
          "message_type": "assistant",
          "reply_to_message": replyToMessage,
          "chunk_sequence": i + 1,
          "chunk_type": "response",
          "message_chunk": chunk
        });
      }

      // 发送结束标记
      await new Promise(resolve => setTimeout(resolve, 200));
      callback({
        "session_id": sessionId,
        "message_id": messageId,
        "message_type": "system",
        "reply_to_message": replyToMessage,
        "chunk_sequence": 0,
        "chunk_type": "reply_end",
        "message_chunk": "Processing completed"
      });

      setLoading(false);

    } catch (error) {
      setLoading(false);
      console.error('Mock ask error:', error);
    }
  };

  // 构建 componentProps，参考 Fullscreen.jsx 的结构
  const componentProps = {
    id: "dev-ui-search",
    shadow: null,
    logo: {
      light: "/favicon.ico",
      "light-mobile": "/favicon.ico",
    },
    placeholder: "搜索任何内容...",
    welcome:
      "欢迎使用 UI Search 开发环境！您可以在这里测试搜索功能和 AI 助手。",
    aiOverview: {
      enabled: true,
      showActions: true,
      assistant: "dev-assistant",
      title: "AI 概览",
      height: "400px",
    },
    widgets: [],
    onSearch: mockSearch,
    onAsk: mockAsk,
    config: {
      aggregations: {
        "source.id": {
          displayName: "source",
        },
        lang: {
          displayName: "language",
        },
        category: {
          displayName: "category",
        },
        type: {
          displayName: "type",
        },
      },
    },
  };

  const queryParamsProps = enableQueryParams
    ? {
        queryParams,
        setQueryParams,
      }
    : {
        queryParams: queryParamsState,
        setQueryParams: setQueryParamsState,
      };

  return (
    <FullscreenPage
      {...componentProps}
      {...queryParamsProps}
      enableQueryParams={enableQueryParams}
    />
  );
}

const root = createRoot(document.getElementById("root"));
root.render(<DevApp />);
