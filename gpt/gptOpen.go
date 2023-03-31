package gpt

import (
	"strings"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	"github.com/qingconglaixueit/wechatbot/config"
)

// 3.5版本请求参数message结构
type Message1 struct {
	Role string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest 请求体
type OpenAIRequest struct {
	Env string `json:"env"`
	Model            string  `json:"model"`
	Prompt           string  `json:"prompt,omitempty"`
	Messages         []Message  `json:"messages,omitempty"`
	MaxTokens        uint    `json:"max_tokens"`
	Temperature      float64 `json:"temperature"`
	TopP             int     `json:"top_p"`
	FrequencyPenalty int     `json:"frequency_penalty"`
	PresencePenalty  int     `json:"presence_penalty"`
}

// OpenAIResponse 响应体
type OpenAIResponse struct {
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

type ChoiceItem1 struct {
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
func Completions1(msg string) (string, error) {
	var openAIResponse *OpenAIResponse
	var resErr error
	for retry := 1; retry <= 3; retry++ {
		if retry > 1 {
			time.Sleep(time.Duration(retry-1) * 100 * time.Millisecond)
		}
		openAIResponse, resErr = oepnAiCompletions(msg, retry)
		if resErr != nil {
			log.Printf("gpt request(%d) error: %v\n", retry, resErr)
			continue
		}
		if openAIResponse.Error.Message == "" {
			break
		}
	}
	if resErr != nil {
		return "", resErr
	}
	var reply string
	if openAIResponse != nil && len(openAIResponse.Choices) > 0 {
		if(strings.HasPrefix(openAIResponse.Model, "gpt-3.5-turbo")){
			reply = openAIResponse.Choices[0].Message.Content
		}else{
			reply = openAIResponse.Choices[0].Text
		}
	}
	return reply, nil
}

func oepnAiCompletions(msg string, runtimes int) (*OpenAIResponse, error) {
	cfg := config.LoadConfig()

	requestBody := OpenAIRequest{
		Model:            cfg.Model,
		Prompt:           msg,
		MaxTokens:        cfg.MaxTokens,
		Temperature:      cfg.Temperature,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	httpUrl := "https://openforai.com/index.php?rest_route=/ai-chatbot/v1/chat"
	requestBody.Prompt = "";
	msgInfo := Message{
		Role: "user",
		Content: msg,
	}
	requestBody.Messages = []Message{msgInfo}

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
	client := &http.Client{Timeout: 30 * time.Second}
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

	OpenAIResponse := &OpenAIResponse{}
	err = json.Unmarshal(body, OpenAIResponse)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal responseBody error: %v", err)
	}
	return OpenAIResponse, nil
}
