package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	mongo2 "github.com/rbroggi/faceittha/internal/actors/mongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcactor "github.com/rbroggi/faceittha/internal/actors/grpc"
	"github.com/rbroggi/faceittha/internal/core/usecase"
	pb "github.com/rbroggi/faceittha/pkg/sdk/v1"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the DebugLevel severity or above.
	log.SetLevel(log.DebugLevel)
}

var (
	grpcServerEndpoint = flag.String("grpc-server-endpoint", "localhost:50051", "gRPC server endpoint")
	httpServerEndpoint = flag.String("http-server-endpoint", "localhost:8080", "HTTP server endpoint")
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	url := os.Getenv("MONGODB_URL")
	if url == "" {
		url = "mongodb://mongouser:mongopwd@localhost:27017/faceittha?authSource=admin&readPreference=primary&ssl=false&replicaSet=rs0"
	}
	// Set client options
	clientOptions := options.Client().ApplyURI(url)
	db, err := mongo.Connect(ctx, clientOptions)
	if err := db.Ping(ctx, nil); err != nil {
		log.WithError(err).Error("db does not appear to be reachable")
		return err

	}
	defer db.Disconnect(ctx)
	collection := db.Database("faceittha").Collection("users")

	mongoActor, err := mongo2.NewMongoDB(mongo2.MongoDBArgs{UserCollection: collection})
	if err != nil {
		log.WithError(err).Error("could not initialize mongo actor")
		return err
	}
	userSvcUsecase := usecase.NewUserService(usecase.UserServiceArgs{Repository: mongoActor})
	userServer := grpcactor.NewUserService(grpcactor.UserServiceArgs{Usecase: userSvcUsecase})

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	err = pb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
	if err != nil {
		return err
	}

	err = pb.RegisterHealthServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
	if err != nil {
		return err
	}

	go func() {
		if err := http.ListenAndServe(*httpServerEndpoint, mux); err != nil {
			panic(err)
		}
	}()

	lis, err := net.Listen("tcp", *grpcServerEndpoint)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, userServer)
	pb.RegisterHealthServiceServer(s, &grpcactor.HealthService{})

	// Register reflection service on gRPC server.
	reflection.Register(s)

	// Start gRPC server
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()

	log.
		WithField("http-server-addr", *httpServerEndpoint).
		WithField("grpc-server-addr", *grpcServerEndpoint).
		Info("servers up or soon to be up. listening to SIGTERM, SIGINT, SIGQUIT for stoping the server")

	// Wait for signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-ch

	// Stop server
	s.GracefulStop()

	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}
