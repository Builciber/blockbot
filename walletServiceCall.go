package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type ReqBody struct {
	TelegramID int64 `json:"telegram_id"`
}

type ErrorResp struct {
	Error string `json:"error"`
}

func WalletServiceCall(method, url, apikey string, requestBody interface{}, responseBody interface{}) error {
	// `responseBody` must be a pointer to a struct
	b, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+apikey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		errMsg := ErrorResp{}
		err = json.Unmarshal(body, &errMsg)
		if err != nil {
			return err
		}
		return errors.New(errMsg.Error)
	}
	err = json.Unmarshal(body, responseBody)
	if err != nil {
		return err
	}
	return nil
}
