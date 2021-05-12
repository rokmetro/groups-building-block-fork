package utils

import (
	"log"
	"net/http"
)

//LogRequest logs the request as hide some header fields because of security reasons
func LogRequest(req *http.Request) {
	if req == nil {
		return
	}

	method := req.Method
	path := req.URL.Path

	header := make(map[string][]string)
	for key, value := range req.Header {
		var logValue []string
		//do not log api key, cookies and Authorization
		if key == "Rokwire-Api-Key" || key == "Cookie" || key == "Authorization" || key == "Rokwire_gs_api_key" {
			logValue = append(logValue, "---")
		} else {
			logValue = value
		}
		header[key] = logValue
	}
	log.Printf("%s %s %s", method, path, header)
}

//Equal compares two slices
func Equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

//EqualPointers compares two pointers slices
func EqualPointers(a, b *[]string) bool {
	if a == nil && b == nil {
		return true //equals
	}
	if a != nil && b == nil {
		return false // not equals
	}
	if a == nil && b != nil {
		return false // not equals
	}

	//both are not nil
	return Equal(*a, *b)
}
