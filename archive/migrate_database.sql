-- Database Migration Script
-- Adds new fields to summaries table and creates product_mentions table

-- Step 1: Backup existing data (optional but recommended)
-- Run: sqlite3 telegram_bot.db ".backup telegram_bot_backup.db"

-- Step 2: Add new columns to summaries table
ALTER TABLE summaries ADD COLUMN sentiment TEXT;
ALTER TABLE summaries ADD COLUMN credibility_score INTEGER;
ALTER TABLE summaries ADD COLUMN products_mentioned TEXT;
ALTER TABLE summaries ADD COLUMN red_flags_count INTEGER;
ALTER TABLE summaries ADD COLUMN validation_status TEXT;

-- Step 3: Create product_mentions table
CREATE TABLE IF NOT EXISTS product_mentions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    summary_id INTEGER NOT NULL,
    product_name TEXT,
    mention_count INTEGER,
    credibility_score INTEGER,
    sentiment TEXT,
    validation_status TEXT,
    price_mentioned TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (summary_id) REFERENCES summaries(id)
);

-- Step 4: Create indexes for product_mentions
CREATE INDEX IF NOT EXISTS idx_product_mentions_summary
ON product_mentions(summary_id);

CREATE INDEX IF NOT EXISTS idx_product_mentions_name
ON product_mentions(product_name, created_at);

-- Step 5: Verify schema
SELECT 'Migration completed successfully!' as status;
