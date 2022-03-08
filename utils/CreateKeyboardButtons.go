package utils

import (
	"BIEAS_bot/enums"
	"BIEAS_bot/models"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

func CreateKeyboardButtons(ctx context.Context, db *models.DataBase, collection string, filter bson.M) ([]string, error) {
	var keyboardButtons []string

	banks, err := db.GetDocuments(ctx, collection, filter)
	defer banks.Close(ctx)

	if err != nil {
		return nil, errors.New(enums.UserErrors[enums.UNEXPECTED_ERROR])
	} else {
		if banks.RemainingBatchLength() < 1 {
			return nil, errors.New(enums.UserErrors[enums.NO_BANKS])
		}

		for banks.Next(ctx) {
			var bank models.Bank

			err = banks.Decode(&bank)
			if err != nil {
				return nil, errors.New(enums.UserErrors[enums.UNEXPECTED_ERROR])
			}

			keyboardButtons = append(keyboardButtons, bank.Name)
		}

		return keyboardButtons, nil
	}
}
