import React, { useState, useEffect } from "react";
import "./src/i18n"; // Initialize i18n
import { createRoot } from "react-dom/client";
import { FullscreenPage } from "./src/index.jsx";

// 导入 UnoCSS 样式
import "uno.css";
import { Dropdown } from "antd";

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

  const [activeChat, setActiveChat] = useState("1");
  const [chats, setChats] = useState([
    {
      _id: "1",
      _source: {
        title: "Initial Chat",
        created: new Date().toISOString(),
        updated: new Date().toISOString()
      },
      messages: [
        {
          _id: "1-1",
          _source: {
            type: "assistant",
            message: "Hello! I am your AI assistant. How can I help you today?",
            created: new Date().toISOString(),
            user: { username: "Assistant" }
          }
        },
        {
          _id: "1-2",
          _source: {
            type: "user",
            message: "what is coco?",
            created: new Date().toISOString(),
            user: { username: "User" },
            attachments: ["1", "2", "3"]
          }
        }
      ]
    },
    {
      _id: "2",
      _source: {
        title: "Welding Standards Inquiry",
        created: new Date(Date.now() - 86400000).toISOString(),
        updated: new Date(Date.now() - 86400000).toISOString()
      },
      messages: [
        {
          _id: "2-1",
          _source: {
            type: "user",
            message: "Show me some welding standards.",
            created: new Date().toISOString(),
            user: { username: "User" }
          }
        },
        {
          _id: "2-2",
          _source: {
            type: "assistant",
            message: "Here are some relevant details:\n\n- **Standard**: QJ1843A-96\n- **Category**: Welding",
            created: new Date().toISOString(),
            user: { username: "Assistant" }
          }
        }
      ]
    },
    {
      _id: "3",
      _source: {
        title: "Previous Week Discussion",
        created: new Date(Date.now() - 604800000).toISOString(),
        updated: new Date(Date.now() - 604800000).toISOString()
      },
      messages: []
    }
  ]);

  const onHistorySelect = (chat) => {
    const chatId = chat._id || chat;
    console.log("History selected:", chatId);
    setActiveChat(chatId);
  };

  const handleSendMessage = async (content) => {
    if (!activeChat) return;

    const newMessage = {
      _id: Date.now().toString(),
      _source: {
        type: "user",
        message: content,
        created: new Date().toISOString(),
        user: { username: "User" }
      }
    };

    setChats(prevChats => prevChats.map(chat => {
      if (chat._id === activeChat) {
        return {
          ...chat,
          messages: [...(chat.messages || []), newMessage]
        };
      }
      return chat;
    }));

    // Simulate assistant response
    setTimeout(() => {
        const assistantMsg = {
            _id: (Date.now() + 1).toString(),
            _source: {
                type: "assistant",
                message: "I received your message: " + content + "\n\nBased on my analysis, here is a comprehensive answer.\n\nCoco AI is designed to help you find information quickly and efficiently.",
                created: new Date().toISOString(),
                user: { username: "Assistant" },
                details: [
                    {
                        type: "query_intent",
                        payload: {
                            category: "General Inquiry",
                            intent: "User Interaction",
                            query: [content],
                            keyword: ["interaction", "test"],
                            suggestion: ["Tell me more about Coco AI", "How does search work?"]
                        }
                    },
                    {
                        type: "think",
                        description: "The user has sent a message. I need to acknowledge it and provide a relevant response.\n\n1. Analyze input content.\n2. Retrieve relevant knowledge.\n3. Formulate response."
                    },
                    {
                        type: "fetch_source",
                        payload: [
                            {
                                id: "doc_1",
                                title: "Coco AI Documentation",
                                summary: "Official documentation for Coco AI features and usage.",
                                url: "https://docs.coco-ai.com"
                            },
                            {
                                id: "doc_2",
                                title: "User Guide",
                                summary: "Comprehensive guide for new users.",
                                url: "https://guide.coco-ai.com"
                            }
                        ]
                    }
                ]
            }
        };
        setChats(prevChats => prevChats.map(chat => {
            if (chat._id === activeChat) {
                return {
                ...chat,
                messages: [...(chat.messages || []), assistantMsg]
                };
            }
            return chat;
        }));
    }, 1000);
  };

  const currentChatObj = chats.find(c => c._id === activeChat);
  const currentMessages = currentChatObj ? currentChatObj.messages : [];


  const onHistorySearch = (query) => {
    console.log("History search:", query);
    // Simulate client-side filtering for dev
    // In real app, this might be a server call or just local filtering
  };

  const onHistoryRefresh = async () => {
    console.log("History refreshing...");
    await new Promise(resolve => setTimeout(resolve, 1000));
    console.log("History refreshed");
  };

  const onHistoryRename = async (chatId, newTitle) => {
    console.log("Rename chat:", chatId, newTitle);
    setChats(chats.map(chat =>
      chat._id === chatId
        ? { ...chat, _source: { ...chat._source, title: newTitle } }
        : chat
    ));
  };

  const onHistoryRemove = async (chatId) => {
    console.log("Remove chat:", chatId);
    setChats(chats.filter(chat => chat._id !== chatId));
    if (activeChat === chatId) {
      setActiveChat(null);
    }
  };

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
              url: "https://gips1.baidu.com/it/u=3579958525,4293415030&fm=3074&app=3074&f=PNG?w=2560&h=1440",
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
          // image
          {
            _index: "coco_document-v2",
            _type: "_doc",
            _id: "d2alse8qlqbca26pbjv0",
            _score: 0.6860195,
            _source: {
              category: "壁纸",
              content: "",
              created: "2025-08-08T02:17:29.394215628Z",
              icon: "",
              id: "d2alse8qlqbca26pbju7",
              lang: "cn",
              last_updated_by: {
                timestamp: "2025-08-08T02:25:00Z",
                user: {
                  username: "test",
                },
              },
              owner: {
                username: "test",
              },
              size: 1048576,
              source: {
                id: "d2aloi8qlqbca26pbilg",
                name: "壁纸",
                type: "connector",
              },
              summary: "",
              tags: ["壁纸"],
              title: "黑色壁纸全屏🌌,探索星空的奥秘✨",
              type: "image",
              updated: "2025-08-08T02:45:38.382266717Z",
              thumbnail: "https://gips1.baidu.com/it/u=3579958525,4293415030&fm=3074&app=3074&f=PNG?w=2560&h=1440",
              url: "https://gips1.baidu.com/it/u=3579958525,4293415030&fm=3074&app=3074&f=PNG?w=2560&h=1440",
            },
          },
          {
            _index: "coco_document-v2",
            _type: "_doc",
            _id: "d2alse8qlqbca26pbjv0",
            _score: 0.6860195,
            _source: {
              category: "壁纸",
              content: "",
              created: "2025-08-08T02:17:29.394215628Z",
              icon: "",
              id: "d2alse8qlqbca26pbju1",
              lang: "cn",
              last_updated_by: {
                timestamp: "2025-08-08T02:25:00Z",
                user: {
                  username: "test",
                },
              },
              owner: {
                username: "test",
              },
              size: 1048576,
              source: {
                id: "d2aloi8qlqbca26pbilg",
                name: "壁纸",
                type: "connector",
              },
              summary: "",
              tags: ["壁纸"],
              title: "摄影壁纸创意图,捕捉山水间的灵动之美🏞️",
              type: "image",
              updated: "2025-08-08T02:45:38.382266717Z",
              thumbnail: "https://img1.baidu.com/it/u=3879890807,997649473&fm=253&fmt=auto&app=138&f=JPEG?w=889&h=500",
              url: "https://img1.baidu.com/it/u=3879890807,997649473&fm=253&fmt=auto&app=138&f=JPEG?w=889&h=500",
            },
          },
          {
            _index: "coco_document-v2",
            _type: "_doc",
            _id: "d2alse8qlqbca26pbjv0",
            _score: 0.6860195,
            _source: {
              category: "壁纸",
              content: "",
              created: "2025-08-08T02:17:29.394215628Z",
              icon: "",
              id: "d2alse8qlqbca26pbju8",
              lang: "cn",
              last_updated_by: {
                timestamp: "2025-08-08T02:25:00Z",
                user: {
                  username: "test",
                },
              },
              owner: {
                username: "test",
              },
              size: 1048576,
              source: {
                id: "d2aloi8qlqbca26pbilg",
                name: "壁纸",
                type: "connector",
              },
              summary: "",
              tags: ["壁纸"],
              title: "摄影壁纸创意图,捕捉山水间的灵动之美🏞️",
              type: "image",
              updated: "2025-08-08T02:45:38.382266717Z",
              thumbnail: "https://img2.baidu.com/it/u=1088560728,493918909&fm=253&app=138&f=JPEG?w=889&h=500",
              url: "https://img2.baidu.com/it/u=1088560728,493918909&fm=253&app=138&f=JPEG?w=889&h=500",
            }
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
    theme: 'light',
    language: 'zh-CN',
    logo: {
      // light: "/favicon.ico",
      // "light_mobile": "/favicon.ico",
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
    onLogoClick: () => {
      console.log('logo click')
    },
    messages: currentMessages,
    onSendMessage: handleSendMessage,
    // History props
    chats: chats,
    activeChat: activeChat,
    onHistorySelect: onHistorySelect,
    onHistorySearch: onHistorySearch,
    onHistoryRefresh: onHistoryRefresh,
    onHistoryRename: onHistoryRename,
    onHistoryRemove: onHistoryRemove,
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
