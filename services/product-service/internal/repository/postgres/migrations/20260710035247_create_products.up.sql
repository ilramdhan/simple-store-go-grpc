CREATE TABLE products
(
    id          UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    price_cents BIGINT       NOT NULL CHECK (price_cents >= 0),
    stock       INTEGER      NOT NULL CHECK (stock >= 0),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(100) NOT NULL,
    updated_by  VARCHAR(100) NOT NULL,
    deleted_at  TIMESTAMPTZ -- NULL berarti aktif
);

-- 1. Partial Indexing untuk Soft Delete (Sangat cepat dan hemat RAM)
CREATE INDEX idx_products_active ON products (id) WHERE deleted_at IS NULL;

-- 2. Index untuk pencarian nama (Karena di domain kita punya filter SearchName)
CREATE INDEX idx_products_name ON products (name) WHERE deleted_at IS NULL;