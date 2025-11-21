package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/bot/core"
	"github.com/govdbot/govd/enums"
	"go.uber.org/zap"
)

type OllamaRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images,omitempty"`
	Stream bool     `json:"stream"`
}
type OllamaResponse struct {
	Response string `json:"response"`
}

func modelQuery(prompt OllamaRequest) (*OllamaResponse, error) {
	zap.S().Info("payload")
	payloadBuffer := new(bytes.Buffer)
	zap.S().Info("encoder")
	err := json.NewEncoder(payloadBuffer).Encode(prompt)
	if err != nil {
		return nil, err
	}

	zap.S().Info("request")
	req, err := http.NewRequest(http.MethodPost, os.Getenv("MODEL_URL"), payloadBuffer)
	if err != nil {
		return nil, err
	}
	zap.S().Info("header")

	req.Header.Set("Content-Type", "application/json")

	zap.S().Info("do")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	zap.S().Info("close")

	defer res.Body.Close()

	zap.S().Info("decoded")
	decodedRes := new(OllamaResponse)

	zap.S().Info("decoder")
	err = json.NewDecoder(res.Body).Decode(decodedRes)
	if err != nil {
		return nil, err
	}

	zap.S().Info(res.Status)

	zap.S().Info("return")
	return decodedRes, nil
}

func AskHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	textPrompt := &OllamaRequest{}
	sender := ctx.EffectiveSender.Name()
	textPrompt.Prompt = "The prompters name is " + sender + ".  Only acknowledge them by name if absolutely necessary.  "
	msg := ctx.EffectiveMessage.Text

	imgs := ctx.EffectiveMessage.Photo
	// check if image is attached
	if len(imgs) > 0 {
		// Set sending status
		core.SendingEffect(bot, ctx.EffectiveChat.Id, enums.MediaTypePhoto)

		img := imgs[len(imgs)-1]

		imagePrompt := &OllamaRequest{}
		imagePrompt.Model = os.Getenv("VISION_MODEL_NAME")
		imagePrompt.Prompt = "Describe the following image in relatively simple terms that a 5 year old could understand: "
		imagePrompt.Stream = false

		// Get file info
		file, err := bot.GetFile(img.FileId, nil)
		if err != nil {
			zap.S().Errorf("failed to get file url")
			return err
		}

		// Download the image
		resp, err := http.Get(file.URL(bot, nil))
		if err != nil {
			zap.S().Errorf("failed to download file")
			return err
		}
		defer resp.Body.Close()

		imgBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			zap.S().Errorf("unable to decode image")
			return err
		}

		// Encode to base64
		base64Str := base64.StdEncoding.EncodeToString(imgBytes)

		imagePrompt.Images = []string{base64Str}

		imageRes, err := modelQuery(*imagePrompt)

		if err != nil {
			zap.S().Errorf("unable to describe image")
			return err
		}

		textPrompt.Prompt += "The prompter has sent you an image with the following description and will either prompt you about it or expect you to describe the image in your own words: " + imageRes.Response + ".  "
	}

	textPrompt.Prompt += os.Getenv("TEXT_MODEL_PROMPT") + " The prompt is:" + msg
	// Set typing effect
	core.TypingEffect(bot, ctx.EffectiveChat.Id)
	textPrompt.Model = os.Getenv("TEXT_MODEL_NAME")
	textPrompt.Stream = false
	zap.S().Info("starting model query")

	zap.S().Debugf("Final model payload: %+v", textPrompt.Prompt)

	response, err := modelQuery(*textPrompt)
	zap.S().Info("got model response")
	if err != nil {
		zap.S().Errorf("unable to get response")
		return err
	}

	zap.S().Info("sending message")
	zap.S().Info(response.Response)
	_, err = ctx.EffectiveMessage.Reply(bot, strings.ReplaceAll(response.Response, "<|constrain|>", ""), nil)

	if err != nil {
		zap.S().Errorf(err.Error())
		return err
	}

	return nil
}
