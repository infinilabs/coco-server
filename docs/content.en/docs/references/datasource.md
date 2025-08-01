---
title: "Datasource"
weight: 90
---

# Datasource

## Work with *Datasource*

Datasource defines where the data comes from, usually we can use a specify connector to fetch data from a specify datasource.

### Create a Datasource

We can use the connector to connect specify datasource.


```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/datasource/ -d'
{
    "name":"My Hugo Site",
    "type":"connector",
    "connector":{
        "id":"hugo_site",
         "config":{
            "urls": [ "https://pizza.rs/index.json" ]
        }
    }
}'

//response
{
  "_id": "cu1rf03q50k43nn2pi6g",
  "result": "created"
}
```

> `config` specifies the necessary configurations for this connector. For detailed information, refer to the [connector reference documentation](./connectors/).

### View a Datasource
```shell
curl -XGET http://localhost:9000/datasource/cu1rf03q50k43nn2pi6g
```


### Delete the Datasource

```shell
//request
curl  -H 'Content-Type: application/json'   -XDELETE http://localhost:9000/datasource/cu1rf03q50k43nn2pi6g -d'
{
  "_id": "cu1rf03q50k43nn2pi6g",
  "result": "deleted"
}'
```


### Update a Datasource
```shell
curl -XPUT http://localhost:9000/datasource/cu1rf03q50k43nn2pi6g?replace=true -d '{
    "name":"My Hugo Site",
    "type":"connector",
    "connector":{
        "id":"hugo_site",
         "config":{
            "urls": [ "https://pizza.rs/index.json" ]
        }
    }
}'

//response
{
  "_id": "cxx9vr3q50k38nobvmcg",
  "result": "updated"
}
```

> `?replace=true` can safely ignore errors for non-existent items.

### Search Datasources
```shell
curl -XGET http://localhost:9000/datasource/_search
```

### Insert Document to this Datasource

```shell
curl -H 'Content-Type: application/json'   -XPOST http://localhost:9000/datasource/cu1rf03q50k43nn2pi6g/_doc -d'
{
   "title": "Your title",
   ...OMITTED...
}'
```

### Insert Document to this Datasource With Identity

```shell
curl -H 'Content-Type: application/json'   -XPOST http://localhost:9000/datasource/cu1rf03q50k43nn2pi6g/_doc/myuuid123 -d'
{
   "title": "Your title",
   ...OMITTED...
}'
```

> Note: The ID must be unique across all datasources.