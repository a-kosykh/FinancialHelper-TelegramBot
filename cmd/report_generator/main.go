package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	_ "github.com/lib/pq"
	pb "gitlab.ozon.dev/akosykh114/telegram-bot/api"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/database"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var addr = "localhost:50051"

var (
	KafkaTopic         = "report-topic"
	KafkaConsumerGroup = "report-consumer-group"
	BrokersList        = []string{"localhost:9092"}
	Assignor           = "range"
)

var ExpencesDB *database.ExpencesDB
var grpcReportClient pb.ReportSenderClient

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := logger.InitLogger("data/zap_report_generator_config.json")
	// Инициализация логгера

	logger.Info("Initializing Report Generator (Kafka Comsumer)...")

	// Инициализация объектов слоя БД
	db, err := sql.Open("postgres", "host=localhost port=5432 dbname=telegram-bot-db user=postgres password=admin sslmode=disable")
	if err != nil {
		logger.Fatal("db open error", zap.Error(err))
	}
	defer db.Close()
	ExpencesDB = database.NewExpencesDB(db)

	// Установление соединение с grpc-сервером
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("grpc connection error", zap.Error(err))
	}
	defer conn.Close()

	grpcReportClient = pb.NewReportSenderClient(conn)

	if err := startConsumerGroup(ctx, BrokersList); err != nil {
		logger.Fatal("consumer group", zap.Error(err))
	}

	<-ctx.Done()

	logger.Info("Stopping (Kafka Comsumer)...")
}

func startConsumerGroup(ctx context.Context, brokerList []string) error {
	consumerGroupHandler := Consumer{}

	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0

	switch Assignor {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "round-robin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		logger.Fatal("Unrecognized consumer group partition assignor", zap.String("Assignor", Assignor))
	}

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup(brokerList, KafkaConsumerGroup, config)
	if err != nil {
		return err
	}

	err = consumerGroup.Consume(ctx, []string{KafkaTopic}, &consumerGroupHandler)
	if err != nil {
		return err
	}
	return nil

}

func processMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	logger.Info(fmt.Sprintf("New message received from topic:%s, offset:%d, partition:%d, key:%s,"+" value:%s\n", msg.Topic, msg.Offset, msg.Partition, string(msg.Key), string(msg.Value)))
	userID, err := strconv.ParseInt(string(msg.Key), 10, 64)
	if err != nil {
		return err
	}
	ts, err := strconv.ParseInt(string(msg.Value), 10, 64)
	if err != nil {
		return err
	}

	rv, err := ExpencesDB.GetUserExpences(ctx, domain.User{UserID: userID}, time.Unix(ts, 0))
	if err != nil {
		return err
	}
	fmt.Println(rv)

	logger.Info(fmt.Sprintf("Successful to read message: %s", string(msg.Value)))

	err = SendMessage(ctx, rv)
	if err != nil {
		return err
	}

	return nil
}

const (
	timestampFormat = time.StampNano // "Jan _2 15:04:05.000"
)

func CreateMessage(expences []domain.Expence) *pb.Report {
	var userID int64 = -1
	if len(expences) > 0 {
		userID = expences[0].UserID
	}
	msg := &pb.Report{
		UserId: wrapperspb.Int64(userID),
	}

	var expencesField []*pb.Expence
	for _, v := range expences {
		e := &pb.Expence{
			Id:           wrapperspb.Int64(v.ID),
			CategoryId:   wrapperspb.Int64(v.CategoryID),
			CategoryName: wrapperspb.String(v.CategoryName),
			Ts:           wrapperspb.Int64(v.Timestamp.Unix()),
			Total:        wrapperspb.Int64(v.Total),
		}
		expencesField = append(expencesField, e)
	}
	msg.Expences = expencesField

	return msg
}

func SendMessage(ctx context.Context, expences []domain.Expence) error {
	md := metadata.Pairs("timestamp", time.Now().Format(timestampFormat))
	ctx = metadata.NewOutgoingContext(ctx, md)

	var header, trailer metadata.MD
	msg := CreateMessage(expences)
	_, err := grpcReportClient.SendReport(ctx, msg, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		return err
	}
	return nil
}

// Consumer represents a Sarama consumer group consumer.
type Consumer struct {
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	logger.Info("consumer - setup")
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	logger.Info("consumer - cleanup")
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			err := processMessage(session.Context(), message)
			if err != nil {
				return err
			}

			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
}
