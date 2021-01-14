package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

var SettingsMap map[string]string
var TagsMap map[string]string = make(map[string]string)

var AWS_CUSTOMER_LANDING_ACCOUNT_ID string
var AWS_CUSTOMER_LANDING_REGION_CODE string

/*
* Create the accountmap with all AWS account IDs
 */
func InitializeSettings() {
	LoadEnvVariables()
	//var awsAccounts map[string]string
	SettingsMap = make(map[string]string)

	// Scan all properties
	lines, err := ScanLines("./settings.properties")
	if err != nil {
		panic(err)
	}

	// Assign entries to map
	totalTagsToMatch := 0
	isFound := false
	for _, line := range lines {
		lineWords := strings.Split(line, "=")
		if lineWords[0] == "["+AWS_CUSTOMER_PROFILE_NAME+"]" {
			isFound = true
			continue
		}
		if isFound {
			if lineWords[0] == "AWS_LOCAL_MFA_USER_PROFILE" {
				AWS_LOCAL_MFA_USER_PROFILE = lineWords[1]
				continue
			}
			if lineWords[0] == "AWS_CUSTOMER_LANDING_ACCOUNT_ID" {
				AWS_CUSTOMER_LANDING_ACCOUNT_ID = lineWords[1]
				continue
			}
			if lineWords[0] == "AWS_CUSTOMER_LANDING_REGION_CODE" {
				AWS_CUSTOMER_LANDING_REGION_CODE = lineWords[1]
				continue
			}
			if lineWords[0] == "AWS_IAM_ROLE_NAME_TO_BE_ASSUMED" {
				AWS_IAM_ROLE_NAME_TO_BE_ASSUMED = lineWords[1]
				continue
			}
			if lineWords[0] == "AWS_LOCAL_IAM_USER_MFA" {
				AWS_LOCAL_IAM_USER_MFA = lineWords[1]
				continue
			}
			if lineWords[0] == "AWS_BUCKET_NAME_VPCFLOW_LOGS" {
				AWS_BUCKET_NAME_VPCFLOW_LOGS = lineWords[1]
				continue
			}
			if lineWords[0] == "AWS_BUCKET_NAME_CLOUDTRAIL_LOGS" {
				AWS_BUCKET_NAME_CLOUDTRAIL_LOGS = lineWords[1]
				continue
			}
			if lineWords[0] == "AWS_BUCKET_NAME_KINESIS_LOGS" {
				AWS_BUCKET_NAME_KINESIS_LOGS = lineWords[1]
				continue
			}
			if lineWords[0] == "TOTAL_TAGS" {
				totalTagsToMatch, err = strconv.Atoi(lineWords[1])
				if err != nil {
					panic(err)
				}
				continue
			}
			// Fetching tag key names
			if totalTagsToMatch > 0 {
				TagsMap[lineWords[0]] = lineWords[1]
				totalTagsToMatch--
				continue
			}

			// Fetching account names
			if lineWords[0] == "AWS_ACCOUNT_TOTAL" {
				continue
			}
			if strings.HasPrefix(lineWords[0], "[") {
				break
			}
			SettingsMap[lineWords[0]] = lineWords[1]
		}
	}
	//SettingsMap = awsAccounts
}

// ReqSessionCredens ...
// Get temporary session credentials
func ReqSessionCredens(mfaToken string) *sts.Credentials {
	fmt.Println("ReqSessionCredens... token:" + mfaToken)
	fmt.Println("ReqSessionCredens... AWS_LOCAL_MFA_USER_PROFILE:" + AWS_LOCAL_MFA_USER_PROFILE)

	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: AWS_LOCAL_MFA_USER_PROFILE,
	})
	svc := sts.New(sess)
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(3600),
		SerialNumber:    aws.String(AWS_LOCAL_IAM_USER_MFA),
		TokenCode:       aws.String(mfaToken),
	}
	result, err := svc.GetSessionToken(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case sts.ErrCodeRegionDisabledException:
				fmt.Println(sts.ErrCodeRegionDisabledException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}
	return result.Credentials
}

// GetAssumedRoleSession ...
// Get assumed role credentials. If (instanceCredens == nil), then this is called by InstanceProfile, otherwise User CLI
func GetAssumedRoleSession(sessionCredens *sts.Credentials, awsAccountID string, region string) *session.Session {
	var rolesess *session.Session
	var roleToAssume *string

	AccessKeyID := *sessionCredens.AccessKeyId
	SecretAccessKey := *sessionCredens.SecretAccessKey
	SessionToken := *sessionCredens.SessionToken
	rolesess, _ = session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(AccessKeyID, SecretAccessKey, SessionToken),
	})
	roleToAssume = aws.String("arn:aws:iam::" + awsAccountID + ":role/" + AWS_IAM_ROLE_NAME_TO_BE_ASSUMED)

	// Assuming role
	svc := sts.New(rolesess)
	assumeinput := &sts.AssumeRoleInput{
		ExternalId:      aws.String("123ABC"),
		RoleArn:         roleToAssume,
		RoleSessionName: aws.String("NC-local-SDK-session"),
	}

	roleResult, err := svc.AssumeRole(assumeinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case sts.ErrCodeMalformedPolicyDocumentException:
				fmt.Println(sts.ErrCodeMalformedPolicyDocumentException, aerr.Error())
			case sts.ErrCodePackedPolicyTooLargeException:
				fmt.Println(sts.ErrCodePackedPolicyTooLargeException, aerr.Error())
			case sts.ErrCodeRegionDisabledException:
				fmt.Println(sts.ErrCodeRegionDisabledException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}

	// Get account credentials
	AccountAccessKeyID := *roleResult.Credentials.AccessKeyId
	AccountSecretAccessKey := *roleResult.Credentials.SecretAccessKey
	AccountSessionToken := *roleResult.Credentials.SessionToken
	accountsess, err := session.NewSession(&aws.Config{
		// Region:      aws.String(AWS_SESSION_REGION),
		Credentials: credentials.NewStaticCredentials(AccountAccessKeyID, AccountSecretAccessKey, AccountSessionToken),
	})
	if err != nil {
		return nil
	}
	return accountsess
}

// GetInstanceProfileSession ...
func GetInstanceProfileSession() *session.Session {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(AWS_LOCAL_IAM_USER_REGION)},
	)
	if err != nil {
		log.Println("ERROR", err)
		panic(err)
	}
	return sess
}

// GetInstanceAssumeRoleSession ...
func GetInstanceAssumeRoleSession(noRoleSession *session.Session, awsAccountID string) *session.Session {
	// Assuming role
	svc := sts.New(noRoleSession)
	assumeinput := &sts.AssumeRoleInput{
		ExternalId:      aws.String("ec2-report-sdk-role-Instance-assumed"),
		RoleArn:         aws.String("arn:aws:iam::" + awsAccountID + ":role/ec2-report-sdk-role"),
		RoleSessionName: aws.String("ec2-report-sdk-role-instance-session"),
	}
	roleResult, err := svc.AssumeRole(assumeinput)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case sts.ErrCodeMalformedPolicyDocumentException:
				fmt.Println(sts.ErrCodeMalformedPolicyDocumentException, aerr.Error())
			case sts.ErrCodePackedPolicyTooLargeException:
				fmt.Println(sts.ErrCodePackedPolicyTooLargeException, aerr.Error())
			case sts.ErrCodeRegionDisabledException:
				fmt.Println(sts.ErrCodeRegionDisabledException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil
	}

	// Create session object
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(AWS_EC2_REGION),
		Credentials: credentials.NewStaticCredentials(*roleResult.Credentials.AccessKeyId, *roleResult.Credentials.SecretAccessKey, *roleResult.Credentials.SessionToken),
	})

	if err != nil {
		return nil
	}
	return sess
}

func ScanLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, nil
}
