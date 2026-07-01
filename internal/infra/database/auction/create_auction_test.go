package auction

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/juliocesarboaroli/auction-concurrency/internal/entity/auction_entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCreateAuction_AutoClose(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("deve fechar o leilão automaticamente após a duração configurada", func(mt *mtest.T) {
		// Resposta para o InsertOne
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		// Resposta para o UpdateOne (fechamento automático)
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: 1},
			{Key: "nModified", Value: 1},
		})

		closed := make(chan string, 1)

		repo := &AuctionRepository{
			Collection:      mt.Coll,
			auctionDuration: 100 * time.Millisecond,
			onAuctionClosed: func(id string) { closed <- id },
		}

		auctionEntity := &auction_entity.Auction{
			Id:          uuid.New().String(),
			ProductName: "Smartphone Teste",
			Category:    "Eletrônicos",
			Description: "Descrição suficientemente longa para passar na validação.",
			Condition:   auction_entity.New,
			Status:      auction_entity.Active,
			Timestamp:   time.Now(),
		}

		internalErr := repo.CreateAuction(context.Background(), auctionEntity)
		if internalErr != nil {
			t.Fatalf("CreateAuction retornou erro inesperado: %v", internalErr)
		}

		select {
		case closedId := <-closed:
			if closedId != auctionEntity.Id {
				t.Errorf("esperado id=%s, recebido id=%s", auctionEntity.Id, closedId)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("leilão não foi fechado automaticamente dentro do tempo esperado")
		}
	})

	mt.Run("não deve iniciar goroutine de fechamento se o insert falhar", func(mt *mtest.T) {
		// Simula falha no InsertOne (ex: duplicate key)
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    11000,
			Message: "duplicate key error",
		}))

		closed := make(chan string, 1)

		repo := &AuctionRepository{
			Collection:      mt.Coll,
			auctionDuration: 100 * time.Millisecond,
			onAuctionClosed: func(id string) { closed <- id },
		}

		auctionEntity := &auction_entity.Auction{
			Id:          uuid.New().String(),
			ProductName: "Produto Duplicado",
			Category:    "Categoria",
			Description: "Descrição suficientemente longa para passar na validação.",
			Condition:   auction_entity.Used,
			Status:      auction_entity.Active,
			Timestamp:   time.Now(),
		}

		internalErr := repo.CreateAuction(context.Background(), auctionEntity)
		if internalErr == nil {
			t.Fatal("esperado erro no CreateAuction quando o insert falha, mas não houve erro")
		}

		select {
		case <-closed:
			t.Fatal("goroutine de fechamento não deveria ter sido iniciada quando o insert falhou")
		case <-time.After(300 * time.Millisecond):
			// Correto: nenhum fechamento ocorreu
		}
	})
}
