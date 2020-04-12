# AWS-ELK Stack

[![ELK Logo](./logo.png "[ELK Logo")](https://www.elastic.co/what-is/elk-stack)

# Table of Contents
1. [Description](#Description)
2. [Deployment](#Deployment)
3. [CLI](#CLI)
4. [Kibana](#Kibana)
5. [API reference](#API-reference)

	5.1 [ElasticSearch cURL Commands](#ElasticSearch-cURL-Commands)

	5.2	[DSL Queries](#DSL-Queries)
---

## Description
ELK stack is data analysis orchestration service that runs three different components (Elasticsearch, Logstash & Kibana) to feed data, process and do indexation and finally display on a frontend client that enables end user to create different visualizations of the data.

Diferente user cases for ELK might be:
- Consolidate different logs from different sources with same schema or different and try to visualize at glance for audit purposes.

- Identify KPIs to given stakeholders the right tools in order to make strategic decisions on daily basis

- Apply Machine Learning techniques on custom subsets of data and get insights about possible data issues

This AWS-ELK is just an ELK stack that utilizes the Elasticsearch to consume logs from AWS services like CloudTrail & VPCFlowLogs.

---

## Deployment
The ELK stack is setup in multicontainer and then running on ECS service. 

---

## CLI

Open a console under same path as templates.

Files:
* deployment/templates/ecs-deployment-host.yaml
* deployment/templates/parameters.cfg
* deployment/templates/ecs-deployment-fargate.yaml
* deployment/templates/parameters-fargate.cfg

#### Cloudformation

You can create using EC2 registered instance or Fargate:

##### EC2 Instance cluster
```
aws --profile nc-inaki cloudformation deploy --template-file ecs-deployment-host.yaml --stack-name ecs-elk --parameter-overrides $(cat parameters-host.cfg) --capabilities CAPABILITY_NAMED_IAM
```

##### Fargate Instance cluster
```
aws --profile nc-inaki cloudformation deploy --template-file ecs-deployment-fargate.yaml --stack-name ecs-elk --parameter-overrides $(cat parameters-fargate.cfg) --capabilities CAPABILITY_NAMED_IAM
```

---

## Kibana

Before use Kibana, there is a requirement to create an index pattern. So, for that this steps will create an index and push some data.

#### Initial setup

1. Create a temp variable. (Obviously this can be arranged with a permanent CNAME record on R53 service that points to the ELB DNS)
```
ELK_HOST=$(aws --profile MY-PROFILE cloudformation describe-stacks --stack-name ecs-elk --query Stacks[].Outputs[0].OutputValue | sed -n 2,2p | cut -b 6-71)
```

2. Test the access url for Kibana frontend and Elasticsearch:

```
curl -f http://$ELASTICSEARCH
```
```
curl -f http://$ELASTICSEARCH:9200
```

3. Create VPCFlowlogs index
```
curl $ELASTICSEARCH:9200/vpclogs?pretty -H 'Content-Type: application/json' -d'{"mappings": {"doc": {"properties": {"account-id": {"type": "long"},"protocol": {"type": "integer"},"srcaddr": {"type": "keyword"},"dstaddr": {"type": "keyword"},"start": {"type": "date"},"end": {"type": "date"}}}}}' -XPUT
```

4. Now everything is ready to dump data. Checkout [Golang](#Golang)

5. Upon index creation there are different actions that should be arrange to get some pretty data visualizations on Kibana:
Open Kibana:
```
open http://$ELASTICSEARCH
```

http://ecs-f-appli-1tpvq334ofn26-1505576480.eu-central-1.elb.amazonaws.com/api/saved_objects/index-pattern

{"attributes":{"title":"vpclogs","timeFieldName":"start"}}

```
curl -X POST $ELASTICSEARCH/api/saved_objects/index-pattern/vpclogs -H 'Content-Type: application/json' -d'
{
  "attributes": {
    "title": "vpc*",
    "timeFieldName":"start"
  }
}
```


#### Create searches

	TODO

#### Create visualization

	TODO

#### Create Dashboard

	TODO

---

## API reference

All CRUD actions to create and manipulate data processed in EalsticSearch can be performed either using: 

* Console on Kibana at "Menu" - "DEV tools"

* Implement any RESTful Api client, e.g. cURL commands

---

## ElasticSearch cURL Commands
You can run any command using the following syntax:

```
$curl <PROTOCOL>://<HOST>:<PORT>/<PATH>/<OPERATION_NAME>?<QUERY_STRING> -d '<BODY>' -X<VERB>
```

> VERB: This can take values for the request method type: GET, POST, PUT, DELETE, HEAD.

> PROTOCOL: This is either http or https.

> HOST: This is the hostname of the node in the cluster. For local installations, this can be 'localhost' or '127.0.0.1'.

> PORT: This is the port on which the Elasticsearch instance is currently running. The default is 9200.

> PATH: This corresponds to the name of the index, type, and ID to be queried, for example: /index/type/id.

> OPERATION_NAME: This corresponds to the name of the operation to be performed, for example: _search, _count, and so on.

> QUERY_STRING: This is an optional parameter to be specified for query string parameters. For example, ?pretty for pretty print of JSON documents.

> BODY: This makes a request for body text.

#### Create index structure in Kibana
```
curl $ELASTICSEARCH:9200/vpclogs?pretty -H 'Content-Type: application/json' -d'{"mappings": {"doc": {"properties": {"account-id": {"type": "long"},"protocol": {"type": "integer"},"srcaddr": {"type": "keyword"},"dstaddr": {"type": "keyword"},"start": {"type": "date"},"end": {"type": "date"}}}}}' -XPUT
```

#### Bulk import the data to the right index
```
curl -H 'Content-Type: application/x-ndjson' $ELASTICSEARCH:9200/vpclogs/doc/_bulk?pretty --data-binary @tmpfile.json -XPOST
```

#### Check All available indexes:
```
curl http://$ELASTICSEARCH:9200/_cat/indices?v -XGET
```

#### List all nodes in a cluster:
```
curl http://$ELASTICSEARCH:9200/_cat/nodes?v -XGET
```

#### Check health of the cluster:
```
curl http://$ELASTICSEARCH:9200/_cluster/health?pretty=true -XGET
```

#### Check specific health level of the cluster:
```
curl http://$ELASTICSEARCH:9200/_cluster/health?level=cluster&pretty=true -XGET
curl http://$ELASTICSEARCH:9200/_cluster/health?level=shards&pretty=true -XGET
curl http://$ELASTICSEARCH:9200/_cluster/health?level=indices&pretty=true -XGET
```

#### Create index
```
curl $ELASTICSEARCH:9200/<index_name>?pretty -XPUT
```

#### Get items
```
curl $ELASTICSEARCH:9200/<index_name>/<index_type>/<item_id>?pretty -XGET
```

#### Delete document
```
curl $ELASTICSEARCH:9200/<index_name>/<index_type>/<item_id>?pretty -XDELETE
```

#### Delete All
```
curl $ELASTICSEARCH:9200/vpclogs/_delete_by_query?pretty -H 'Content-Type: application/json' -d'{"query":{"match_all":{}}}' -XPOST
```

```
curl $ELASTICSEARCH:9200/cloudtraillogs/_delete_by_query?pretty -H 'Content-Type: application/json' -d'{"query":{"match_all":{}}}' -XPOST
```

#### Get current ID

```
curl $ELASTICSEARCH:9200/vpclogs/_search?pretty -H 'Content-Type: application/json' -d '{"stored_fields": ["_id"],"query": {"match_all": {}},"sort": {"_id": "desc"},"size": 1}' -XGET

curl $ELASTICSEARCH:9200/vpclogs/_search?pretty -H 'Content-Type: application/json' -d '{"stored_fields": ["_id"],"query": {"match_all": {}},"sort": {"_id": "asc"},"size": 1}' -XGET
```

---

## DSL Queries

The syntax reference can be found in following link:

https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl.html

#### Create index
```
PUT /vpclogs
{
 "mappings": {
  "doc": {
   "properties": {
    "account-id": {"type": "long"},
    "protocol": {"type": "integer"},
    "srcaddr": {"type": "keyword"},
    "dstaddr": {"type": "keyword"},
    "start": {"type": "date"},
    "end": {"type": "date"}
   }
  }
 }
}
```

```
PUT /cloudtraillogs
{
 "mappings": {
  "doc": {
   "properties": {
    "eventVersion": {"type": "long"},
    "userIdentity.type": {"type": "keyword"},
    "userIdentity.invokedBy": {"type": "keyword"},
    "eventTime": {"type": "date"},
    "eventSource": {"type": "keyword"},
    "eventName": {"type": "keyword"},
    "awsRegion": {"type": "keyword"},
    "sourceIPAddress": {"type": "keyword"},
    "userAgent": {"type": "keyword"},
    "requestParameters.roleArn": {"type": "keyword"},
    "requestParameters.roleSessionName": {"type": "keyword"},
    "requestParameters.externalId": {"type": "keyword"},
    "requestParameters.durationSeconds": {"type": "long"},
    "responseElements.credentials.accessKeyId": {"type": "keyword"},
    "responseElements.credentials.expiration": {"type": "date"},
    "responseElements.credentials.sessionToken": {"type": "text"},
    "assumedRoleUser.assumedRoleId": {"type": "keyword"},
    "assumedRoleUser.arn": {"type": "keyword"},
    "requestID": {"type": "keyword"},
    "eventID": {"type": "keyword"},
    "resources": {"type": "keyword"},
    "eventType": {"type": "keyword"},
    "recipientAccountId": {"type": "keyword"},
    "sharedEventID": {"type": "keyword"}
   }
  }
 }
}
```

#### Get data values
```
GET /vpclogs/_search?pretty
{
  "query": {
    "bool" : {
      "must" : {
        "range": {
          "start": {
            "gte": "2020-03-22",
            "lt": "2020-03-24"
          }
        }
      },
      "filter": {
        "term" : { "account-id" : "007385363882" }
      }
    }
  }
}
```

#### Delete by query
```
POST /vpclogs/_delete_by_query?pretty
{
    "query": {
      "match": {
        "account-id": "007385363882"
      }
    }
}
```

```
POST /vpclogs/_delete_by_query?pretty
{
  "query": {
    "bool" : {
      "must" : {
        "range": {
          "start": {
            "gte": "2020-03-22",
            "lt": "2020-03-24"
          }
        }
      },
      "filter": {
        "term" : { "account-id" : "007385363882" }
      }
    }
  }
}
```
