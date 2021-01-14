package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
)

var S3Client *s3.S3
var ItemId int64 = 0

// GetS3FileNames ...
func GetS3FileNames(dateToMatch string, token string, awsAccountId string, s3Keys []string, logType string) ([]string, string) {
	//fmt.Println("GetS3FileNames ...")
	//svc := s3.New(accountsess, aws.NewConfig().WithRegion("eu-central-1"))
	listFoldersInput := &s3.ListObjectsV2Input{}

	switch logType {
	case VpcFlowLog:
		listFoldersInput.SetPrefix("AWSLogs/" + awsAccountId + "/vpcflowlogs/eu-central-1/" + dateToMatch)
		listFoldersInput.SetBucket(AWS_BUCKET_NAME_VPCFLOW_LOGS)
		break
	case CloudTrailLog:
		listFoldersInput.SetPrefix("AWSLogs/" + awsAccountId + "/CloudTrail/eu-central-1/" + dateToMatch)
		listFoldersInput.SetBucket(AWS_BUCKET_NAME_CLOUDTRAIL_LOGS)
		break
	case KinesisLog:
		listFoldersInput.SetPrefix("logs/" + dateToMatch)
		listFoldersInput.SetBucket(AWS_BUCKET_NAME_KINESIS_LOGS)
		break
	default:
		panic("Not valid logType GetS3FileNames!")
	}

	if token != "" {
		listFoldersInput.SetContinuationToken(token)
	}

	output, err := S3Client.ListObjectsV2(listFoldersInput)
	if err != nil {
		fmt.Println("Error ListObjects!")
		panic(err)
	}
	//fmt.Println("total result: ", len(output.Contents))
	count := len(s3Keys)
	for _, val := range output.Contents {
		//fmt.Println("KEY:", *val.Key)
		s3Keys = append(s3Keys, *val.Key)
		//fmt.Println("YES")
		count++
	}

	moreToken := ""
	if output.NextContinuationToken != nil {
		moreToken = *output.NextContinuationToken
	}
	return s3Keys, moreToken
}

// ProcessFileToElasticsearch ...
func ProcessFileToElasticsearch(bucket string, key string, logType string) int {
	// Download the file
	localFile := downloadFile(bucket, key)
	if !localFile {
		panic("ERROR Downloading file!")
	}
	// Parse file into json
	var totalLines int
	var filesToBeDeleted string
	switch logType {
	case VpcFlowLog:
		//fmt.Println("Parsing VpcFlowLog file")
		// Perform unzip
		//fmt.Println("Perform unzip")
		runBashCommand("gunzip tmpfile.log.gz")
		totalLines = ParseVPCLogToJSON(key)
		filesToBeDeleted = "./tmpfile.json ./tmpfile.log"

		// Push data to Elasticsearch
		pushFileToElastic(
			"curl -H 'Content-Type: application/x-ndjson' "+ELASTIC_SEARCH_URL+":9200/vpclogs/doc/_bulk?pretty --data-binary @tmpfile.json -XPOST",
			"rm "+filesToBeDeleted)
		break
	case CloudTrailLog:
		//fmt.Println("Parsing CloudTrailLog file")
		totalLines = ParseCloudTrailLogToJSON(key, "tmpfile.log.gz")
		filesToBeDeleted = "./tmpfile.json tmpfile.log.gz"
		// Push data to Elasticsearch
		pushFileToElastic(
			"curl -H 'Content-Type: application/x-ndjson' "+ELASTIC_SEARCH_URL+":9200/cloudtraillogs/doc/_bulk?pretty --data-binary @tmpfile.json -XPOST",
			"rm "+filesToBeDeleted)
		break
	case KinesisLog:
		//fmt.Println("Parsing KinesisLog file")
		runBashCommand("mv tmpfile.log.gz tmpfile.log")
		totalLines = ParseKinesisLogToJSON(key)
		filesToBeDeleted = "tmpfile.json tmpfile.log"

		// Push data to Elasticsearch
		pushFileToElastic(
			"curl -H 'Content-Type: application/x-ndjson' "+ELASTIC_SEARCH_URL+":9200/kinesislogs/doc/_bulk?pretty --data-binary @tmpfile.json -XPOST",
			"rm "+filesToBeDeleted)
		break
	default:
		panic("Not valid logType ProcessFileToElasticsearch!")
	}
	return totalLines
}

// downloadFile ...
func downloadFile(bucket string, key string) bool {
	//fmt.Println("Downloading file: ")
	//S3Client = s3.New(accountsess, aws.NewConfig().WithRegion("eu-central-1"))
	getObjReq := &s3.GetObjectInput{}
	getObjReq.SetBucket(bucket)
	getObjReq.SetKey(key)
	output, err := S3Client.GetObject(getObjReq)
	if err != nil {
		fmt.Println("Error ListObjects!")
		panic(err)
	}

	// Save bytes to file
	buf := new(bytes.Buffer)
	buf.ReadFrom(output.Body)
	err = ioutil.WriteFile("./tmpfile.log.gz", buf.Bytes(), 0644)
	if err != nil {
		return false
	}
	return true
}

// pushFileToElastic ...
func pushFileToElastic(cmd1, cmd2 string) {
	//fmt.Println("pushFileToElastic file: ")
	errorFound := false
	output := runBashCommand(cmd1)
	if strings.HasPrefix(output, "ERROR:") {
		errorFound = true
	} else {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "errors") {
				if strings.Contains(line, "true") {
					fmt.Println("errorFound = true")
					fmt.Println(line)
					errorFound = true
				}
				break
			}
		}
	}
	//fmt.Println("File imported into ElssticSearch index", output)
	if errorFound {
		fmt.Println("There was an error at pushFileToElastic")
		fmt.Println("COMMAND:", cmd1)
		fmt.Println("ERROR", output)
		panic("Error at pushFileToElastic")
	}
	runBashCommand(cmd2)
}

// GetCurrentId ...
func GetCurrentId(logType string) int64 {
	var indexName string
	switch logType {
	case VpcFlowLog:
		indexName = "vpclogs"
		break
	case CloudTrailLog:
		indexName = "cloudtraillogs"
		break
	case KinesisLog:
		indexName = "kinesislogs"
		break
	default:
		panic("Not valid logType GetCurrentId!")
	}
	cmd := "curl " + ELASTIC_SEARCH_URL + ":9200/" + indexName + "/_search?pretty -H 'Content-Type: application/json' -d '{\"stored_fields\": [\"id\"],\"query\": {\"match_all\": {}},\"sort\": {\"_score\": \"desc\"},\"size\": 1}' -XGET"
	output := runBashCommand(cmd)
	lines := strings.Split(output, "\n")
	isFirstIgnored := false
	for _, line := range lines {
		if strings.Contains(line, "\"total\"") {
			if !isFirstIgnored {
				isFirstIgnored = true
				continue
			}
			//fmt.Println(line)
			stringValue := strings.TrimSpace(strings.Split(line, ":")[1])
			stringValue = stringValue[:len(stringValue)-1]
			//fmt.Println("*" + stringValue + "*")
			intValue, err := strconv.ParseInt(stringValue, 10, 64)
			if err != nil {
				panic(err)
			}
			return intValue
		}
	}
	return 0
}

// runBashCommand ...
func runBashCommand(command string) string {
	b, err := exec.Command("/bin/bash", "-c", command).Output()
	if err != nil {
		fmt.Println("")
		fmt.Println("Error Running command", command)
		panic(err)
	}
	//fmt.Println("Running command", command)
	return string(b)
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
