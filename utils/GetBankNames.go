package utils

import (
	"BIEAS_bot/enums"
	"BIEAS_bot/models"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

func GetBankNames(ctx context.Context, db *models.DataBase, account int) ([]string, error) {
	var bankNames []string

	banks, err := db.GetDocuments(ctx, "banks", bson.M{"account": account})
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

			bankNames = append(bankNames, bank.Name)
		}

		return bankNames, nil
	}
}
