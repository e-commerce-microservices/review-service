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
	log.Println("product id: ", req.GetProductId())
	if err != nil {
		return nil, err
	}
	if !resp.GetIsBought() {
		return nil, errors.New("Sản phẩm này chưa được mua")
	}

	// auth
	claims, err := srv.authClient.GetUserClaims(ctx, _empty)
	if err != nil {
		return nil, err
	}

	id, _ := strconv.ParseInt(claims.GetId(), 10, 64)

	review, err := srv.queries.InsertReview(ctx, repository.InsertReviewParams{
		UserID:    id,
		ProductID: req.GetProductId(),
		NumStar:   int32(req.GetNumStar()),
		Content:   req.GetContent(),
	})
	if err != nil {
		return nil, err
	}

	listImage := []string{}
	for _, dataChunk := range req.GetImageDataChunk() {
		thumbnail, err := uploadImage(ctx, dataChunk, srv.imageClient)
		if err != nil {
			log.Println("error when upload image: ", err)
			continue
		}
		err = srv.queries.InsertImage(ctx, repository.InsertImageParams{
			ReviewID: review.ID,
			ImageUrl: thumbnail,
		})
		if err == nil {
			listImage = append(listImage, thumbnail)
		}
	}

	// bought
	return &pb.CreateReviewResponse{
		Message: "Thêm review thành công",
		Review: &pb.Review{
			ReviewId:  review.ID,
			UserId:    id,
			ProductId: req.GetProductId(),
			ImageUrl:  listImage,
			NumStar:   review.NumStar,
			Content:   review.Content,
		},
	}, nil
}

func (srv reviewService) GetAllReviewByProductID(ctx context.Context, req *pb.GetAllReviewByProductIDRequest) (*pb.GetAllReviewByProductIDResponse, error) {

	reviews, err := srv.queries.GetAllReviewByProductID(ctx, req.GetProductId())
	if err != nil {
		return nil, err
	}

	result := make([]*pb.Review, 0, len(reviews))
	for _, review := range reviews {
		// get image
		images, _ := srv.queries.GetImagesByOrderID(ctx, review.ID)

		result = append(result, &pb.Review{
			ReviewId:  review.ID,
			UserId:    review.UserID,
			ProductId: review.ProductID,
			ImageUrl:  images,
			NumStar:   review.NumStar,
			Content:   review.Content,
		})
	}

	return &pb.GetAllReviewByProductIDResponse{
		ListReview: result,
	}, nil
}

func (srv reviewService) DeleteReview(ctx context.Context, req *pb.DeleteReviewRequest) (*pb.DeleteReviewResponse, error) {
	err := srv.queries.DeleteReview(ctx, req.GetReviewId())
	if err != nil {
		return nil, err
	}

	return &pb.DeleteReviewResponse{
		Message: "Xóa thành công",
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
