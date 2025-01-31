package gpt

import (
	"strings"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
	"github.com/qingconglaixueit/wechatbot/config"
)

// 3.5版本请求参数message结构
type Message struct {
	Role string `json:"role"`
	Content string `json:"content"`
}

// ChatGPTRequestBody 请求体
type ChatGPTRequestBody struct {
	Model            string  `json:"model"`
	Prompt           string  `json:"prompt,omitempty"`
	Messages         []Message  `json:"messages,omitempty"`
	MaxTokens        uint    `json:"max_tokens"`
	Temperature      float64 `json:"temperature"`
	TopP             int     `json:"top_p"`
	FrequencyPenalty int     `json:"frequency_penalty"`
	PresencePenalty  int     `json:"presence_penalty"`
}

// ChatGPTResponseBody 响应体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChoiceItem           `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
	Error   struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    interface{} `json:"code"`
	} `json:"error"`
}

type ChoiceItem struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	Logprobs     int    `json:"logprobs"`
	FinishReason string `json:"finish_reason"`
	Message Message `json:"message"`
}


// Completions gtp文本模型回复
//curl https://api.openai.com/v1/completions
//-H "Content-Type: application/json"
//-H "Authorization: Bearer your chatGPT key"
//-d '{"model": "text-davinci-003", "prompt": "give me good song", "temperature": 0, "max_tokens": 7}'
func Completions(msg string) (string, error) {
	var gptResponseBody *ChatGPTResponseBody
	var resErr error
	for retry := 1; retry <= 3; retry++ {
		if retry > 1 {
			time.Sleep(time.Duration(retry-1) * 100 * time.Millisecond)
		}
		gptResponseBody, resErr = httpRequestCompletions(msg, retry)
		if resErr != nil {
			log.Printf("gpt request(%d) error: %v\n", retry, resErr)
			continue
		}
		if gptResponseBody.Error.Message == "" {
			break
		}
	}
	if resErr != nil {
		return "", resErr
	}
	var reply string
	if gptResponseBody != nil && len(gptResponseBody.Choices) > 0 {
		if(strings.HasPrefix(gptResponseBody.Model, "gpt-3.5-turbo")){
			reply = gptResponseBody.Choices[0].Message.Content
		}else{
			reply = gptResponseBody.Choices[0].Text
		}
	}
	return reply, nil
}

func httpRequestCompletions(msg string, runtimes int) (*ChatGPTResponseBody, error) {
	cfg := config.LoadConfig()
	if cfg.ApiKey == "" {
		return nil, errors.New("api key required")
	}

	requestBody := ChatGPTRequestBody{
		Model:            cfg.Model,
		Prompt:           msg,
		MaxTokens:        cfg.MaxTokens,
		Temperature:      cfg.Temperature,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	httpUrl := "https://api.openai.com/v1/completions"
	if cfg.Model == "gpt-3.5-turbo" {
		httpUrl = "https://api.openai.com/v1/chat/completions"
		requestBody.Prompt = "";
		msgInfo := Message{
			Role: "user",
			Content: msg,
		}
		requestBody.Messages = []Message{msgInfo}
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal requestBody error: %v", err)
	}

	log.Printf("gpt request(%d) json: %s\n", runtimes, string(requestData))

	req, err := http.NewRequest(http.MethodPost, httpUrl, bytes.NewBuffer(requestData))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequest error: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
	
	// 创建代理地址
	proxyUrl, err := url.Parse("http://127.0.0.1:7890")
	if err != nil {
		panic(err)
	}
	transport := &http.Transport{
		Proxy:http.ProxyURL(proxyUrl),
		ResponseHeaderTimeout:time.Second*120,
	}
	
	client := &http.Client{Timeout: 30 * time.Second, Transport: transport}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.Do error: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll error: %v", err)
	}

	log.Printf("gpt response(%d) json: %s\n", runtimes, string(body))

	gptResponseBody := &ChatGPTResponseBody{}
	err = json.Unmarshal(body, gptResponseBody)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal responseBody error: %v", err)
	}
	return gptResponseBody, nil
}
