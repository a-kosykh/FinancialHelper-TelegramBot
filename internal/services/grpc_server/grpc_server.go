package grpcserver

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	pb "gitlab.ozon.dev/akosykh114/telegram-bot/api"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var addr = "localhost:50051"

func formatServiceLog(log string) string {
	return "<Report GRPC Server>: " + log
}

type GrpcReportServer struct {
	pb.UnimplementedReportSenderServer
	expencesChan chan []domain.Expence
}

func New() *GrpcReportServer {
	rv := &GrpcReportServer{
		expencesChan: make(chan []domain.Expence),
	}
	return rv
}

func (s *GrpcReportServer) GetReportExpencesChan() chan []domain.Expence {
	return s.expencesChan
}

func (s *GrpcReportServer) SendReport(ctx context.Context, reportMsg *pb.Report) (*pb.ReportResponse, error) {
	logger.Info(formatServiceLog("new message..."))

	userId := reportMsg.UserId.GetValue()
	var expences []domain.Expence
	for _, v := range reportMsg.Expences {
		e := &domain.Expence{
			ID:           v.Id.Value,
			UserID:       userId,
			CategoryID:   v.CategoryId.Value,
			CategoryName: v.CategoryName.GetValue(),
			Timestamp:    time.Unix(v.Ts.GetValue(), 0),
			Total:        v.Total.Value,
		}
		expences = append(expences, *e)
	}

	select {
	case <-ctx.Done():
		return &pb.ReportResponse{ResponseCode: wrapperspb.Int64(1)}, nil
	case s.expencesChan <- expences:
		//
	}

	return &pb.ReportResponse{ResponseCode: wrapperspb.Int64(1)}, nil
}

func (s *GrpcReportServer) StartService(ctx context.Context, wg *sync.WaitGroup) error {
	logger.Info(formatServiceLog("Starting server..."))

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	serv := grpc.NewServer()
	pb.RegisterReportSenderServer(serv, s)
	logger.Info(formatServiceLog(fmt.Sprintf("server listening - %v", addr)))

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := serv.Serve(lis); err != nil {
			logger.Fatal(formatServiceLog("serving error"), zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Info(formatServiceLog("Stopping server..."))
		serv.GracefulStop()
	}()

	return nil
}
