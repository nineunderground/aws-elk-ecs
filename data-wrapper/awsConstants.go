package main

import "os"

// **********************
// ENVARIMENTAL VARIABLES
// **********************
// var ELASTIC_SEARCH_URL = os.Getenv("ELASTICSEARCH")

var AWS_EC2_REPORT_SERVICE_ACCOUNT = os.Getenv("AWS_EC2_REPORT_SERVICE_ACCOUNT")

// Bucket config
var AWS_S3_BUCKET_REGION = os.Getenv("AWS_S3_BUCKET_REGION")
var AWS_S3_BUCKET_NAME = os.Getenv("AWS_S3_BUCKET_NAME")
var AWS_S3_BUCKET_FOLDER = os.Getenv("AWS_S3_BUCKET_FOLDER")
var AWS_S3_BUCKET_FOLDER_SUBFOLDER = os.Getenv("AWS_S3_BUCKET_FOLDER_SUBFOLDER")

// Topic config
var AWS_SNS_TOPIC_REGION = os.Getenv("AWS_SNS_TOPIC_REGION")
var AWS_SNS_TOPIC_ARN = os.Getenv("AWS_SNS_TOPIC_ARN")

// Excel config
var AWS_EC2_REGION = os.Getenv("AWS_EC2_REGION")

// IAM Role to perform the SDK actions
var AWS_IAM_ROLE_NAME_TO_BE_ASSUMED = os.Getenv("AWS_IAM_ROLE_NAME_TO_BE_ASSUMED") //"Nordcloud-Automation"

// **********************
// LOCAL VARIABLES
// **********************
var AWS_LOCAL_IAM_USER_REGION = string("eu-central-1")
var AWS_LOCAL_MFA_USER_PROFILE = "" // Loaded from properties
var AWS_LOCAL_IAM_USER_MFA = ""     // Loaded from properties

// **********************
// OTHER VARIABLES
// **********************
var AWS_BUCKET_NAME_VPCFLOW_LOGS = ""    // Loaded from properties
var AWS_BUCKET_NAME_CLOUDTRAIL_LOGS = "" // Loaded from properties
var AWS_BUCKET_NAME_KINESIS_LOGS = ""    // Loaded from properties

var ALL_AWS_ACCOUNTS = "ALL"

// Other config
// const EXCEL_FILENAME = string("AWS_EC2_Overview_Report.xlsx")
// const EXCEL_SHEET = string("VM Inventory")
const RUN_LOCAL = true
const RUN_LAMBDA = false

const VpcFlowLog = string("LOG_VPC_FLOW_LOGS")
const CloudTrailLog = string("LOG_CLOUDTRAIL_EVENTS")
const KinesisLog = string("LOG_KINESIS_SSH_LOGS")

var AWS_CUSTOMER_PROFILE_NAME string

func LoadEnvVariables() {
	if AWS_EC2_REPORT_SERVICE_ACCOUNT == "" {
		AWS_EC2_REPORT_SERVICE_ACCOUNT = string("803523913631")
	}
	if AWS_S3_BUCKET_REGION == "" {
		AWS_S3_BUCKET_REGION = string("eu-central-1")
	}
	if AWS_S3_BUCKET_NAME == "" {
		AWS_S3_BUCKET_NAME = string("dx-monitoring-ssemea")
	}
	if AWS_S3_BUCKET_FOLDER == "" {
		AWS_S3_BUCKET_FOLDER = string("ec2-report")
	}
	if AWS_S3_BUCKET_FOLDER_SUBFOLDER == "" {
		AWS_S3_BUCKET_FOLDER_SUBFOLDER = string("archive")
	}
	if AWS_SNS_TOPIC_REGION == "" {
		AWS_SNS_TOPIC_REGION = string("eu-central-1")
	}
	if AWS_SNS_TOPIC_ARN == "" {
		AWS_SNS_TOPIC_ARN = string("arn:aws:sns:" + AWS_SNS_TOPIC_REGION + ":" + AWS_EC2_REPORT_SERVICE_ACCOUNT + ":ec2-report-static-ssemea-criticalalarm-topic")
	}
	if AWS_EC2_REGION == "" {
		AWS_EC2_REGION = string("eu-central-1")
	}
}
