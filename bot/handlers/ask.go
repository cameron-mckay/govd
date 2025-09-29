package handlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"os"
	"bytes"
	"encoding/json"
	"net/http"
)

type GptRequest struct {
	model string
	prompt string
	stream bool 
}
type GptResponse struct {
	response string
}

func AskHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	prompt := &GptRequest{}

	msg := ctx.EffectiveMessage.Text
	prompt.model = os.Getenv("MODEL_NAME")
	prompt.prompt = os.Getenv("MODEL_PROMPT")+msg
	prompt.stream = false
	
	payloadBuffer := new(bytes.Buffer)
	json.NewEncoder(payloadBuffer).Encode(prompt)

	req, err := http.NewRequest(http.MethodGet, os.Getenv("MODEL_URL"), payloadBuffer)
	if err != nil {
		return  err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return  err
	}

	defer res.Body.Close()

	decodedRes := new(GptResponse)

	err = json.NewDecoder(res.Body).Decode(decodedRes)
	if err != nil {
		return err
	}

	ctx.EffectiveMessage.Reply(bot, decodedRes.response, nil)

	return nil
}
