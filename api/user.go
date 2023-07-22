package api

import (
	"chat/utils"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"strings"
)

func processLine(buf []byte) []string {
	data := strings.Trim(string(buf), "\n")
	rep := strings.NewReplacer(
		"data: {",
		"\"data\": {",
	)
	data = rep.Replace(data)
	array := strings.Split(data, "\n")
	resp := make([]string, 0)
	for _, item := range array {
		item = fmt.Sprintf("{%s}", strings.TrimSpace(item))
		if item == "{data: [DONE]}" {
			break
		}
		var form ChatGPTStreamResponse
		if err := json.Unmarshal([]byte(item), &form); err != nil {
			log.Fatal(err)
		}
		choices := form.Data.Choices
		if len(choices) > 0 {
			resp = append(resp, choices[0].Delta.Content)
		}
	}
	return resp
}

func StreamRequest(model string, messages []ChatGPTMessage, token int, callback func(string)) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", utils.ConvertBody(ChatGPTRequest{
		Model:    model,
		Messages: messages,
		MaxToken: token,
		Stream:   true,
	}))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+viper.GetString("openai.anonymous"))

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.ProtoMajor != 2 {
		callback("OpenAI 异常: http/2 not supported")
		return
	}

	for {
		buf := make([]byte, 1024)
		n, err := res.Body.Read(buf)

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		for _, item := range processLine(buf[:n]) {
			callback(item)
		}
	}
}
