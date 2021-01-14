package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cheggaaa/pb"
)

var ELASTIC_SEARCH_URL = ""

// Build for AWS: GOOS=linux go build -o aws-es *.go
// Build for local: go build -o aws-es *.go

// TODO: Create And/Or Delete indexes with curl commmands
// TODO: https://github.com/go-validator/validator
// KibanaDataBulkRequest ...
type KibanaDataBulkRequest struct {
	LogType      string `json:"logType"`
	LogDate      string `json:"logDate"`
	AWSAccountId string `json:"accountId"`
	BucketName   string `json:"bucketName"`
}

func tmain() {
	fmt.Println("START")
	fmt.Println("077146224815: ", RemoveLeftZeros("077146224815"))
	fmt.Println("540493980477: ", RemoveLeftZeros("540493980477"))
	fmt.Println("007385363882: ", RemoveLeftZeros("007385363882"))
	fmt.Println("END")
}

var executionArguments []string

func main2() {
	//line := "/privx/ec2:i-0b12dd1c0f8d60f04	1610539283132	Jan 13 12:01:22 ip-10-129-33-74 sshd[8533]: Did not receive identification string from 10.129.33.25 port 32010"
	key := "s3filename"
	totalLines := 0

	fileToWrite, err := os.OpenFile("./tmpfile.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer fileToWrite.Close()

	fileToRead, err := os.Open("./tmpfile.log")
	if err != nil {
		panic("Error ParseKinesisLogToJSON!")
	}
	defer fileToRead.Close()
	scanner := bufio.NewScanner(fileToRead)
	scanner.Split(bufio.ScanLines)

	isHeaderSkipped := true
	re := regexp.MustCompile("[^\t]+")

	for scanner.Scan() {
		line := scanner.Text() + "\n"
		if !isHeaderSkipped {
			isHeaderSkipped = true
			continue
		}

		// processing lines some are duplicated...
		actualEntry := strings.Split(line, "/privx/ec2:")

		for _, entryVal := range actualEntry {
			if entryVal == "" {
				continue
			}

			// Writing index line
			indexLine := "{\"index\":{\"_index\":\"kinesislogs\"}}" + "\n"
			if _, err = fileToWrite.WriteString(indexLine); err != nil {
				fmt.Println("Line read, indexLine:", indexLine)
				panic(err)
			}

			dataLine := "{"

			// EXTRA FIELDS FOR S3 TRACEABILITY
			dataLine = dataLine + "\"s3filename\":\"" + key + "\","

			// Setting internal ID
			dataLine = dataLine + "\"id\":" + strconv.FormatInt(ItemId, 10) + ","
			ItemId++
			fmt.Println("*" + entryVal + "*")

			linePart := re.FindAll([]byte(entryVal), -1)
			linePart1Str := string(linePart[0])
			fmt.Println(linePart1Str)
			linePart2Str := string(linePart[1])
			fmt.Println(linePart2Str)
			linePart3Str := string(linePart[2])
			fmt.Println(linePart3Str)

			if linePart1Str != "-" {
				dataLine = dataLine + "\"instanceId\":" + "\"" + linePart1Str + "\"" + ","
			}
			if linePart2Str != "-" {
				dataLine = dataLine + "\"date-miliseconds\":" + linePart2Str + ","
			}
			if linePart3Str != "-" {
				// Removing new line if it is a if it is last character
				fmt.Println("linePart3Str...")
				fmt.Println("*" + linePart3Str + "*")
				if linePart3Str[len(linePart3Str)-1] == 10 {
					dataLine = dataLine + "\"log-entry-description\":" + "\"" + linePart3Str[0:len(linePart3Str)-1] + "\"" + ","
					fmt.Println("*" + linePart3Str[0:len(linePart3Str)-1] + "*")
				} else {
					dataLine = dataLine + "\"log-entry-description\":" + "\"" + linePart3Str + "\"" + ","
				}
			}

			// Removing comma if it is last character
			if dataLine[len(dataLine)-1] == 44 {
				dataLine = dataLine[0 : len(dataLine)-1]
			}
			// if dataLine[len(dataLine)-2] == 10 {
			// 	dataLine = dataLine[0:len(dataLine)-2] + dataLine[len(dataLine)-1:len(dataLine)]
			// }
			//fmt.Println("Printing last character...")
			//fmt.Println(dataLine[len(dataLine)-2])

			dataLine = dataLine + "}" + "\n"
			fmt.Println(dataLine) // fmt.Println(dataLine[len(dataLine)-10 : len(dataLine)-1])

			// Writing data line
			if _, err = fileToWrite.WriteString(dataLine); err != nil {
				panic(err)
			}
			totalLines++

		}

	}

	fmt.Println("File parsed!")
	fmt.Println(totalLines)

	// re := regexp.MustCompile("[^\t]+")
	// result := re.FindAll([]byte(line), -1)
	// //fmt.Printf("%q\n", result[0])
	// str := "TODO"
	// str = string(result[0])

	// fmt.Printf("%q\n", str)
	// //fmt.Printf("%q\n", result)

}

func main() {
	// Run as Lambda
	//lambda.Start(HandleRequest)

	// Run locally
	test := os.Getenv("ELASTICSEARCH")
	fmt.Println("ELASTICSEARCH...", test)

	// Debug on
	// go run *.go 2021/01/12 KONE-SSEMEA LOG_KINESIS_SSH_LOGS $ELASTICSEARCH
	executionArguments = os.Args
	// executionArguments = make([]string, 5)
	// executionArguments[0] = "command"
	// executionArguments[1] = "2021/01/12"
	// executionArguments[2] = "KONE-SSEMEA"
	// executionArguments[3] = "LOG_KINESIS_SSH_LOGS"
	// executionArguments[4] = "ecs-e-Appli-SQ2KHQAKSV7T-60621908.eu-central-1.elb.amazonaws.com"
	fmt.Println("executionArguments", executionArguments)
	//os.Exit(3)
	// Debug off

	if len(executionArguments) != 6 {
		PrintHelp("")
	}
	var testReq KibanaDataBulkRequest
	do(executionArguments[5], RUN_LOCAL, testReq)
	os.Exit(0)
}

// HandleRequest ... Run from Lambda
func HandleRequest(ctx context.Context, req KibanaDataBulkRequest) (string, error) {
	//do("kone", RUN_LAMBDA, req)
	return "OK", nil
}

func PrintHelp(msg string) {
	if msg != "" {
		fmt.Println("Error", msg)
	}
	fmt.Println("Sintaxis: date accountName logtype")
	fmt.Println("YYYY/MM/DD AWS-Account-Alias {LOG_CLOUDTRAIL_EVENTS | LOG_VPC_FLOW_LOGS | LOG_KINESIS_SSH_LOGS} profile")
	fmt.Println("For example...")
	fmt.Println("2020/04/17 SAP-SELLIT-PROD LOG_CLOUDTRAIL_EVENTS Kibana_url kone")
	fmt.Println("")
	fmt.Println("Valid values:")
	fmt.Println("Date format -> YYYY/MM/DD")
	fmt.Println("Account name -> Any valid account alias name from account list from settings.properties file. (All keys after AWS_ACCOUNT_TOTAL key)")
	fmt.Println("log type -> LOG_CLOUDTRAIL_EVENTS | LOG_VPC_FLOW_LOGS | LOG_KINESIS_SSH_LOGS")
	os.Exit(2)
}

func do(settingsProfile string, isRunningLocally bool, req KibanaDataBulkRequest) {
	fmt.Println("START")

	AWS_CUSTOMER_PROFILE_NAME = settingsProfile
	InitializeSettings()
	fmt.Println("PROFILE: ")
	fmt.Println(AWS_CUSTOMER_PROFILE_NAME)

	if isRunningLocally {
		req = KibanaDataBulkRequest{}
		req.LogDate = executionArguments[1]
		req.AWSAccountId = SettingsMap[executionArguments[2]]
		req.LogType = executionArguments[3]

		// Kibana endpoint
		ELASTIC_SEARCH_URL = executionArguments[4]
		fmt.Println("ELASTIC_SEARCH_URL: ", ELASTIC_SEARCH_URL)

		if req.LogType == VpcFlowLog {
			req.BucketName = AWS_BUCKET_NAME_VPCFLOW_LOGS
		} else if req.LogType == CloudTrailLog {
			req.BucketName = AWS_BUCKET_NAME_CLOUDTRAIL_LOGS
		} else if req.LogType == KinesisLog {
			req.BucketName = AWS_BUCKET_NAME_KINESIS_LOGS
		} else {
			PrintHelp("Not valid LogType!")
		}

		fmt.Println("KibanaDataBulkRequest:", req)
		runningUsingIAMUserProfile(req)
	} else {
		runningUsingIAMInstanceProfile()
	}
	fmt.Println("DONE")
}

// runningUsingIAMUserProfile ...
func runningUsingIAMUserProfile(req KibanaDataBulkRequest) {

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter MFA... ")
	mfa, _ := reader.ReadString('\n') //'\n'
	//mfa := "507402 "
	fmt.Print(mfa)
	if len(mfa) != 7 {
		os.Exit(2)
	}
	mfa = mfa[0:6]

	// Main NC credentials
	sessionCredens := ReqSessionCredens(mfa)
	if sessionCredens == nil {
		fmt.Println("Credentials error")
		os.Exit(3)
	}
	fmt.Println("Credentials ok")
	fmt.Println("")

	// Assume role
	// var roleSession *session.Session
	// if AWS_CUSTOMER_PROFILE_NAME == "upm" {
	// 	roleSession = GetAssumedRoleSession(sessionCredens, "803523913631", "eu-central-1")
	// } else if AWS_CUSTOMER_PROFILE_NAME == "kone" {
	// 	roleSession = GetAssumedRoleSession(sessionCredens, "531322851491", "eu-central-1")
	// }
	if AWS_CUSTOMER_LANDING_ACCOUNT_ID == "" || AWS_CUSTOMER_LANDING_REGION_CODE == "" {
		fmt.Println("Landing settings error")
		os.Exit(4)
	}
	roleSession := GetAssumedRoleSession(sessionCredens, AWS_CUSTOMER_LANDING_ACCOUNT_ID, AWS_CUSTOMER_LANDING_REGION_CODE)

	// 1. Prepare the request
	ItemId = GetCurrentId(req.LogType) + 1
	//fmt.Println("At the moment index num:", ItemId)
	itemIdS := make([]int64, 0)
	itemIdS = append(itemIdS, ItemId)
	rowsBefore := itemIdS[0]

	fmt.Println("testReq:", req)
	// S3 Request to list objects (i.e. Logs) in bucket
	allKeys := make([]string, 0)

	// 2. Loop all the accounts to get only filenames needed for given time
	fmt.Println("Fetching filenames...")
	S3Client = s3.New(roleSession, aws.NewConfig().WithRegion("eu-central-1"))
	accountS3Keys := make([]string, 0)
	hasmore := ""
	resetDate := req.LogDate
	if req.AWSAccountId != ALL_AWS_ACCOUNTS {
		for accountName, accountId := range SettingsMap {
			if req.AWSAccountId == accountId {
				fmt.Println("AWS Account: ", accountName)
				if req.LogType == KinesisLog {
					size := 23
					for h := range N(size) {
						hour := ""
						if h < 10 {
							hour = "0" + strconv.Itoa(h)
						} else {
							hour = strconv.Itoa(h)
						}
						req.LogDate = resetDate + "/" + hour + "/"
						for true {
							accountS3Keys, hasmore = GetS3FileNames(req.LogDate, hasmore, accountId, accountS3Keys, req.LogType)
							if hasmore == "" {
								break
							}
						}
						for _, s3Key := range accountS3Keys {
							allKeys = append(allKeys, s3Key)
						}
					}
				} else {
					for true {
						accountS3Keys, hasmore = GetS3FileNames(req.LogDate, hasmore, accountId, accountS3Keys, req.LogType)
						if hasmore == "" {
							break
						}
					}
					for _, s3Key := range accountS3Keys {
						allKeys = append(allKeys, s3Key)
					}
				}
				break
			}
		}
	} else {
		// Loop all TODO DEBUG
		accountS3Keys := make([]string, 0)
		for accountName, accountId := range SettingsMap {
			hasmore := ""
			fmt.Println("AWS Account: ", accountName)
			if req.LogType == KinesisLog {
				os.Exit(10)
			}
			for true {
				fmt.Println("Total accountS3Keys: ", len(accountS3Keys))
				accountS3Keys, hasmore = GetS3FileNames(req.LogDate, hasmore, accountId, accountS3Keys, req.LogType)
				if hasmore == "" {
					break
				}
			}
		}
		for _, s3Key := range accountS3Keys {
			allKeys = append(allKeys, s3Key)
		}
	}

	totalFiles := len(allKeys)
	fmt.Println("total files to be processed... ", totalFiles)

	// Prepare progress bar
	progressBar := pb.StartNew(totalFiles)
	//progressBar.Format

	// 3. Process log files
	totalRowsImported := 0
	for _, key := range allKeys {
		//fmt.Println("File processing... ", key)
		linesImported := ProcessFileToElasticsearch(req.BucketName, key, req.LogType)
		totalRowsImported = totalRowsImported + linesImported
		progressBar.Increment()
	}
	progressBar.Finish()
	fmt.Println("")

	// 4. Verify the imported items
	rowsAfter := ItemId
	//fmt.Println("At the moment index num:", rowsAfter)
	fmt.Println("Imported items: ", totalRowsImported)
	fmt.Println("Expected items: ", rowsAfter-rowsBefore)
	fmt.Println("")

}

// N is an alias for an unallocated struct
func N(size int) []struct{} {
	return make([]struct{}, size)
}

// runningUsingIAMInstanceProfile ...
func runningUsingIAMInstanceProfile() {
	fmt.Println("")
	fmt.Println("Running runningUsingIAMInstanceProfile")
	fmt.Println("")

	// Main NC credentials
	roleSession := GetInstanceProfileSession()
	if roleSession == nil {
		fmt.Println("Credentials error")
		os.Exit(3)
	}
	fmt.Println("Credentials ok")
}

func getToday() string {
	return time.Now().Format("2006/01/02")
}

func getYesterday() string {
	return time.Now().AddDate(0, 0, -1).Format("2006/01/02")
}
