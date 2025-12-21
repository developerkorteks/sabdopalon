package client

import (
	"context"
	"fmt"
	"path/filepath"
	"telegram-summarizer/internal/db"
	"telegram-summarizer/internal/logger"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// Client represents the Telegram MTProto client
type Client struct {
	client    *telegram.Client
	api       *tg.Client
	db        *db.DB
	phone     string
	appID     int
	appHash   string
	sessionDir string
}

// Config holds client configuration
type Config struct {
	AppID      int
	AppHash    string
	Phone      string
	SessionDir string
	Database   *db.DB
}

// NewClient creates a new Telegram client
func NewClient(cfg Config) *Client {
	return &Client{
		phone:      cfg.Phone,
		appID:      cfg.AppID,
		appHash:    cfg.AppHash,
		sessionDir: cfg.SessionDir,
		db:         cfg.Database,
	}
}

// Start initializes and starts the client
func (c *Client) Start(ctx context.Context) error {
	logger.Info("🚀 Starting Telegram Client (gotd/td)...")
	logger.Info("📡 Connecting to Telegram servers...")
	logger.Info("⏳ This may take 10-30 seconds on first connection...")
	
	// Session storage
	sessionStorage := &session.FileStorage{
		Path: filepath.Join(c.sessionDir, "session.json"),
	}
	
	// Create client with update handler and options
	client := telegram.NewClient(c.appID, c.appHash, telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler: telegram.UpdateHandlerFunc(func(ctx context.Context, u tg.UpdatesClass) error {
			return c.handleUpdate(ctx, u)
		}),
		// Use DC2 (Singapore) - closer to Indonesia
		DC: 2,
	})
	
	c.client = client
	
	logger.Info("🔌 Client initialized, attempting to connect...")
	
	// Start client
	return client.Run(ctx, func(ctx context.Context) error {
		logger.Info("✅ Connection established!")
		logger.Info("🔐 Starting authentication...")
		
		// Get API
		c.api = client.API()
		
		// Authenticate
		if err := c.authenticate(ctx); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		
		logger.Info("✅ Client authenticated successfully")
		logger.Info("✅ Message handlers registered")
		
		// Fetch all dialogs (groups) at startup
		logger.Info("📋 Fetching all your groups...")
		if err := c.fetchAllDialogs(ctx); err != nil {
			logger.Error("Failed to fetch dialogs: %v", err)
		}
		
		logger.Info("📱 Client is ready to receive messages!")
		
		// Keep running
		<-ctx.Done()
		return ctx.Err()
	})
}

// authenticate handles user authentication
func (c *Client) authenticate(ctx context.Context) error {
	logger.Info("🔐 Authenticating...")
	
	flow := auth.NewFlow(
		auth.Constant(c.phone, "", auth.CodeAuthenticatorFunc(
			func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
				logger.Info("📱 Verification code sent to your Telegram app")
				logger.Info("Please enter the code:")
				
				var code string
				fmt.Print("> ")
				fmt.Scanln(&code)
				return code, nil
			},
		)),
		auth.SendCodeOptions{},
	)
	
	client := c.client.Auth()
	if err := client.IfNecessary(ctx, flow); err != nil {
		return err
	}
	
	// Get self info
	self, err := c.api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
	if err != nil {
		return fmt.Errorf("failed to get self: %w", err)
	}
	
	user := self.Users[0].(*tg.User)
	logger.Info("✅ Logged in as: %s %s (@%s)", user.FirstName, user.LastName, user.Username)
	logger.Info("   Phone: %s", user.Phone)
	logger.Info("   User ID: %d", user.ID)
	
	return nil
}

// handleUpdate processes all updates
func (c *Client) handleUpdate(ctx context.Context, u tg.UpdatesClass) error {
	switch updates := u.(type) {
	case *tg.Updates:
		// Regular updates with users and chats
		return c.processUpdates(ctx, updates.Updates, updates.Users, updates.Chats)
	case *tg.UpdatesCombined:
		// Combined updates
		return c.processUpdates(ctx, updates.Updates, updates.Users, updates.Chats)
	case *tg.UpdateShort:
		// Short update (no users/chats)
		return c.processSingleUpdate(ctx, updates.Update, nil, nil)
	case *tg.UpdateShortMessage:
		// Short message update
		logger.Debug("Short message update received")
	case *tg.UpdateShortChatMessage:
		// Short chat message update
		logger.Debug("Short chat message update received")
	}
	return nil
}

// processUpdates processes multiple updates with entities
func (c *Client) processUpdates(ctx context.Context, updates []tg.UpdateClass, users []tg.UserClass, chats []tg.ChatClass) error {
	// Build entities map
	userMap := make(map[int64]tg.UserClass)
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			userMap[user.ID] = user
		}
	}
	
	chatMap := make(map[int64]tg.ChatClass)
	for _, c := range chats {
		switch chat := c.(type) {
		case *tg.Chat:
			chatMap[chat.ID] = chat
		case *tg.Channel:
			chatMap[chat.ID] = chat
		}
	}
	
	// Process each update
	for _, update := range updates {
		c.processSingleUpdate(ctx, update, userMap, chatMap)
	}
	
	return nil
}

// processSingleUpdate processes a single update
func (c *Client) processSingleUpdate(ctx context.Context, update tg.UpdateClass, users map[int64]tg.UserClass, chats map[int64]tg.ChatClass) error {
	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		return c.handleMessage(ctx, u.Message, users, chats)
	case *tg.UpdateNewChannelMessage:
		return c.handleMessage(ctx, u.Message, users, chats)
	}
	return nil
}

// handleMessage processes a message
func (c *Client) handleMessage(ctx context.Context, messageClass tg.MessageClass, users map[int64]tg.UserClass, chats map[int64]tg.ChatClass) error {
	// Extract message
	msg, ok := messageClass.(*tg.Message)
	if !ok {
		return nil
	}
	
	// Skip if no text
	if msg.Message == "" {
		logger.Debug("Skipping: no text")
		return nil
	}
	
	// Filter: minimum length
	if len(msg.Message) < 10 {
		logger.Debug("Skipping: too short (%d chars)", len(msg.Message))
		return nil
	}
	
	// Get peer info
	var chatID int64
	var chatName string
	
	switch peer := msg.PeerID.(type) {
	case *tg.PeerChannel:
		chatID = peer.ChannelID
		// Try to get chat name from entities
		if chats != nil {
			if chat, ok := chats[peer.ChannelID]; ok {
				if channel, ok := chat.(*tg.Channel); ok {
					chatName = channel.Title
				}
			}
		}
		if chatName == "" {
			chatName = fmt.Sprintf("Channel_%d", peer.ChannelID)
		}
	case *tg.PeerChat:
		chatID = peer.ChatID
		// Try to get chat name from entities
		if chats != nil {
			if chat, ok := chats[peer.ChatID]; ok {
				if c, ok := chat.(*tg.Chat); ok {
					chatName = c.Title
				}
			}
		}
		if chatName == "" {
			chatName = fmt.Sprintf("Chat_%d", peer.ChatID)
		}
	default:
		// Skip private messages
		return nil
	}
	
	// Get sender info
	var userID int64
	var username string
	
	if msg.FromID != nil {
		switch from := msg.FromID.(type) {
		case *tg.PeerUser:
			userID = from.UserID
			if users != nil {
				if user, ok := users[from.UserID]; ok {
					if u, ok := user.(*tg.User); ok {
						username = u.Username
						if username == "" {
							username = u.FirstName
						}
					}
				}
			}
			if username == "" {
				username = fmt.Sprintf("User_%d", from.UserID)
			}
		}
	}
	
	// Add to tracked groups (auto-track) - so it appears in /listgroups
	c.db.AddTrackedGroup(chatID, chatName, "")
	
	// Update group activity (for message count statistics)
	c.db.UpdateGroupActivity(chatID, time.Unix(int64(msg.Date), 0))
	
	// ✅ CHECK if group is ACTIVE before saving message
	group := c.db.GetTrackedGroup(chatID)
	if group == nil || group.IsActive == 0 {
		logger.Debug("⏭️  Skipping message from inactive group: %s (ID: %d)", chatName, chatID)
		return nil  // Don't save message from inactive groups
	}
	
	// Save to database (only for ACTIVE groups)
	dbMsg := &db.Message{
		ChatID:        chatID,
		UserID:        userID,
		Username:      username,
		MessageText:   msg.Message,
		MessageLength: len(msg.Message),
		Timestamp:     time.Unix(int64(msg.Date), 0),
	}
	
	if err := c.db.SaveMessage(dbMsg); err != nil {
		logger.Error("Failed to save message: %v", err)
		return err
	}
	
	logger.Info("✅ [%s] %s: %s", chatName, username, truncateText(msg.Message, 50))
	
	return nil
}


// truncateText truncates text to max length
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// fetchAllDialogs fetches all groups at startup
func (c *Client) fetchAllDialogs(ctx context.Context) error {
	logger.Debug("Fetching all dialogs...")
	
	// Get all dialogs (chats, channels, groups)
	dialogs, err := c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100, // Get first 100
	})
	if err != nil {
		return fmt.Errorf("failed to get dialogs: %w", err)
	}
	
	var chats []tg.ChatClass
	
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	}
	
	logger.Info("📊 Found %d chats", len(chats))
	
	// Track all groups
	groupCount := 0
	for _, chat := range chats {
		switch ch := chat.(type) {
		case *tg.Channel:
			// Track channels/supergroups
			if ch.Megagroup || ch.Broadcast {
				c.db.AddTrackedGroup(ch.ID, ch.Title, ch.Username)
				groupCount++
				logger.Debug("  📂 %s (ID: %d)", ch.Title, ch.ID)
			}
		case *tg.Chat:
			// Track regular groups
			c.db.AddTrackedGroup(ch.ID, ch.Title, "")
			groupCount++
			logger.Debug("  📂 %s (ID: %d)", ch.Title, ch.ID)
		}
	}
	
	logger.Info("✅ Tracked %d groups successfully", groupCount)
	return nil
}

// Stop stops the client gracefully
func (c *Client) Stop() error {
	logger.Info("🛑 Stopping client...")
	// Client will stop when context is cancelled
	// No explicit Close() method in gotd/td
	return nil
}
