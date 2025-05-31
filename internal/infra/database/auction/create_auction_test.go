package auction_test

import (
	"context"
	"os"
	"testing"
	"time"

	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockCollection simula a collection do MongoDB
type MockCollection struct {
	mock.Mock
}

func (m *MockCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document)
	result, _ := args.Get(0).(*mongo.InsertOneResult)
	return result, args.Error(1)
}

func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update)
	result, _ := args.Get(0).(*mongo.UpdateResult)
	return result, args.Error(1)
}

func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter)
	cursor, _ := args.Get(0).(*mongo.Cursor)
	return cursor, args.Error(1)
}

func (m *MockCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	args := m.Called(ctx, filter)
	resultObj := args.Get(0)
	err := args.Error(1)
	sr := mongo.NewSingleResultFromDocument(resultObj, err, bson.DefaultRegistry)
	return sr
}

// Testa a expiração automática usando mocks
func TestRoutineExpireWithMock(t *testing.T) {
	ctx := context.Background()

	os.Setenv("AUCTION_INTERVAL", "10ms")
	os.Setenv("AUCTION_EXPIRE", "20ms")

	mockCol := new(MockCollection)

	testAuction := &auction_entity.Auction{
		Id:          "mock-auction-1",
		ProductName: "Mocked Product",
		Timestamp:   time.Now().Add(-1 * time.Minute),
		Status:      auction_entity.Active,
	}

	// Espera-se que UpdateOne seja chamado com filtro e update corretos
	mockCol.On("UpdateOne", mock.Anything, mock.MatchedBy(func(filter interface{}) bool {
		f, ok := filter.(bson.M)
		return ok && f["_id"] == testAuction.Id && f["status"] == auction_entity.Active
	}), mock.Anything).Return(nil, nil).Once()

	repo := &auction.AuctionRepository{
		Collection: mockCol,
	}

	repo.CreateRoutineCheckExpire(ctx, testAuction)

	// Espera o goroutine rodar
	time.Sleep(50 * time.Millisecond)

	// Verifica se UpdateOne foi chamado como esperado
	mockCol.AssertExpectations(t)
	assert.True(t, true, "Auction expiration routine executed")
}
