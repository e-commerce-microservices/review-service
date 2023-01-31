package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/e-commerce-microservices/review-service/pb"
	"github.com/e-commerce-microservices/review-service/repository"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/metadata"
)

type reviewService struct {
	queries     *repository.Queries
	authClient  pb.AuthServiceClient
	orderClient pb.OrderServiceClient
	imageClient pb.ImageServiceClient
	pb.UnimplementedReviewServiceServer
}

var _empty = &empty.Empty{}

func (srv reviewService) CreateReview(ctx context.Context, req *pb.CreateReviewRequest) (*pb.CreateReviewResponse, error) {
	// extract md
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("invalid request")
	}
	// inject md
	ctx = metadata.NewOutgoingContext(ctx, md)

	// check order is handled
	resp, err := srv.orderClient.CheckOrderIsHandled(ctx, &pb.CheckOrderIsHandledRequest{
		ProductId: req.GetProductId(),
	})
	if err != nil {
		return nil, err
	}
	if !resp.GetIsBought() {
		return nil, errors.New("un authorization")
	}

	// auth
	claims, err := srv.authClient.GetUserClaims(ctx, _empty)
	if err != nil {
		return nil, err
	}

	id, _ := strconv.ParseInt(claims.GetId(), 10, 64)

	reviewID, err := srv.queries.InsertReview(ctx, repository.InsertReviewParams{
		UserID:    id,
		ProductID: req.GetProductId(),
		NumStar:   int32(req.GetNumStar()),
	})
	if err != nil {
		return nil, err
	}

	for _, dataChunk := range req.GetImageDataChunk() {
		thumbnail, err := uploadImage(ctx, dataChunk, srv.imageClient)
		if err != nil {
			log.Println("error when upload image: ", err)
			continue
		}
		err = srv.queries.InsertImage(ctx, repository.InsertImageParams{
			ReviewID: int64(reviewID),
			ImageUrl: thumbnail,
		})
	}

	// bought
	return &pb.CreateReviewResponse{
		Message: "Tạo thành công",
	}, nil
}

func (srv reviewService) GetAllReview(context.Context, *pb.CreateReviewRequest) (*pb.CreateReviewResponse, error) {
	return &pb.CreateReviewResponse{
		Message: "",
	}, nil
}

func (srv reviewService) UpdateReview(context.Context, *pb.UpdateReviewRequest) (*pb.UpdateReviewResponse, error) {
	return &pb.UpdateReviewResponse{
		Message: "Cập nhật thành công",
	}, nil
}

func (srv reviewService) Ping(ctx context.Context, _ *empty.Empty) (*pb.Pong, error) {
	return &pb.Pong{
		Message: "pong",
	}, nil
}

func toBytes(str string) []byte {
	bytes, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		log.Fatal(err)
	}
	return bytes
}

func uploadImage(ctx context.Context, dataChunk string, imageClient pb.ImageServiceClient) (string, error) {
	// upload image
	stream, err := imageClient.UploadImage(ctx)
	if err != nil {
		return "", err
	}
	// send mime type
	tmp := strings.Split(dataChunk, "data:image/")
	mimeType := strings.Split(tmp[1], ";")[0]
	err = stream.Send(&pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				ImageType: mimeType,
			},
		},
	})
	if err != nil {
		return "", err
	}

	// send data
	dataChunk = strings.Split(dataChunk, ",")[1]
	stream.Send(&pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_ChunkData{
			ChunkData: toBytes(dataChunk),
		},
	})

	res, err := stream.CloseAndRecv()
	if err != nil {
		return "", err
	}

	return res.GetImageUrl(), nil
}
