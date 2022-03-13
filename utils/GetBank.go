package utils

import (
	"BIEAS_bot/enums"
	"BIEAS_bot/models"
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
)

func GetBank(ctx context.Context, db *models.DataBase, account int, name string) (*models.Bank, error) {
	var bank *models.Bank

	err := db.GetDocument(ctx, "banks", bson.M{
		"account": account,
		"name":    name,
	}).Decode(&bank)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, errors.New(enums.UserErrors[enums.BANK_NOT_FOUND])
		} else {
			return nil, err
		}
	}

	return bank, nil
}
