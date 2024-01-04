package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type CloudRequest struct {
	method      string
	params      map[string]string
	action      string
	productType string
	body        io.Reader
}

const (
	cckProductType   = "cck"
	PaasProductType  = "lb"
	version          = "2019-08-08"
	signatureVersion = "1.0"
	signatureMethod  = "HMAC-SHA1"
	timeStampFormat  = "2006-01-02T15:04:05Z"
)

func NewCCKRequest(action, method string, params map[string]string, body io.Reader) (*CloudRequest, error) {
	return NewRequest(action, method, params, cckProductType, body)
}

func NewPaasRequest(action, method string, params map[string]string, body io.Reader) (*CloudRequest, error) {
	return NewRequest(action, method, params, PaasProductType, body)
}

func NewRequest(action, method string, params map[string]string, productType string, body io.Reader) (*CloudRequest, error) {
	method = strings.ToUpper(method)
	req := &CloudRequest{
		method:      method,
		params:      params,
		action:      action,
		productType: productType,
		body:        body,
	}
	return req, nil
}

func NewOpenapiRequest(action, method string, params map[string]string, body, resp interface{}) (code int, err error) {
	var (
		response = &http.Response{}
		respBody []byte
		req      = &CloudRequest{}
	)
	if method == http.MethodGet {
		req, _ = NewPaasRequest(action, method, params, nil)
	} else {
		reqBytes, _ := json.Marshal(body)
		req, _ = NewPaasRequest(action, method, nil, bytes.NewReader(reqBytes))
	}

	response, err = DoOpenApiRequest(req)
	if err != nil {
		return
	}

	defer response.Body.Close()
	respBody, err = io.ReadAll(response.Body)
	if err != nil {
		return
	}

	code = response.StatusCode
	if len(respBody) != 0 && resp != nil {
		err = json.Unmarshal(respBody, resp)
		if err != nil {
			if len(respBody) > 1000 {
				return code, fmt.Errorf("%v", string(respBody[:200]))
			} else {
				return code, fmt.Errorf("%v", string(respBody))
			}
		}
	}
	return
}

func DoOpenApiRequest(req *CloudRequest) (resp *http.Response, err error) {
	if !IsAccessKeySet() {
		return nil, fmt.Errorf("AccessKeyID or accessKeySecret is empty")
	}
	reqUrl := getUrl(req)
	sendRequest, err := http.NewRequest(req.method, reqUrl, req.body)
	if err != nil {
		return
	}
	log.Infof("send request url: %s", reqUrl)
	resp, err = http.DefaultClient.Do(sendRequest)
	return
}

func DoRequest(method, url string, body io.Reader) (resp *http.Response, err error) {
	sendRequest, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}
	log.Infof("send request url: %s", url)
	resp, err = http.DefaultClient.Do(sendRequest)
	return
}

func getUrl(req *CloudRequest) string {
	urlParams := map[string]string{
		"Action":           req.action,
		"AccessKeyId":      AccessKeyID,
		"SignatureMethod":  signatureMethod,
		"SignatureNonce":   uuid.New().String(),
		"SignatureVersion": signatureVersion,
		"Timestamp":        time.Now().UTC().Format(timeStampFormat),
		"Version":          version,
	}
	if req.params != nil {
		for k, v := range req.params {
			urlParams[k] = v
		}
	}
	var paramSortKeys sort.StringSlice
	for k, _ := range urlParams {
		paramSortKeys = append(paramSortKeys, k)
	}
	sort.Sort(paramSortKeys)
	var urlStr string
	for _, k := range paramSortKeys {
		urlStr += "&" + percentEncode(k) + "=" + percentEncode(urlParams[k])
	}
	urlStr = req.method + "&%2F&" + percentEncode(urlStr[1:])

	h := hmac.New(sha1.New, []byte(AccessKeySecret))
	h.Write([]byte(urlStr))
	signStr := base64.StdEncoding.EncodeToString(h.Sum(nil))

	urlParams["Signature"] = signStr

	urlVal := url.Values{}
	for k, v := range urlParams {
		urlVal.Add(k, v)
	}
	urlValStr := urlVal.Encode()
	reqUrl := fmt.Sprintf("%s/%s?%s", APIHost, req.productType, urlValStr)
	return reqUrl
}

func percentEncode(str string) string {
	str = url.QueryEscape(str)
	strings.Replace(str, "+", "%20", -1)
	strings.Replace(str, "*", "%2A", -1)
	strings.Replace(str, "%7E", "~", -1)
	return str
}
