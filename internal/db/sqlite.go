package db

import (
	"database/sql"
	"fmt"
	"telegram-summarizer/internal/logger"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// InitDB initializes the SQLite database
func InitDB(dbPath string) (*DB, error) {
	logger.Info("Initializing database: %s", dbPath)
	
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	db := &DB{conn: conn}
	
	// Create tables
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}
	
	logger.Info("✅ Database initialized successfully")
	return db, nil
}

// createTables creates the necessary database tables
func (db *DB) createTables() error {
	logger.Debug("Creating database tables...")
	
	// Messages table
	messagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT,
		message_text TEXT NOT NULL,
		message_length INTEGER,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	
	// Summaries table
	summariesTable := `
	CREATE TABLE IF NOT EXISTS summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		summary_type TEXT,
		period_start DATETIME,
		period_end DATETIME,
		summary_text TEXT,
		message_count INTEGER,
		sentiment TEXT,
		credibility_score INTEGER,
		products_mentioned TEXT,
		red_flags_count INTEGER,
		validation_status TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	
	// Product mentions table
	productMentionsTable := `
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
	);`
	
	// Tracked groups table
	trackedGroupsTable := `
	CREATE TABLE IF NOT EXISTS tracked_groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER UNIQUE NOT NULL,
		group_name TEXT,
		group_username TEXT,
		join_date DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_active INTEGER DEFAULT 0,
		last_message_date DATETIME,
		summary_enabled_date DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	
	// Create indexes
	messagesIndex := `
	CREATE INDEX IF NOT EXISTS idx_messages_chat_time 
	ON messages(chat_id, timestamp);`
	
	summariesIndex := `
	CREATE INDEX IF NOT EXISTS idx_summaries_chat_time 
	ON summaries(chat_id, period_start);`
	
	trackedGroupsIndex := `
	CREATE INDEX IF NOT EXISTS idx_tracked_groups_active
	ON tracked_groups(is_active, last_message_date);`
	
	productMentionsIndex1 := `
	CREATE INDEX IF NOT EXISTS idx_product_mentions_summary
	ON product_mentions(summary_id);`
	
	productMentionsIndex2 := `
	CREATE INDEX IF NOT EXISTS idx_product_mentions_name
	ON product_mentions(product_name, created_at);`
	
	// Execute all statements
	statements := []string{
		messagesTable,
		summariesTable,
		trackedGroupsTable,
		productMentionsTable,
		messagesIndex,
		summariesIndex,
		trackedGroupsIndex,
		productMentionsIndex1,
		productMentionsIndex2,
	}
	
	for _, stmt := range statements {
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}
	
	logger.Debug("✅ Tables created successfully")
	return nil
}

// SaveMessage saves a message to the database
func (db *DB) SaveMessage(msg *Message) error {
	logger.Debug("Saving message: ChatID=%d, UserID=%d, Length=%d", 
		msg.ChatID, msg.UserID, msg.MessageLength)
	
	query := `
		INSERT INTO messages (chat_id, user_id, username, message_text, message_length, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)`
	
	result, err := db.conn.Exec(query, 
		msg.ChatID, 
		msg.UserID, 
		msg.Username, 
		msg.MessageText, 
		msg.MessageLength,
		msg.Timestamp,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	
	id, _ := result.LastInsertId()
	msg.ID = id
	
	logger.Info("✅ Message saved: ID=%d, User=%s, ChatID=%d", id, msg.Username, msg.ChatID)
	return nil
}

// GetMessagesByTimeRange retrieves messages within a time range
func (db *DB) GetMessagesByTimeRange(chatID int64, startTime, endTime time.Time) ([]Message, error) {
	logger.Debug("Fetching messages: ChatID=%d, From=%s, To=%s", 
		chatID, startTime.Format("15:04:05"), endTime.Format("15:04:05"))
	
	query := `
		SELECT id, chat_id, user_id, username, message_text, message_length, timestamp, created_at
		FROM messages
		WHERE chat_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC`
	
	rows, err := db.conn.Query(query, chatID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()
	
	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.UserID,
			&msg.Username,
			&msg.MessageText,
			&msg.MessageLength,
			&msg.Timestamp,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	
	logger.Info("📊 Found %d messages in time range", len(messages))
	return messages, nil
}

// DeleteMessagesByTimeRange deletes messages within a time range for a chat
func (db *DB) DeleteMessagesByTimeRange(chatID int64, startTime, endTime time.Time) error {
	logger.Debug("Deleting messages: ChatID=%d, Range=%s to %s", 
		chatID, startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"))
	
	query := `
		DELETE FROM messages
		WHERE chat_id = ?
		AND timestamp >= ?
		AND timestamp <= ?`
	
	result, err := db.conn.Exec(query, chatID, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	logger.Info("🗑️  Deleted %d messages for ChatID=%d", rowsAffected, chatID)
	
	return nil
}

// SaveSummary saves a summary to the database
func (db *DB) SaveSummary(summary *Summary) error {
	logger.Debug("Saving summary: ChatID=%d, Type=%s, Messages=%d", 
		summary.ChatID, summary.SummaryType, summary.MessageCount)
	
	query := `
		INSERT INTO summaries (
			chat_id, summary_type, period_start, period_end, summary_text, message_count,
			sentiment, credibility_score, products_mentioned, red_flags_count, validation_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	result, err := db.conn.Exec(query,
		summary.ChatID,
		summary.SummaryType,
		summary.PeriodStart,
		summary.PeriodEnd,
		summary.SummaryText,
		summary.MessageCount,
		summary.Sentiment,
		summary.CredibilityScore,
		summary.ProductsMentioned,
		summary.RedFlagsCount,
		summary.ValidationStatus,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}
	
	id, _ := result.LastInsertId()
	summary.ID = id
	
	logger.Info("✅ Summary saved: ID=%d, Type=%s, ChatID=%d", id, summary.SummaryType, summary.ChatID)
	return nil
}

// GetLastSummaryTime gets the end time of the last summary for a chat
func (db *DB) GetLastSummaryTime(chatID int64, summaryType string) (time.Time, error) {
	logger.Debug("Getting last summary time: ChatID=%d, Type=%s", chatID, summaryType)
	
	query := `
		SELECT period_end
		FROM summaries
		WHERE chat_id = ? AND summary_type = ?
		ORDER BY period_end DESC
		LIMIT 1`
	
	var lastTime time.Time
	err := db.conn.QueryRow(query, chatID, summaryType).Scan(&lastTime)
	
	if err == sql.ErrNoRows {
		logger.Debug("No previous summary found")
		return time.Time{}, nil
	}
	
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last summary time: %w", err)
	}
	
	logger.Debug("Last summary time: %s", lastTime.Format("2006-01-02 15:04:05"))
	return lastTime, nil
}

// GetSummaries retrieves summaries for a chat
func (db *DB) GetSummaries(chatID int64, summaryType string, limit int) ([]Summary, error) {
	logger.Debug("Fetching summaries: ChatID=%d, Type=%s, Limit=%d", chatID, summaryType, limit)
	
	query := `
		SELECT id, chat_id, summary_type, period_start, period_end, summary_text, message_count, created_at
		FROM summaries
		WHERE chat_id = ? AND summary_type = ?
		ORDER BY period_start DESC
		LIMIT ?`
	
	rows, err := db.conn.Query(query, chatID, summaryType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer rows.Close()
	
	var summaries []Summary
	for rows.Next() {
		var s Summary
		err := rows.Scan(
			&s.ID,
			&s.ChatID,
			&s.SummaryType,
			&s.PeriodStart,
			&s.PeriodEnd,
			&s.SummaryText,
			&s.MessageCount,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	
	logger.Info("📊 Found %d summaries", len(summaries))
	return summaries, nil
}

// AddTrackedGroup adds a group to tracking list (or updates name if exists)
func (db *DB) AddTrackedGroup(chatID int64, groupName string, groupUsername string) error {
	logger.Debug("Adding tracked group: ChatID=%d, Name=%s", chatID, groupName)
	
	// First, check if group already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM tracked_groups WHERE chat_id = ?)`
	err := db.conn.QueryRow(checkQuery, chatID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}
	
	if exists {
		// Update only name and username, preserve is_active and other fields
		updateQuery := `
			UPDATE tracked_groups 
			SET group_name = ?, group_username = ?
			WHERE chat_id = ?`
		_, err := db.conn.Exec(updateQuery, groupName, groupUsername, chatID)
		if err != nil {
			return fmt.Errorf("failed to update tracked group: %w", err)
		}
		logger.Debug("Updated group name: %s (ID: %d)", groupName, chatID)
	} else {
		// Insert new group with default is_active = 0
		insertQuery := `
			INSERT INTO tracked_groups (chat_id, group_name, group_username, is_active)
			VALUES (?, ?, ?, 0)`
		_, err := db.conn.Exec(insertQuery, chatID, groupName, groupUsername)
		if err != nil {
			return fmt.Errorf("failed to insert tracked group: %w", err)
		}
		logger.Info("✅ New group tracked: %s (ID: %d)", groupName, chatID)
	}
	
	return nil
}

// UpdateGroupActivity updates last message date for a group
func (db *DB) UpdateGroupActivity(chatID int64, lastMessageDate time.Time) error {
	logger.Debug("Updating group activity: ChatID=%d", chatID)
	
	query := `
		UPDATE tracked_groups 
		SET last_message_date = ?
		WHERE chat_id = ?`
	
	_, err := db.conn.Exec(query, lastMessageDate, chatID)
	if err != nil {
		return fmt.Errorf("failed to update group activity: %w", err)
	}
	
	return nil
}

// GetTrackedGroup gets a single tracked group by chat ID
func (db *DB) GetTrackedGroup(chatID int64) *TrackedGroup {
	logger.Debug("Fetching tracked group: ChatID=%d", chatID)
	
	query := `
		SELECT chat_id, group_name, group_username, join_date, is_active, last_message_date
		FROM tracked_groups
		WHERE chat_id = ?
		LIMIT 1`
	
	var g TrackedGroup
	var lastMessageDate sql.NullTime
	
	err := db.conn.QueryRow(query, chatID).Scan(
		&g.ChatID,
		&g.GroupName,
		&g.GroupUsername,
		&g.JoinDate,
		&g.IsActive,
		&lastMessageDate,
	)
	
	if err == sql.ErrNoRows {
		logger.Debug("Group not found: ChatID=%d", chatID)
		return nil
	}
	
	if err != nil {
		logger.Error("Failed to query tracked group: %v", err)
		return nil
	}
	
	if lastMessageDate.Valid {
		g.LastMessageDate = lastMessageDate.Time
	}
	
	return &g
}

// GetTrackedGroups gets all tracked groups
func (db *DB) GetTrackedGroups() []TrackedGroup {
	logger.Debug("Fetching tracked groups...")
	
	query := `
		SELECT chat_id, group_name, group_username, join_date, is_active, last_message_date
		FROM tracked_groups
		WHERE is_active >= 0
		ORDER BY last_message_date DESC`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		logger.Error("Failed to query tracked groups: %v", err)
		return nil
	}
	defer rows.Close()
	
	var groups []TrackedGroup
	for rows.Next() {
		var g TrackedGroup
		var lastMessageDate sql.NullTime
		
		err := rows.Scan(
			&g.ChatID,
			&g.GroupName,
			&g.GroupUsername,
			&g.JoinDate,
			&g.IsActive,
			&lastMessageDate,
		)
		if err != nil {
			logger.Error("Failed to scan tracked group: %v", err)
			continue
		}
		
		if lastMessageDate.Valid {
			g.LastMessageDate = lastMessageDate.Time
		}
		
		groups = append(groups, g)
	}
	
	logger.Info("📊 Found %d tracked groups", len(groups))
	return groups
}

// EnableGroupSummary enables summarization for a group
func (db *DB) EnableGroupSummary(chatID int64) error {
	logger.Info("Enabling summary for ChatID=%d", chatID)
	
	query := `
		UPDATE tracked_groups 
		SET is_active = 1, summary_enabled_date = CURRENT_TIMESTAMP
		WHERE chat_id = ?`
	
	_, err := db.conn.Exec(query, chatID)
	if err != nil {
		return fmt.Errorf("failed to enable group summary: %w", err)
	}
	
	logger.Info("✅ Summary enabled for ChatID=%d", chatID)
	return nil
}

// DisableGroupSummary disables summarization for a group
func (db *DB) DisableGroupSummary(chatID int64) error {
	logger.Info("Disabling summary for ChatID=%d", chatID)
	
	query := `
		UPDATE tracked_groups 
		SET is_active = 0
		WHERE chat_id = ?`
	
	_, err := db.conn.Exec(query, chatID)
	if err != nil {
		return fmt.Errorf("failed to disable group summary: %w", err)
	}
	
	logger.Info("✅ Summary disabled for ChatID=%d", chatID)
	return nil
}

// GetActiveGroups gets all groups with summarization enabled
func (db *DB) GetActiveGroups() []TrackedGroup {
	logger.Debug("Fetching active groups...")
	
	query := `
		SELECT chat_id, group_name, group_username, join_date, is_active, last_message_date
		FROM tracked_groups
		WHERE is_active = 1
		ORDER BY last_message_date DESC`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		logger.Error("Failed to query active groups: %v", err)
		return nil
	}
	defer rows.Close()
	
	var groups []TrackedGroup
	for rows.Next() {
		var g TrackedGroup
		var lastMessageDate sql.NullTime
		
		err := rows.Scan(
			&g.ChatID,
			&g.GroupName,
			&g.GroupUsername,
			&g.JoinDate,
			&g.IsActive,
			&lastMessageDate,
		)
		if err != nil {
			logger.Error("Failed to scan active group: %v", err)
			continue
		}
		
		if lastMessageDate.Valid {
			g.LastMessageDate = lastMessageDate.Time
		}
		
		groups = append(groups, g)
	}
	
	logger.Info("📊 Found %d active groups", len(groups))
	return groups
}

// GetGroupMessageCount24h gets message count for a group in last 24 hours
func (db *DB) GetGroupMessageCount24h(chatID int64) int {
	query := `
		SELECT COUNT(*) as count
		FROM messages
		WHERE chat_id = ? 
		AND timestamp >= datetime('now', '-24 hours')`
	
	var count int
	err := db.conn.QueryRow(query, chatID).Scan(&count)
	if err != nil {
		logger.Error("Failed to get message count: %v", err)
		return 0
	}
	
	return count
}

// Close closes the database connection
func (db *DB) Close() error {
	logger.Info("Closing database connection")
	return db.conn.Close()
}

// GetSummariesByTimeRange gets summaries within a time range
func (db *DB) GetSummariesByTimeRange(chatID int64, summaryType string, start, end time.Time) []Summary {
	logger.Debug("Fetching summaries: ChatID=%d, Type=%s, Range=%s to %s", 
		chatID, summaryType, start.Format("15:04"), end.Format("15:04"))
	
	query := `
		SELECT id, chat_id, summary_type, period_start, period_end, 
		       summary_text, message_count, sentiment, credibility_score,
		       products_mentioned, red_flags_count, validation_status, created_at
		FROM summaries
		WHERE chat_id = ?
		AND summary_type = ?
		AND period_start >= ?
		AND period_end <= ?
		ORDER BY period_start ASC`
	
	rows, err := db.conn.Query(query, chatID, summaryType, start, end)
	if err != nil {
		logger.Error("Failed to query summaries: %v", err)
		return nil
	}
	defer rows.Close()
	
	var summaries []Summary
	for rows.Next() {
		var s Summary
		var sentiment, products, validation sql.NullString
		var credibility, redFlags sql.NullInt64
		
		err := rows.Scan(
			&s.ID, &s.ChatID, &s.SummaryType,
			&s.PeriodStart, &s.PeriodEnd,
			&s.SummaryText, &s.MessageCount,
			&sentiment, &credibility, &products,
			&redFlags, &validation, &s.CreatedAt,
		)
		if err != nil {
			logger.Error("Failed to scan summary: %v", err)
			continue
		}
		
		if sentiment.Valid {
			s.Sentiment = sentiment.String
		}
		if credibility.Valid {
			s.CredibilityScore = int(credibility.Int64)
		}
		if products.Valid {
			s.ProductsMentioned = products.String
		}
		if redFlags.Valid {
			s.RedFlagsCount = int(redFlags.Int64)
		}
		if validation.Valid {
			s.ValidationStatus = validation.String
		}
		
		summaries = append(summaries, s)
	}
	
	logger.Info("Found %d summaries in range", len(summaries))
	return summaries
}

// DeleteMessagesOlderThan deletes messages older than the specified date
func (db *DB) DeleteMessagesOlderThan(chatID int64, beforeDate time.Time) (int64, error) {
	logger.Debug("Deleting messages older than %s for chat %d", beforeDate.Format("2006-01-02 15:04"), chatID)
	
	query := `DELETE FROM messages WHERE chat_id = ? AND timestamp < ?`
	
	result, err := db.conn.Exec(query, chatID, beforeDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old messages: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	logger.Info("🗑️  Deleted %d old messages from chat %d", rowsAffected, chatID)
	return rowsAffected, nil
}

// SaveProductMention saves a product mention to database
func (db *DB) SaveProductMention(pm *ProductMention) error {
	logger.Debug("Saving product mention: %s (SummaryID=%d)", pm.ProductName, pm.SummaryID)
	
	query := `
		INSERT INTO product_mentions (
			summary_id, product_name, mention_count,
			credibility_score, sentiment, validation_status, price_mentioned
		) VALUES (?, ?, ?, ?, ?, ?, ?)`
	
	result, err := db.conn.Exec(query,
		pm.SummaryID, pm.ProductName, pm.MentionCount,
		pm.CredibilityScore, pm.Sentiment, pm.ValidationStatus, pm.PriceMentioned,
	)
	if err != nil {
		return fmt.Errorf("failed to save product mention: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	
	pm.ID = id
	logger.Debug("✅ Product mention saved: ID=%d, Product=%s", pm.ID, pm.ProductName)
	return nil
}

// GetProductTrends gets product mentions over the specified number of days
func (db *DB) GetProductTrends(productName string, days int) []ProductMention {
	logger.Debug("Getting trends for product: %s (last %d days)", productName, days)
	
	query := `
		SELECT id, summary_id, product_name, mention_count,
		       credibility_score, sentiment, validation_status, price_mentioned, created_at
		FROM product_mentions
		WHERE product_name = ?
		AND created_at >= datetime('now', '-' || ? || ' days')
		ORDER BY created_at DESC`
	
	rows, err := db.conn.Query(query, productName, days)
	if err != nil {
		logger.Error("Failed to query product trends: %v", err)
		return nil
	}
	defer rows.Close()
	
	var mentions []ProductMention
	for rows.Next() {
		var pm ProductMention
		var price sql.NullString
		
		err := rows.Scan(
			&pm.ID, &pm.SummaryID, &pm.ProductName, &pm.MentionCount,
			&pm.CredibilityScore, &pm.Sentiment, &pm.ValidationStatus,
			&price, &pm.CreatedAt,
		)
		if err != nil {
			logger.Error("Failed to scan product mention: %v", err)
			continue
		}
		
		if price.Valid {
			pm.PriceMentioned = price.String
		}
		
		mentions = append(mentions, pm)
	}
	
	logger.Info("Found %d mentions of %s in last %d days", len(mentions), productName, days)
	return mentions
}
