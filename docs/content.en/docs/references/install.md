---
title: "System Initialization"
weight: 500
---

# System Initialization

```
curl -XPOST http://localhost:9000/setup/_initialize -d'
{
	"name":"Coco",
	"email":"hello@coco.rs",
	"password":"mypassword",
	"llm":{
		  "type":"ollama",
		  "endpoint":"http://xxx",
		  "default_model":"deepseek_r1"
	}
}'
```