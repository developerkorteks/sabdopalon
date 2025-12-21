package db

import "time"

// Message represents a chat message
type Message struct {
	ID            int64
	ChatID        int64
	UserID        int64
	Username      string
	MessageText   string
	MessageLength int
	Timestamp     time.Time
	CreatedAt     time.Time
}

// Summary represents a generated summary
type Summary struct {
	ID               int64
	ChatID           int64
	SummaryType      string // '4h', 'daily', 'manual-24h'
	PeriodStart      time.Time
	PeriodEnd        time.Time
	SummaryText      string
	MessageCount     int
	Sentiment        string // 'positive', 'neutral', 'negative'
	CredibilityScore int    // 1-5, overall credibility
	ProductsMentioned string // JSON array of product names
	RedFlagsCount    int    // Number of propaganda/spam indicators
	ValidationStatus string // 'valid', 'mixed', 'suspicious'
	CreatedAt        time.Time
}

// TrackedGroup represents a tracked Telegram group
type TrackedGroup struct {
	ChatID          int64
	GroupName       string
	GroupUsername   string
	JoinDate        time.Time
	IsActive        int // 0=scrape only, 1=summarize
	LastMessageDate time.Time
}

// ProductMention represents a product mentioned in a summary
type ProductMention struct {
	ID               int64
	SummaryID        int64
	ProductName      string
	MentionCount     int
	CredibilityScore int    // 1-5, credibility of this product
	Sentiment        string // 'positive', 'neutral', 'negative'
	ValidationStatus string // 'valid', 'suspicious'
	PriceMentioned   string
	CreatedAt        time.Time
}
