-- Hapus index terlebih dahulu (Best practice)
DROP INDEX IF EXISTS idx_products_name;
DROP INDEX IF EXISTS idx_products_active;

-- Hapus tabel
DROP TABLE IF EXISTS products;