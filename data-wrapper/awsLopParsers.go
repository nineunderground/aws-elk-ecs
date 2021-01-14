package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var apiCallbacks int = 0

//********************************
//* CloudTrailLog struct parsing *
//********************************
type Records struct {
	Records []Record `json:"Records"`
}

type Record struct {
	EventVersion    string       `json: "eventVersion"`
	UserIdentity    UserIdentity `json: "userIdentity"`
	EventTime       string       `json: "eventTime"`
	EventSource     string       `json: "eventSource"`
	EventName       string       `json: "eventName"`
	AwsRegion       string       `json: "awsRegion"`
	SourceIPAddress string       `json: "sourceIPAddress"`
	UserAgent       string       `json: "userAgent"`
	ReqParams       string       `json: "requestParameters"`
	// requestParameters is wrapped manually
	// responseElements is wrapped manually
	RequestID          string     `json: "requestID"`
	EventID            string     `json: "eventID"`
	Resources          []Resource `json: "resources"`
	EventType          string     `json: "eventType"`
	RecipientAccountId string     `json: "recipientAccountId"`
	SharedEventID      string     `json: "sharedEventID"`
}

type UserIdentity struct {
	Type           string         `json: "type"`
	InvokedBy      string         `json: "invokedBy"`
	PrincipalId    string         `json: "principalId"`
	Arn            string         `json: "arn"`
	SessionContext SessionContext `json: "sessionContext"`
}

type SessionContext struct {
	Attributes Attributes `json: "attributes"`
}

type Attributes struct {
	mfaAuthenticated bool   `json: "mfaAuthenticated"`
	creationDate     string `json: "creationDate"`
}

type RequestParameter struct {
	roleArn         string `json: "roleArn"`
	roleSessionName string `json: "roleSessionName"`
	externalId      string `json: "externalId"`
	durationSeconds int    `json: "durationSeconds"`
}

type Resource struct {
	ARN       string `json: "ARN"`
	AccountId string `json: "accountId"`
	Type      string `json: "type"`
}

type AWSIPAddress struct {
	City          string  `json:"city"`
	ContinentCode string  `json:"continent_code"`
	ContinentName string  `json:"continent_name"`
	CountryCode   string  `json:"country_code"`
	CountryName   string  `json:"country_name"`
	IP            string  `json:"ip"`
	Latitude      float64 `json:"latitude"`
	Location      struct {
		CallingCode             string `json:"calling_code"`
		Capital                 string `json:"capital"`
		CountryFlag             string `json:"country_flag"`
		CountryFlagEmoji        string `json:"country_flag_emoji"`
		CountryFlagEmojiUnicode string `json:"country_flag_emoji_unicode"`
		GeonameID               int64  `json:"geoname_id"`
		IsEu                    bool   `json:"is_eu"`
		Languages               []struct {
			Code   string `json:"code"`
			Name   string `json:"name"`
			Native string `json:"native"`
		} `json:"languages"`
	} `json:"location"`
	Longitude  float64 `json:"longitude"`
	RegionCode string  `json:"region_code"`
	RegionName string  `json:"region_name"`
	Type       string  `json:"type"`
	Zip        string  `json:"zip"`
}

// ParseCloudTrailLogToJSON ...
func ParseCloudTrailLogToJSON(key string, sourceFile string) int {

	// PreProcess the file content to be streamline as json entries
	totalJSONEntries := GetJSONAsString(sourceFile)

	// Prepare the output file
	destFile := "./tmpfile.json"
	fileToWrite, err := os.OpenFile(destFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	defer fileToWrite.Close()
	if err != nil {
		panic(err)
	}

	// Loop all JSON entries
	totalItems := 0
	for _, jsonAsString := range totalJSONEntries {
		//fmt.Println("jsonAsString:")
		//fmt.Println(jsonAsString)

		// Parse JSON entry
		var item Record
		error := json.Unmarshal([]byte(jsonAsString), &item)
		if error != nil {
			panic(error)
		}

		// Writing output index line
		indexLine := "{\"index\":{\"_index\":\"cloudtraillogs\"}}" + "\n"
		if _, err = fileToWrite.WriteString(indexLine); err != nil {
			panic(err)
		}
		ItemId++

		// Writing output data line
		dataLine := "{"

		// EXTRA FIELDS FOR S3 TRACEABILITY
		dataLine = dataLine + "\"s3filename\":\"" + key + "\","
		keyFolders := strings.Split(key, "/")
		s3folderpath := keyFolders[4] + "-" + keyFolders[5] + "-" + keyFolders[6]
		dataLine = dataLine + "\"s3folderpath\":\"" + s3folderpath + "\","
		// Fetch account id from filename, otherwise inline value must be 0 or missing
		accountID := keyFolders[1]
		accountName := getAWSAccountName(accountID)
		dataLine = dataLine + "\"account-id\":" + RemoveLeftZeros(accountID) + ","
		dataLine = dataLine + "\"account-name\":\"" + accountName + "\","

		// CLOUDTRAIL FIELDS:
		dataLine = dataLine + "\"eventVersion\":\"" + item.EventVersion + "\","
		dataLine = dataLine + "\"userIdentity.type\":\"" + item.UserIdentity.Type + "\","
		dataLine = dataLine + "\"userIdentity.invokedBy\":\"" + item.UserIdentity.InvokedBy + "\","

		principalID := item.UserIdentity.PrincipalId
		if principalID != "" {
			if len(strings.Split(principalID, ":")) > 1 {
				principalID = strings.Split(principalID, ":")[1]
			}
		}
		dataLine = dataLine + "\"userIdentity.principalId\":\"" + principalID + "\","

		dataLine = dataLine + "\"userIdentity.arn\":\"" + item.UserIdentity.Arn + "\","
		isMFA := strconv.FormatBool(item.UserIdentity.SessionContext.Attributes.mfaAuthenticated)
		dataLine = dataLine + "\"userIdentity.SessionContext.Attributes.mfaAuthenticated\":\"" + isMFA + "\","
		dataLine = dataLine + "\"eventTime\":\"" + item.EventTime + "\","
		dataLine = dataLine + "\"eventSource\":\"" + item.EventSource + "\","
		dataLine = dataLine + "\"eventName\":\"" + item.EventName + "\","
		dataLine = dataLine + "\"awsRegion\":\"" + item.AwsRegion + "\","
		dataLine = dataLine + "\"sourceIPAddress\":\"" + item.SourceIPAddress + "\","
		geolocationLine := getGeoLocation(dataLine, item.SourceIPAddress)
		if geolocationLine != "" {
			dataLine = dataLine + "\"locationIP\":\"" + geolocationLine + "\","
		}
		dataLine = dataLine + "\"userAgent\":\"" + item.UserAgent + "\","
		reqParamsValue := getInnerValueFromJsonString(jsonAsString, "requestParameters\":", "responseElements\":")
		if reqParamsValue != "null" {
			dataLine = dataLine + "\"requestParameters\":\"" + reqParamsValue + "\","
		}
		respParamsValue := getInnerValueFromJsonString(jsonAsString, "responseElements\":", "requestID\":")
		if respParamsValue != "null" {
			dataLine = dataLine + "\"responseElements\":\"" + respParamsValue + "\","
		}
		dataLine = dataLine + "\"requestID\":\"" + item.RequestID + "\","
		dataLine = dataLine + "\"eventID\":\"" + item.EventID + "\","
		resourcesLine := "\"resources\":\""
		for _, itemRes := range item.Resources {
			resourcesLine = resourcesLine + "ARN->" + itemRes.ARN + " "
		}
		if len(item.Resources) > 0 {
			dataLine = dataLine + resourcesLine + "\","
		}
		dataLine = dataLine + "\"eventType\":\"" + item.EventType + "\","
		dataLine = dataLine + "\"recipientAccountId\":\"" + item.RecipientAccountId + "\","
		if item.SharedEventID != "" {
			dataLine = dataLine + "\"sharedEventID\":\"" + item.SharedEventID + "\""
		}

		// Remove the last comma if there is any
		if dataLine[len(dataLine)-1] == 44 {
			dataLine = dataLine[0 : len(dataLine)-1]
		}
		dataLine = dataLine + "}" + "\n"
		if _, err = fileToWrite.WriteString(dataLine); err != nil {
			panic(err)
		}
		totalItems++
	}
	//totalItems = len(data.Records)
	//fmt.Println("Total Items parsed!", totalItems)

	//fmt.Println("File parsed!")
	return totalItems
}

// ParseVPCLogToJSON ...
func ParseVPCLogToJSON(key string) int {
	totalLines := 0

	fileToWrite, err := os.OpenFile("./tmpfile.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer fileToWrite.Close()

	fileToRead, err := os.Open("./tmpfile.log")
	if err != nil {
		panic("Error ParseVPCLogToJSON!")
	}
	defer fileToRead.Close()
	scanner := bufio.NewScanner(fileToRead)
	scanner.Split(bufio.ScanLines)

	isHeaderSkipped := false

	for scanner.Scan() {
		line := scanner.Text() + "\n"
		if !isHeaderSkipped {
			isHeaderSkipped = true
			continue
		}

		// Writing index line
		indexLine := "{\"index\":{\"_index\":\"vpclogs\"}}" + "\n"
		if _, err = fileToWrite.WriteString(indexLine); err != nil {
			fmt.Println("Line read, indexLine:", indexLine)
			panic(err)
		}

		fields := strings.Split(line, " ")
		dataLine := "{"

		// EXTRA FIELDS FOR S3 TRACEABILITY
		dataLine = dataLine + "\"s3filename\":\"" + key + "\","
		keyFolders := strings.Split(key, "/")
		s3folderpath := keyFolders[4] + "-" + keyFolders[5] + "-" + keyFolders[6]
		dataLine = dataLine + "\"s3folderpath\":\"" + s3folderpath + "\","

		// Setting internal ID
		dataLine = dataLine + "\"id\":" + strconv.FormatInt(ItemId, 10) + ","
		ItemId++

		if fields[0] != "-" {
			dataLine = dataLine + "\"version\":" + fields[0] + ","
		}

		// Fetch account id from filename, otherwise inline value must be 0 or missing
		accountID := keyFolders[1]
		accountName := getAWSAccountName(accountID)
		dataLine = dataLine + "\"account-id\":" + RemoveLeftZeros(accountID) + ","
		dataLine = dataLine + "\"account-name\":\"" + accountName + "\","

		if fields[2] != "-" {
			dataLine = dataLine + "\"interface-id\":\"" + fields[2] + "\","
		}
		if fields[3] != "-" {
			dataLine = dataLine + "\"srcaddr\":\"" + fields[3] + "\","
			// TODO
			// dataLine = dataLine + "\"incoming traffic from private network | public internet\":\"" + IsPublicInternetIP(fields[3]) + "\","
		}
		if fields[4] != "-" {
			dataLine = dataLine + "\"dstaddr\":\"" + fields[4] + "\","
			// TODO
			// dataLine = dataLine + "\"outgoing traffic from private network | public internet\":\"" + IsPublicInternetIP(fields[4]) + "\","
		}
		if fields[5] != "-" {
			dataLine = dataLine + "\"srcport\":" + fields[5] + ","
		}
		if fields[6] != "-" {
			dataLine = dataLine + "\"dstport\":" + fields[6] + ","
		}
		if fields[7] != "-" {
			dataLine = dataLine + "\"protocol\":" + fields[7] + ","
			//dataLine = dataLine + "\"protocol.name\":" + getProtocolName(fields[7]) + "," // TODO
		}
		if fields[8] != "-" {
			dataLine = dataLine + "\"packets\":" + fields[8] + ","
		}
		if fields[9] != "-" {
			dataLine = dataLine + "\"bytes\":" + fields[9] + ","
		}
		if fields[10] != "-" {
			dataLine = dataLine + "\"start\":" + fields[10] + "000,"
		}
		if fields[11] != "-" {
			dataLine = dataLine + "\"end\":" + fields[11] + "000,"
		}
		durationStart, _ := strconv.Atoi(fields[10])
		durationEnd, _ := strconv.Atoi(fields[11])
		if durationStart != 0 && durationEnd != 0 {
			duration := durationEnd - durationStart
			dataLine = dataLine + "\"duration\":" + strconv.Itoa(duration) + ","
		}

		if fields[12] != "-" {
			dataLine = dataLine + "\"action\":\"" + fields[12] + "\","
		}
		if fields[13] != "-" {
			dataLine = dataLine + "\"log-status\":\"" + fields[13]
		}
		val := fields[13]
		//fmt.Println("Last char:")
		if val[len(val)-1] == 10 {
			dataLine = dataLine[0 : len(dataLine)-1]
		}
		if val != "-" {
			dataLine = dataLine + "\""
		}
		//fmt.Println("LINE parsed!")
		//fmt.Println(dataLine)
		if dataLine[len(dataLine)-1] == 44 {
			dataLine = dataLine[0 : len(dataLine)-1]
		}
		dataLine = dataLine + "}" + "\n"
		//fmt.Println(dataLine)

		// Writing data line
		if _, err = fileToWrite.WriteString(dataLine); err != nil {
			panic(err)
		}
		totalLines++
	}
	//fmt.Println("File parsed!")
	return totalLines
}

// ParseKinesisLogToJSON ...
func ParseKinesisLogToJSON(key string) int {

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
				//fmt.Println("Line read, indexLine:", indexLine)
				panic(err)
			}

			dataLine := "{"

			// EXTRA FIELDS FOR S3 TRACEABILITY
			dataLine = dataLine + "\"s3filename\":\"" + key + "\","

			// Setting internal ID
			dataLine = dataLine + "\"id\":" + strconv.FormatInt(ItemId, 10) + ","
			ItemId++
			// fmt.Println("*" + entryVal + "*")

			linePart := re.FindAll([]byte(entryVal), -1)
			linePart1Str := string(linePart[0])
			// fmt.Println(linePart1Str)
			linePart2Str := string(linePart[1])
			// fmt.Println(linePart2Str)
			linePart3Str := string(linePart[2])
			// fmt.Println(linePart3Str)

			if linePart1Str != "" {
				dataLine = dataLine + "\"instanceId\":" + "\"" + linePart1Str + "\"" + ","
			}
			if linePart2Str != "" {
				dataLine = dataLine + "\"logdate\":" + linePart2Str + ","
			}
			if linePart3Str != "" {

				dataLine = dataLine + "\"logdateDesc\":" + "\"" + linePart3Str[0:15] + "\"" + ","
				linePart3StrDescArray := strings.Split(linePart3Str, " ")
				dataLine = dataLine + "\"hostname\":" + "\"" + linePart3StrDescArray[3] + "\"" + ","

				// Removing new line if it is a if it is last character
				// fmt.Println("linePart3Str...")
				// fmt.Println("*" + linePart3Str + "*")
				if linePart3Str[len(linePart3Str)-1] == 10 {
					dataLine = dataLine + "\"description\":" + "\"" + linePart3Str[0:len(linePart3Str)-1] + "\"" + ","
					// fmt.Println("*" + linePart3Str[0:len(linePart3Str)-1] + "*")
				} else {
					dataLine = dataLine + "\"description\":" + "\"" + linePart3Str + "\"" + ","
				}
				// If description contains "pam" or not...
				if strings.Contains(linePart3Str, "pam") {
					dataLine = dataLine + "\"isPamEntry\":" + strconv.FormatInt(1, 10) + ","
				} else {
					dataLine = dataLine + "\"isPamEntry\":" + strconv.FormatInt(0, 10) + ","
				}

			}

			// Removing comma if it is last character
			if dataLine[len(dataLine)-1] == 44 {
				dataLine = dataLine[0 : len(dataLine)-1]
			}

			dataLine = dataLine + "}" + "\n"
			// fmt.Println(dataLine)

			// Writing data line
			if _, err = fileToWrite.WriteString(dataLine); err != nil {
				panic(err)
			}
			totalLines++

		}

	}

	//fmt.Println("File parsed!")
	//fmt.Println(totalLines)
	return totalLines
}

// Get names from ./protocol-numbers.csv
func getProtocolName(protocolCodeId string) string {
	return "TODO"
}

func getAWSAccountName(accountId string) string {
	for name, id := range SettingsMap {
		if id == accountId {
			return name
		}
	}
	return ""
}

// https://medium.com/starting-up-security/investigating-cloudtrail-logs-c2ecdf578911
// GetJSONAsString ...
func GetJSONAsString(sourceFile string) []string {
	// Preprocess the File
	fileToRead, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		panic(err)
	}
	// Remove the first part
	fileToRead = fileToRead[12:]
	// Remove the last part
	fileToRead = fileToRead[:len(fileToRead)-2]
	// Split into JsonRaw entries, ignoring first entry
	eventEntries := strings.Split(string(fileToRead), "eventVersion")[1:]
	completeEventEntries := make([]string, len(eventEntries), len(eventEntries))
	idx := 0
	for _, val := range eventEntries {
		completeEventEntries[idx] = "{\"eventVersion" + val
		if idx == len(eventEntries)-1 {
			completeEventEntries[idx] = completeEventEntries[idx][:len(completeEventEntries[idx])]
		} else {
			completeEventEntries[idx] = completeEventEntries[idx][:len(completeEventEntries[idx])-3]
		}
		idx++
	}
	return completeEventEntries
}

func getInnerValueFromJsonString(jsonAsString string, from string, to string) string {
	sideValue := strings.Split(jsonAsString, from)
	if strings.HasPrefix(sideValue[1], "null,") {
		return "null"
	}
	innerValues := strings.Split(sideValue[1], to)
	innerValue := innerValues[0]
	finalValue := innerValue[:len(innerValue)-2]

	// Escape all '"' for '\"'
	return strings.ReplaceAll(finalValue, "\"", "\\\"")
}

func getGeoLocation(dataLine string, ip string) string {
	// Check if it is valid IP:
	ipValues := strings.Split(ip, ".")
	if len(ipValues) != 4 {
		return ""
	}
	_, err := strconv.ParseInt(ipValues[0], 10, 64)
	if err != nil {
		return ""
	}
	_, err = strconv.ParseInt(ipValues[1], 10, 64)
	if err != nil {
		return ""
	}
	_, err = strconv.ParseInt(ipValues[2], 10, 64)
	if err != nil {
		return ""
	}
	_, err = strconv.ParseInt(ipValues[3], 10, 64)
	if err != nil {
		return ""
	}

	// If IP is within Private network class, then ignore it
	if !IsPublicInternetIP(ip) {
		return ""
	}

	// Check if it is in geo map
	lat, lon := getLatLon(ip)
	if lat == 0.0 && lon == 0.0 {
		// API CALLBACK
		response, err := http.Get("http://api.ipstack.com/" + ip + "?access_key=3145a06cf519b9fdcef3a415de3c5e40&format=1")
		apiCallbacks++
		if err != nil {
			fmt.Println(err)
			return ""
		}
		defer response.Body.Close()
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		// Unmarshal the JSON byte slice to a GeoIP struct
		var item AWSIPAddress
		err = json.Unmarshal(body, &item)
		if err != nil {
			fmt.Println(err)
			return ""
		}
		lat = item.Latitude
		lon = item.Longitude
		AddLatLon(ip, lat, lon)
	}
	locationIP := strconv.FormatFloat(lat, 'f', 6, 64) + "," + strconv.FormatFloat(lon, 'f', 6, 64)
	return locationIP
}

func IsPublicInternetIP(ipBlock string) bool {
	_, cidrRange, _ := net.ParseCIDR(ipBlock + "/32")
	// 10.0.0.0/8  -> 10.0.0.0 – 10.255.255.255
	_, emeaRangeClassA, _ := net.ParseCIDR("10.0.0.0/8")
	if emeaRangeClassA.Contains(cidrRange.IP) {
		return false
	}
	// 172.16.0.0/12 -> 172.16.0.0 – 172.31.255.255
	_, emeaRangeClassB, _ := net.ParseCIDR("172.16.0.0/12")
	if emeaRangeClassB.Contains(cidrRange.IP) {
		return false
	}
	// 192.168.0.0/16 -> 192.168.0.0 – 192.168.255.255
	_, emeaRangeClassC, _ := net.ParseCIDR("192.168.0.0/16")
	if emeaRangeClassC.Contains(cidrRange.IP) {
		return false
	}
	return true
}

func getLatLon(ip string) (float64, float64) {
	geoLatLonFileContent, _ := ScanLines("./geo.csv")
	for _, val := range geoLatLonFileContent {
		line := strings.Split(val, ";")
		if line[0] == ip {
			lat, err := strconv.ParseFloat(line[1], 64)
			if err != nil {
				return 0.0, 0.0
			}
			lon, err := strconv.ParseFloat(line[2], 64)
			if err != nil {
				return 0.0, 0.0
			}
			return lat, lon
		}
	}
	return 0.0, 0.0
}

func AddLatLon(ip string, lat float64, lon float64) {
	fileToWrite, err := os.OpenFile("./geo.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}

	lats := strconv.FormatFloat(lat, 'f', 6, 64)
	lons := strconv.FormatFloat(lat, 'f', 6, 64)
	//fmt.Println("Adding into: ./geo.csv " + ip + " with lat: " + lats + " & lon: " + lons)
	dataLine := "\n" + ip + ";" + lats + ";" + lons
	if _, err = fileToWrite.WriteString(dataLine); err != nil {
		fmt.Println("Error")
		panic(err)
	}
	fileToWrite.Close()
}

func RemoveLeftZeros(accountId string) string {
	// String to long
	accountIdInt, err := strconv.ParseInt(accountId, 10, 64)
	if err != nil {
		panic(err)
	}
	// Long to string
	accountIdStr := strconv.FormatInt(accountIdInt, 10)
	return accountIdStr
}
