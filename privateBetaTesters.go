package main

import (
	"context"
	"encoding/csv"
	"os"
)

func (cfg *apiConfig) insertTestersUsernames(ctx context.Context) error {
	file, err := os.Open("private_beta_testers_batch_one.csv")
	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	result, err := reader.ReadAll()
	if err != nil {
		return err
	}
	usernames := make([]string, len(result))
	for i, arr := range result {
		usernames[i] = arr[0]
	}
	_, err = cfg.DB.InsertBetaTesters(ctx, usernames)
	if err != nil {
		return err
	}
	return nil
}
