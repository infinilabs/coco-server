---
title: "Modify Password"
weight: 50
---

# Modify Password

Modify the current user's password.

```
curl -XPUT http://localhost:9000/account/password -d'{
	"old_password":"xxxx",
	"new_password":"xxxx"
}'
```