package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-pg/pg/v10"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcactor "github.com/rbroggi/faceittha/internal/actors/grpc"
	"github.com/rbroggi/faceittha/internal/actors/postgres"
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

	url := os.Getenv("POSTGRESQL_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	pgopts, err := pg.ParseURL(url)
	if err != nil {
		log.WithError(err).Error("while parsing Postgres URL")
		return err
	}
	db := pg.Connect(pgopts)
	if err := db.Ping(context.Background()); err != nil {
		log.WithError(err).Error("db does not appear to be reachable")
		return err

	}
	pgDB, err := postgres.NewPostgresDB(postgres.PostgresDBArgs{DB: db})
	if err != nil {
		log.WithError(err).Error("error instantiating PostgresDB")
		return err
	}
	userSvcUsecase := usecase.NewUserService(usecase.UserServiceArgs{Repository: pgDB})
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
