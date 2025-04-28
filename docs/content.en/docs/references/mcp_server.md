---
title: "MCP Server"
weight: 90
---

# MCP Server

## Work with *MCP Server*

TThe MCP Server provides commonly used MCP configurations for large model calls.

## MCP Server API

Below is the field description for the MCP Server.

| **Field**  | **Type**  | **Description**                                                                                                                   |
|------------|-----------|-----------------------------------------------------------------------------------------------------------------------------------|
| `name`     | `string`  | The MCP Server's name.                                                                                                            |
| `type`     | `string`  | The type to access the API of the MCP Server, possible values: stdio, streamable http.                                            |
| `icon`     | `string`  | The icon representing the MCP Server in the UI.                                                                                   |
| `category` | `string`  | The category of the MCP Server.                                                                                                   |
| `icon`     | `string`  | The icon representing the model provider in the UI.                                                                               |
| `config`   | `object`  | The specify config for the type, e.g., {"args":["@playwright/mcp@latest", "--headless"], "command": "npx", "env": {"key1":"v1"}}. |
| `enabled`  | `boolean` | Enables or disables MCP Server.                                                                                                   |

### Create a model provider

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/mcp_server/ -d '{
  "name" : "playwright",
  "icon" : "font_robot",
  "type" : "stdio",
  "category" : "Network Tools",
  "config" : {
    "args" : [
      "@playwright/mcp@latest",
      "--headless"
    ],
    "command" : "npx",
    "env" : { }
  },
  "enabled" : true
}'

//response
{
  "_id": "d07jnoac7k8ff566lar0",
  "result": "created"
}
```

### View a MCP Server

```shell
curl -XGET http://localhost:9000/mcp_server/d07jnoac7k8ff566lar0
```

### Delete the MCP Server

```shell
//request
curl  -H 'Content-Type: application/json'   -XDELETE http://localhost:9000/mcp_server/d07jnoac7k8ff566lar0 

//response
{
  "_id": "d07jnoac7k8ff566lar0",
  "result": "deleted"
}'
```

### Update a Model Provider

```shell
curl -XPUT http://localhost:9000/mcp_server/d07jnoac7k8ff566lar0 -d '{
   "name" : "playwright",
  "icon" : "font_robot",
  "type" : "stdio",
  "category" : "Network Tools",
  "config" : {
    "args" : [
      "@playwright/mcp@latest",
      "--headless"
    ],
    "command" : "npx",
    "env" : { }
  },
  "enabled" : true
}'

//response
{
  "_id": "d07jnoac7k8ff566lar0",
  "result": "updated"
}
```

### Search MCP Servers

```shell
curl -XGET http://localhost:9000/mcp_server/_search
```

## MCP Server UI Management

### Search MCP Server
Log in to the Coco-Server admin dashboard, click `MCP Server` in the left menu to view all MCP Server lists, as shown below:  
{{% load-img "/img/mcp-server/list.png" "MCP server list" %}}

> Here, we can also choose an MCP Server from the list and quickly disable or enable it by toggling the switch.

Enter keywords in the search box above the list and click the `Refresh` button to search for matching MCP Server, as shown below:  
{{% load-img "/img/mcp-server/filter-list.png" "MCP server search" %}}


### Add MCP Server
Click `Add` in the top-right corner of the list to create a new MCP Server, as shown below:  
{{% load-img "/img/mcp-server/add-1.png" "add MCP Server" %}}  
> Here, we select the type stdio, and the args and env should be configured with one item per line.
{{% load-img "/img/mcp-server/add-2.png" "add MCP Server" %}}
> Here, we select the type `Streamable HTTP`, and we just need to configure a URL for the MCP Server.

### Delete MCP Server
Select the target MCP Server in the list, click `Delete` on the right side of the entry, and confirm in the pop-up dialog to complete the deletion. As shown below:  
{{% load-img "/img/mcp-server/delete.png" "delete MCP Server" %}}


### Edit MCP Server
Select the target MCP Server in the list, click `Edit` on the right side to enter the editing page. Modify the configuration and click save to update. As shown below:  
{{% load-img "/img/mcp-server/edit.png" "edit MCP Server" %}}