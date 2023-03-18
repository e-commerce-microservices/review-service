-- name: InsertReview :one

INSERT INTO
    review (
        "user_id",
        "product_id",
        "num_star",
        "content"
    )
VALUES ($1, $2, $3, $4) RETURNING  *;

-- name: InsertImage :exec

INSERT INTO image ("review_id", "image_url") VALUES ($1, $2) ;

-- name: SelectReviewByProductID :many

SELECT
    review.id,
    "user_id",
    "product_id",
    "num_star",
    "image_url",
    "content"
FROM review
    INNER JOIN image ON review.id = image.review_id
WHERE review.id = $1;

-- name: DeleteReview :exec

DELETE FROM review WHERE id = $1;

-- name: GetAllReviewByProductID :many
SELECT * FROM review
WHERE "product_id" = $1;

-- name: GetImagesByOrderID :many
SELECT "image_url" FROM "image"
WHERE "review_id" = $1;
