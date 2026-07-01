package auction

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/juliocesarboaroli/auction-concurrency/configuration/logger"
	"github.com/juliocesarboaroli/auction-concurrency/internal/entity/auction_entity"
	"github.com/juliocesarboaroli/auction-concurrency/internal/internal_error"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection      *mongo.Collection
	auctionDuration time.Duration
	onAuctionClosed func(string) // hook para testes; nil em produção
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionDuration: getAuctionDuration(),
	}
}

func getAuctionDuration() time.Duration {
	duration, err := time.ParseDuration(os.Getenv("AUCTION_DURATION"))
	if err != nil {
		return 5 * time.Minute
	}
	return duration
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}

	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	go ar.closeAuction(auctionEntity.Id, ar.auctionDuration)

	return nil
}

func (ar *AuctionRepository) closeAuction(auctionId string, duration time.Duration) {
	time.Sleep(duration)

	filter := bson.M{"_id": auctionId}
	update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}

	if _, err := ar.Collection.UpdateOne(context.Background(), filter, update); err != nil {
		logger.Error(fmt.Sprintf("Error trying to close auction id=%s", auctionId), err)
		return
	}

	logger.Info(fmt.Sprintf("Auction id=%s closed automatically", auctionId))

	if ar.onAuctionClosed != nil {
		ar.onAuctionClosed(auctionId)
	}
}
