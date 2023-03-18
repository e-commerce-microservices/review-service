package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/e-commerce-microservices/review-service/pb"
	"github.com/e-commerce-microservices/review-service/repository"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	// postgres driver
	_ "github.com/lib/pq"
)

func main() {
	// create grpc server
	grpcServer := grpc.NewServer()

	// init user db connection
	pgDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWD"), os.Getenv("DB_DBNAME"),
	)
	conn, err := sql.Open("postgres", pgDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Ping(); err != nil {
		log.Fatal("can't ping to user db", err)
	}

	// init queries
	queries := repository.New(conn)

	// dial image client
	imageServiceConn, err := grpc.Dial("image-service:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatal("can't dial image service: ", err)
	}
	// create image client
	imageClient := pb.NewImageServiceClient(imageServiceConn)

	// dial auth client
	authServiceConn, err := grpc.Dial("auth-service:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatal("can't dial image service: ", err)
	}
	authClient := pb.NewAuthServiceClient(authServiceConn)

	// dial order client
	orderServiceConn, err := grpc.Dial("order-service:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatal("can't dial image service: ", err)
	}
	orderClient := pb.NewOrderServiceClient(orderServiceConn)

	// create review service
	service := reviewService{
		queries:     queries,
		authClient:  authClient,
		orderClient: orderClient,
		imageClient: imageClient,
	}
	// register product service
	pb.RegisterReviewServiceServer(grpcServer, service)

	// listen and serve
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("cannot create listener: ", err)
	}

	log.Printf("start gRPC server on %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("cannot create grpc server: ", err)
	}
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
}
