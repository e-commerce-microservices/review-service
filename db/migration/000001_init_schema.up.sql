CREATE TABLE
    IF NOT EXISTS review (
        "id" serial8 PRIMARY KEY,
        "user_id" serial8 NOT NULL,
        "product_id" serial8 NOT NULL,
        "num_star" integer NOT NULL DEFAULT(5)
    );

CREATE TABLE
    IF NOT EXISTS image (
        "id" serial8 PRIMARY KEY,
        "review_id" serial8 NOT NULL,
        "image_url" text NOT NULL
    );

ALTER TABLE image
ADD
    FOREIGN KEY ("review_id") REFERENCES review ("id") ON DELETE CASCADE;