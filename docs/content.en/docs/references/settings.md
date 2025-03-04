---
title: "System Settings"
weight: 510
---

# System Settings

```
curl -XPOST http://localhost:9000/settings -d'
{
   "server":{
   		"name": "My Coco Server",
        "endpoint":"http://xxxx/",
		"provider": {
			"banner": "http://localhost:9000/banner2.jpg",
			"description": "Coco AI Server - Search, Connect, Collaborate, AI-powered enterprise search, all in one space.",
			"eula": "http://infinilabs.com/eula.txt",
			"icon": "http://localhost:9000/icon.png",
			"name": "INFINI Labs",
			"privacy_policy": "http://infinilabs.com/privacy_policy.txt",
			"website": "http://infinilabs.com"
		},

   },
	"llm":{
		  "type":"ollama", //or openai
		  "endpoint":"http://xxx",
		  "default_model":"deepseek_r1",
		  "parameters":{
			  "top_p":111,
			  "max_tokens":32000,
			  "presence_penalty":0.9,
			  "frequency_penalty":0.9,
			  "enhanced_inference":true,
		  }
	}
}'
```