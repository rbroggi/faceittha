package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/pubsub"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/rbroggi/faceittha/internal/core/usecase"
	grpcactor "github.com/rbroggi/faceittha/internal/actors/grpc"
	subscriberactor "github.com/rbroggi/faceittha/internal/actors/pubsub/subscriber"
	produceractor "github.com/rbroggi/faceittha/internal/actors/pubsub/producer"
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
	grpcServerEndpoint = flag.String("grpc-server-endpoint", "localhost:50052", "gRPC server endpoint")
	httpServerEndpoint = flag.String("http-server-endpoint", "localhost:8081", "HTTP server endpoint")
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	projectID := os.Getenv("PUBSUB_PROJECT_ID")
	if projectID == "" {
		projectID = "faceittha"
	}
	userCDCSubscriptionID := os.Getenv("PUBSUB_USER_EVENT_SUBSCRIPTION_ID")
	if userCDCSubscriptionID == "" {
		userCDCSubscriptionID = "worker.cdc.faceittha.users.sub"
	}
	userEventPublicTopicID := os.Getenv("PUBSUB_PUBLIC_USER_EVENT_TOPIC")
	if userEventPublicTopicID == "" {
		userEventPublicTopicID = "shared.faceittha.UserEvents"
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	defer client.Close()

	topic := client.Topic(userEventPublicTopicID)
	producer, err := produceractor.NewProducer(topic)
	if err != nil {
		return err
	}
	
	informer := usecase.NewInformer(producer)

	subscription := client.Subscription(userCDCSubscriptionID)
	subscriber := subscriberactor.NewSubscriber(subscriberactor.SubscriberArgs{
		UserEventHandler: informer,
		Subscription: subscription,
	})

	// start subscriber
	go func(ctx context.Context) {
		if err := subscriber.Consume(ctx); err != nil {
			panic(err)
		}
	}(ctx)

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	err = pb.RegisterHealthServiceHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
	if err != nil {
		return err
	}

	// start http-gateway server
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