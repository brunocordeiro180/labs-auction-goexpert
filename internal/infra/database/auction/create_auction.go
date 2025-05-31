package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database"
	"fullcycle-auction_go/internal/internal_error"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
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
	Collection database.MongoCollectionInterface
}

func NewAuctionRepository(db *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: db.Collection("auctions"),
	}
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

	ar.CreateRoutineCheckExpire(context.Background(), auctionEntity)

	return nil
}

func (repo *AuctionRepository) CreateRoutineCheckExpire(ctx context.Context, auction *auction_entity.Auction) {
	go func() {
		checkIntervalStr := os.Getenv("AUCTION_INTERVAL")
		checkInterval, err := time.ParseDuration(checkIntervalStr)
		if err != nil {
			checkInterval = time.Minute
		}

		expirationThresholdStr := os.Getenv("AUCTION_EXPIRE")
		expirationThreshold, err := time.ParseDuration(expirationThresholdStr)
		if err != nil {
			expirationThreshold = 3 * time.Minute
		}

		for {
			log.Printf("Monitoring auction: %s (%s)", auction.Id, auction.ProductName)
			logger.Info("Monitoring auction", zap.String("id", auction.Id))

			elapsed := time.Since(auction.Timestamp)
			if elapsed >= expirationThreshold {
				log.Printf("Auction expired: %s", auction.Id)
				logger.Info("Marking auction as completed", zap.String("id", auction.Id))

				filter := bson.M{
					"_id":    auction.Id,
					"status": auction_entity.Active,
				}
				update := bson.M{
					"$set": bson.M{"status": auction_entity.Completed},
				}

				if _, err := repo.Collection.UpdateOne(ctx, filter, update); err != nil {
					log.Printf("Failed to update auction status: %v", err)
					logger.Error("Failed to update auction", err)
				}

				break
			}

			time.Sleep(checkInterval)
		}
	}()
}
