---
title: "Integration"
weight: 90
---

# Integration

## Work with *Integration*

The integration generates a piece of code based on certain configuration parameters, which can be embedded into other websites. This code allows you to quickly use Coco AI's search and chat capabilities.

## Integration API
Below is the field description for the integration.

| **Field**                       | **Type**        | **Description**                                                                                              |
|---------------------------------|-----------------|--------------------------------------------------------------------------------------------------------------|
| `name`                          | `string`        | The integration's name.                                                                                      |
| `type`                          | `string`        | The integration type. Possible values: `embedded`, `floating`, `all`.                                        |
| `datasource`                    | `array[string]` | List of datasource ID associated with the integration. e.g., `["cvei87tath20t2e51cag"]`.                     |
| `enabled_module.search`         | `object`        | Configuration for the search module, e.g., `{"enabled": true,"placeholder": "Search whatever you want..."}`. |
| `enabled_module.ai_chat`        | `object`        | Configuration for the AI chat module, e.g., `{"enabled": true,"placeholder": "Ask whatever you want..."}`    |
| `enabled_module.features`       | `array[string]` | List of enabled features, e.g., `["think_active","search_active","chat_history"]`.                           |
| `access_control.authentication` | `boolean`       | Enables or disables authentication.                                                                          |
| `access_control.chat_history`   | `boolean`       | Enables or disables chat history.                                                                            |
| `appearance.theme`              | `string`        | The display theme. Options: `auto`, `light`, `dark`. e.g., `auto`.                                           |
| `cors.enabled`                  | `boolean`       | Enables or disables CORS requests.                                                                           |
| `cors.allowed_origins`          | `array[string]` | List of allowed origins for CORS requests.                                                                   |
| `description`                   | `string`        | A brief description of the integration.                                                                      |
| `enabled`                       | `boolean`       | Enables or disables the integration.                                                                         |

### Create a Integration

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/integration/ -d'
{
     "type": "embedded",
    "name": "test_local",
    "datasource": [
      "d895f22ed2ff25ad8c6080af1cc23a21"
    ],
    "enabled_module": {
      "search": {
        "enabled": true,
        "placeholder": "Search whatever you want..."
      },
      "ai_chat": {
        "enabled": true,
        "placeholder": "Ask whatever you want..."
      },
      "features": [
        "think_active",
        "search_active",
        "chat_history"
      ]
    },
    "access_control": {
      "authentication": true,
      "chat_history": false
    },
    "appearance": {
      "theme": "auto"
    },
    "cors": {
      "enabled": true,
      "allowed_origins": [
        "http://localhost:8080"
      ]
    },
    "description": "test website",
    "enabled": true
}'

//response
{
  "_id": "cvj9s15ath21fvf9st00",
  "result": "created"
}
```

### View a Integration
```shell
curl -XGET http://localhost:9000/integration/cvj9s15ath21fvf9st00
```


### Delete the Integration

```shell
//request
curl  -H 'Content-Type: application/json'   -XDELETE http://localhost:9000/integration/cvj9s15ath21fvf9st00 

//response
{
  "_id": "cvj9s15ath21fvf9st00",
  "result": "deleted"
}'
```


### Update a Integration
```shell
curl -XPUT http://localhost:9000/integration/cvj9s15ath21fvf9st00 -d '{
    "type": "floating",
    "name": "test_local",
    "datasource": [
      "d895f22ed2ff25ad8c6080af1cc23a21"
    ],
    "enabled_module": {
      "search": {
        "enabled": true,
        "placeholder": "Search whatever you want..."
      },
      "ai_chat": {
        "enabled": true,
        "placeholder": "Ask whatever you want..."
      },
      "features": [
        "think_active",
        "search_active",
        "chat_history"
      ]
    },
    "access_control": {
      "authentication": true,
      "chat_history": false
    },
    "appearance": {
      "theme": "auto"
    },
    "cors": {
      "enabled": true,
      "allowed_origins": [
        "http://localhost:8080"
      ]
    },
    "description": "test website",
    "enabled": true
}'

//response
{
  "_id": "cvj9s15ath21fvf9st00",
  "result": "updated"
}
```

### Search Integrations
```shell
curl -XGET http://localhost:9000/integration/_search
```

### Update Suggested Topics

```
curl -H'X-API-TOKEN: cvl0qcjq50k2ddtq9230s1sl7oie0b1jrc06caznl51i1fb26wukbf4ftmwdfv4v57dm0qx6raqnaots8owt'  -XPOST http://localhost:9001/integration/cvj9knjq50k3mcuegatg/chat/_suggest -d'["Hello world", "What is Coco AI?", "What is INFINI Pizza?"]'
```

### View Suggested Topics

```
curl -H'X-API-TOKEN: cvl0qcjq50k2ddtq9230s1sl7oie0b1jrc06caznl51i1fb26wukbf4ftmwdfv4v57dm0qx6raqnaots8owt'  -XGET http://localhost:9001/integration/cvj9knjq50k3mcuegatg/chat/_suggest
```


## Integration UI Management

### Search Integration
Log in to the Coco-Server admin dashboard, click `Integration` in the left menu to view all Integration lists, as shown below:  
{{% load-img "/img/integration/list.png" "integration list" %}}

Enter keywords in the search box above the list and click the `Refresh` button to search for matching Integrations, as shown below:  
{{% load-img "/img/integration/filter-list.png" "integration search" %}}


### Add Integration
Click `Add` in the top-right corner of the list to create a new Integration, as shown below:  
{{% load-img "/img/integration/add-1.png" "add integration" %}}  
{{% load-img "/img/integration/add-2.png" "add integration" %}}

The system provides default values for the Integration configuration. Modify these values as needed, then click the save button to complete the creation.


### Delete Integration
Select the target Integration in the list, click `Delete` on the right side of the entry, and confirm in the pop-up dialog to complete the deletion. As shown below:  
{{% load-img "/img/integration/delete.png" "delete integration" %}}


### Edit Integration
Select the target Integration in the list, click `Edit` on the right side to enter the editing page. Modify the configuration and click save to update. As shown below:  
{{% load-img "/img/integration/edit.png" "edit integration" %}}


### Preview Integration
Click the `Preview` button on the right side of the Integration editing page to see the current Integration's effect, as shown below:  
{{% load-img "/img/integration/preview.png" "preview integration" %}}

The preview feature allows testing search and chat functionalities.  