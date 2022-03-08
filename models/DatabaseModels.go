package models

import (
	"context"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---------------------------------------------------------------------------
// ----------------------------------------------------------- DATABASE MODELS
type DataBase struct {
	Collections map[string]*mongo.Collection
}

func (db *DataBase) GetDocuments(ctx context.Context, collection string, filter bson.M) (*mongo.Cursor, error) {
	documents, err := db.Collections[collection].Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func (db *DataBase) GetDocument(ctx context.Context, collection string, filter bson.M) *mongo.SingleResult {
	return db.Collections[collection].FindOne(ctx, filter)
}

// Bank Models ---------------------------------------------------------------
type Bank struct {
	Id        string `json:"id" bson:"id"`
	Account   int    `json:"account" bson:"account"`
	Name      string `json:"name" bson:"name"`
	Balance   int    `json:"balance" bson:"balance"`
	CreatedAt string `json:"created_at" bson:"created_at"`
	UpdatedAt string `json:"updated_at" bson:"updated_at"`
}

func (bank *Bank) Create(ctx context.Context, db *DataBase) error {
	id, err := gonanoid.New()
	if err != nil {
		return err
	}

	bank.Id = id

	bank.Balance = 0

	bank.CreatedAt = time.Now().String()
	bank.UpdatedAt = time.Now().String()

	_, err = db.Collections["banks"].InsertOne(ctx, bank)
	if err != nil {
		return err
	}

	return nil
}

func (bank *Bank) Destroy(ctx context.Context, db *DataBase) error {
	if _, err := db.Collections["banks"].DeleteOne(
		ctx,
		bson.M{
			"account": bank.Account,
			"id":      bank.Id,
		},
	); err != nil {
		return err
	}

	return nil
}

func (bank *Bank) Update(ctx context.Context, db *DataBase, update bson.M) error {
	update["updated_at"] = time.Now().String()

	after := options.After
	options := &options.FindOneAndUpdateOptions{ReturnDocument: &after}
	err := db.Collections["banks"].FindOneAndUpdate(
		ctx,
		bson.M{
			"account": bank.Account,
			"id":      bank.Id,
		},
		bson.M{
			"$set": update,
		},
		options,
	).Decode(&bank)
	if err != nil {
		return err
	}

	return nil
}

// Operation Models ----------------------------------------------------------
type Operation struct {
	Id        string `json:"id" bson:"id"`
	Account   int    `json:"account" bson:"account"`
	Bank      string `json:"bank" bson:"bank"`
	Operation string `json:"operation" bson:"operation"`
	Amount    int    `json:"amount" bson:"amount"`
	Comment   string `json:"comment" bson:"comment"`
	CreatedAt string `json:"created_at" bson:"created_at"`
}

func (operation *Operation) Create(ctx context.Context, db *DataBase) error {
	id, err := gonanoid.New()
	if err != nil {
		return err
	}

	operation.Id = id

	operation.CreatedAt = time.Now().String()

	_, err = db.Collections["operations"].InsertOne(ctx, operation)
	if err != nil {
		return err
	}

	return nil
}
