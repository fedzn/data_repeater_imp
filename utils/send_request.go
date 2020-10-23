package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	SourceHostPort string = "127.0.0.1:2303" // data sorter 服务地址
)

// 给不同的服务发送POST请求
func PostRequest(url string, body_text string) ([]byte, error) {
	return SendRequest(true, url, body_text)
}

// 给不同的服务发送GET请求
func GetRequest(url string, body_text string) ([]byte, error) {
	return SendRequest(false, url, body_text)
}

// 给不同的服务发送请求
func SendRequest(is_post bool, url string, body_text string) ([]byte, error) {
	request_type := "GET"
	if is_post {
		request_type = "POST"
	}
	request, err := http.NewRequest(request_type, url, strings.NewReader(body_text))
	if err != nil {
		log.Printf("http.NewRequest,[err=%s][url=%s]", err, url)
		return []byte(""), err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Connection", "Keep-Alive")

	var resp *http.Response
	resp, err = http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, url)
		return []byte(""), err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("http.Do failed,[err=%s][url=%s]", err, url)
	}
	return b, err
}

// 转发GET请求
func RepeatGetRequest(c *gin.Context) {
	request_route := c.Request.RequestURI
	log.Println("Recv Request  :", request_route)
	code := http.StatusOK

	url := fmt.Sprintf("http://%s%s", SourceHostPort, request_route)
	resp_data, err := GetRequest(url, "")
	if err != nil {
		log.Printf("PostRequest failed,[err=%s][url=%s]", err, url)

		code = http.StatusInternalServerError
	}

	log.Println("Recv Response :", string(resp_data))
	c.String(code, string(resp_data))
}

// 转发POST请求
func RepeatPostRequest(c *gin.Context) {
	request_route := c.Request.RequestURI
	log.Println("Recv Request  :", request_route)
	code := http.StatusOK

	url := fmt.Sprintf("http://%s%s", SourceHostPort, request_route)
	resp_data, err := PostRequest(url, "")
	if err != nil {
		log.Printf("PostRequest failed,[err=%s][url=%s]", err, url)

		code = http.StatusInternalServerError
	}

	log.Println("Recv Response :", string(resp_data))
	c.String(code, string(resp_data))
}
