package dbus

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/khaapp/khaapp-daemon/protocol"
	"github.com/khaapp/khaapp-daemon/storage"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow/types"
	waevents "go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

const (
	ServiceName   = "org.khaapp.Daemon"
	ObjectPath    = godbus.ObjectPath("/org/khaapp/Daemon")
	InterfaceName = "org.khaapp.IMessenger"
)

type cachedGroupInfo struct {
	info      map[string]godbus.Variant
	expiresAt time.Time
}

type MessengerService struct {
	conn         *godbus.Conn
	client       *protocol.WAClient
	messageStore *storage.MessageStore
	logger       *zap.Logger

	mu     sync.RWMutex
	status string

	cacheMu         sync.Mutex
	groupInfoCache  map[string]cachedGroupInfo
	avatarDownloads map[string]bool
}

func ConnectSessionBus() (*godbus.Conn, error) {
	conn, err := godbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}

	reply, err := conn.RequestName(ServiceName, godbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}

	if reply != godbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return nil, fmt.Errorf("D-Bus name %s is already owned", ServiceName)
	}

	return conn, nil
}

func NewMessengerService(conn *godbus.Conn, client *protocol.WAClient, messageStore *storage.MessageStore, logger *zap.Logger) *MessengerService {
	return &MessengerService{
		conn:            conn,
		client:          client,
		messageStore:    messageStore,
		logger:          logger,
		status:          "disconnected",
		groupInfoCache:  make(map[string]cachedGroupInfo),
		avatarDownloads: make(map[string]bool),
	}
}

func (m *MessengerService) Export() error {
	if err := m.conn.Export(m, ObjectPath, InterfaceName); err != nil {
		return err
	}

	node := introspect.NewIntrospectable(&introspect.Node{
		Name: string(ObjectPath),
		Interfaces: []introspect.Interface{
			{
				Name: InterfaceName,
				Methods: []introspect.Method{
					{
						Name: "GetStatus",
						Args: []introspect.Arg{{Name: "status", Type: "s", Direction: "out"}},
					},
					{Name: "RequestLogin"},
					{Name: "Logout"},
					{
						Name: "SendTextMessage",
						Args: []introspect.Arg{
							{Name: "jid", Type: "s", Direction: "in"},
							{Name: "text", Type: "s", Direction: "in"},
							{Name: "message_id", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "GetChats",
						Args: []introspect.Arg{{Name: "chats", Type: "aa{sv}", Direction: "out"}},
					},
					{
						Name: "GetMessages",
						Args: []introspect.Arg{
							{Name: "jid", Type: "s", Direction: "in"},
							{Name: "limit", Type: "i", Direction: "in"},
							{Name: "offset", Type: "i", Direction: "in"},
							{Name: "messages", Type: "aa{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetGroupInfo",
						Args: []introspect.Arg{
							{Name: "jid", Type: "s", Direction: "in"},
							{Name: "info", Type: "a{sv}", Direction: "out"},
						},
					},
					{
						Name: "GetProfilePicture",
						Args: []introspect.Arg{
							{Name: "jid", Type: "s", Direction: "in"},
							{Name: "local_path", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "SearchMessages",
						Args: []introspect.Arg{
							{Name: "jid", Type: "s", Direction: "in"},
							{Name: "query", Type: "s", Direction: "in"},
							{Name: "messages", Type: "aa{sv}", Direction: "out"},
						},
					},
					{
						Name: "DownloadMedia",
						Args: []introspect.Arg{
							{Name: "message_id", Type: "s", Direction: "in"},
							{Name: "jid", Type: "s", Direction: "in"},
							{Name: "local_path", Type: "s", Direction: "out"},
						},
					},
					{
						Name: "MarkAsRead",
						Args: []introspect.Arg{{Name: "jid", Type: "s", Direction: "in"}},
					},
					{
						Name: "FetchLinkPreview",
						Args: []introspect.Arg{
							{Name: "url", Type: "s", Direction: "in"},
							{Name: "preview", Type: "a{sv}", Direction: "out"},
						},
					},
				},
				Signals: []introspect.Signal{
					{Name: "QRCodeUpdated", Args: []introspect.Arg{{Name: "qr_data", Type: "s"}}},
					{Name: "LoginSuccessful", Args: []introspect.Arg{{Name: "phone_number", Type: "s"}}},
					{
						Name: "MessageReceived",
						Args: []introspect.Arg{
							{Name: "jid", Type: "s"},
							{Name: "sender_name", Type: "s"},
							{Name: "text", Type: "s"},
							{Name: "timestamp_unix", Type: "x"},
						},
					},
					{Name: "ConnectionStatusChanged", Args: []introspect.Arg{{Name: "status", Type: "s"}}},
					{Name: "TypingStarted", Args: []introspect.Arg{{Name: "jid", Type: "s"}, {Name: "sender_jid", Type: "s"}}},
					{Name: "TypingStopped", Args: []introspect.Arg{{Name: "jid", Type: "s"}, {Name: "sender_jid", Type: "s"}}},
					{Name: "MessageAcknowledged", Args: []introspect.Arg{{Name: "message_id", Type: "s"}, {Name: "receipt_type", Type: "i"}}},
					{Name: "AvatarUpdated", Args: []introspect.Arg{{Name: "jid", Type: "s"}, {Name: "local_path", Type: "s"}}},
				},
			},
			introspect.IntrospectData,
		},
	})

	return m.conn.Export(node, ObjectPath, "org.freedesktop.DBus.Introspectable")
}

func (m *MessengerService) RunEventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-m.client.Events():
			if evt == nil {
				continue
			}
			switch event := evt.(type) {
			case *waevents.Message:
				text := extractDisplayText(event)
				if text != "" && !event.Info.IsFromMe {
					senderName := event.Info.PushName
					if senderName == "" {
						senderName = m.lookupContactName(event.Info.Sender.String(), event.Info.Chat.String())
					}
					_ = m.conn.Emit(ObjectPath, InterfaceName+".MessageReceived", event.Info.Chat.String(), senderName, text, event.Info.Timestamp.Unix())
				}
			case *waevents.Connected:
				m.setStatus("connected")
				_ = m.conn.Emit(ObjectPath, InterfaceName+".LoginSuccessful", m.client.PhoneNumber())
			case *waevents.Disconnected:
				m.setStatus("disconnected")
			case *waevents.LoggedOut:
				m.setStatus("disconnected")
			case *waevents.Receipt:
				receiptType := mapReceiptType(event.Type)
				if receiptType == 0 {
					continue
				}
				for _, messageID := range event.MessageIDs {
					_ = m.conn.Emit(ObjectPath, InterfaceName+".MessageAcknowledged", string(messageID), int32(receiptType))
				}
			case *waevents.ChatPresence:
				if event.State == types.ChatPresenceComposing {
					_ = m.conn.Emit(ObjectPath, InterfaceName+".TypingStarted", event.Chat.String(), event.Sender.String())
				} else if event.State == types.ChatPresencePaused {
					_ = m.conn.Emit(ObjectPath, InterfaceName+".TypingStopped", event.Chat.String(), event.Sender.String())
				}
			case *waevents.GroupInfo:
				m.invalidateGroupInfoCache(event.JID.String())
			case *waevents.Picture:
				m.invalidateAvatarCache(event.JID.String())
				m.scheduleAvatarFetch(event.JID.String())
			}
		}
	}
}

func (m *MessengerService) AutoConnect() error {
	if !m.client.HasSession() || m.client.IsConnected() {
		return nil
	}

	m.setStatus("connecting")
	if err := m.client.Connect(); err != nil {
		m.setStatus("disconnected")
		return err
	}

	return nil
}

func (m *MessengerService) Shutdown() {
	m.client.StopReconnect()
	m.client.Disconnect()
}

func (m *MessengerService) GetStatus() (string, *godbus.Error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status, nil
}

func (m *MessengerService) RequestLogin() *godbus.Error {
	status := m.currentStatus()
	if m.client.IsConnected() || status == "connecting" || status == "qr_pending" {
		return nil
	}

	if m.client.HasSession() {
		if err := m.AutoConnect(); err != nil {
			m.logger.Error("failed to auto-connect WhatsApp client", zap.Error(err))
			return godbus.MakeFailedError(err)
		}
		return nil
	}

	m.setStatus("connecting")

	ctx := context.Background()
	qrChan, err := m.client.GetQRChannel(ctx)
	if err != nil {
		m.logger.Error("failed to get QR channel", zap.Error(err))
		m.setStatus("disconnected")
		return godbus.MakeFailedError(err)
	}

	if err := m.client.Connect(); err != nil {
		m.logger.Error("failed to connect WhatsApp client", zap.Error(err))
		m.setStatus("disconnected")
		return godbus.MakeFailedError(err)
	}

	m.setStatus("qr_pending")

	go func() {
		for item := range qrChan {
			switch item.Event {
			case "code":
				m.client.RenderQRInTerminal(item.Code)
				encoded, err := generateQRCodeBase64(item.Code)
				if err != nil {
					m.logger.Warn("failed to encode QR PNG", zap.Error(err))
					continue
				}
				_ = m.conn.Emit(ObjectPath, InterfaceName+".QRCodeUpdated", encoded)
			case "success":
				m.setStatus("connected")
				_ = m.conn.Emit(ObjectPath, InterfaceName+".LoginSuccessful", m.client.PhoneNumber())
			case "timeout":
				m.setStatus("disconnected")
			}
		}
	}()

	return nil
}

func (m *MessengerService) Logout() *godbus.Error {
	m.client.StopReconnect()

	if m.client.IsConnected() {
		if err := m.client.Logout(context.Background()); err != nil {
			m.logger.Warn("failed to log out from WhatsApp", zap.Error(err))
		}
	}

	m.client.Disconnect()

	if device := m.client.DeviceStore(); device != nil {
		if err := device.Delete(context.Background()); err != nil {
			m.logger.Warn("failed to delete device store", zap.Error(err))
		}
	}

	if err := m.client.ResetSession(context.Background()); err != nil {
		m.logger.Warn("failed to reset client session", zap.Error(err))
	}

	m.setStatus("disconnected")
	return nil
}

func (m *MessengerService) SendTextMessage(jid string, text string) (string, *godbus.Error) {
	messageID, err := m.client.SendText(jid, text)
	if err != nil {
		return "", godbus.MakeFailedError(err)
	}

	if err := m.messageStore.Save(storage.StoredMessage{
		ID:         messageID,
		JID:        jid,
		FromMe:     true,
		Text:       text,
		Timestamp:  nowUnix(),
		IsRead:     true,
		SenderJID:  jid,
		SenderName: "",
		IsGroup:    strings.HasSuffix(jid, "@g.us"),
		Receipt:    0,
	}); err != nil {
		m.logger.Warn("failed to persist outgoing message", zap.Error(err))
	}

	return messageID, nil
}

func (m *MessengerService) GetChats() ([]map[string]godbus.Variant, *godbus.Error) {
	contacts, err := m.client.ContactNames(context.Background())
	if err != nil {
		m.logger.Warn("failed to load contacts", zap.Error(err))
		return []map[string]godbus.Variant{}, nil
	}

	recentMessages, err := m.messageStore.GetRecentChats()
	if err != nil {
		m.logger.Warn("failed to load recent chats", zap.Error(err))
		return []map[string]godbus.Variant{}, nil
	}

	chats := make([]map[string]godbus.Variant, 0, len(recentMessages))
	for _, message := range recentMessages {
		name := message.JID
		if message.IsGroup {
			info, err := m.loadGroupInfo(message.JID)
			if err == nil {
				if cachedName := variantString(info, "name"); cachedName != "" {
					name = cachedName
				}
			}
		} else if jid, err := types.ParseJID(message.JID); err == nil {
			name = contactName(contacts[jid], message.JID)
		}

		unreadCount, err := m.messageStore.GetUnreadCount(message.JID)
		if err != nil {
			m.logger.Warn("failed to load unread count", zap.Error(err), zap.String("jid", message.JID))
			unreadCount = 0
		}

		avatarPath, _ := m.GetProfilePicture(message.JID)

		chats = append(chats, map[string]godbus.Variant{
			"jid":          godbus.MakeVariant(message.JID),
			"name":         godbus.MakeVariant(name),
			"last_message": godbus.MakeVariant(message.Text),
			"timestamp":    godbus.MakeVariant(message.Timestamp),
			"unread":       godbus.MakeVariant(int32(unreadCount)),
			"is_group":     godbus.MakeVariant(message.IsGroup),
			"avatar_path":  godbus.MakeVariant(avatarPath),
		})
	}

	return chats, nil
}

func (m *MessengerService) GetMessages(jid string, limit int32, offset int32) ([]map[string]godbus.Variant, *godbus.Error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	messages, err := m.messageStore.GetByJID(jid, int(limit), int(offset))
	if err != nil {
		m.logger.Warn("failed to load messages", zap.Error(err), zap.String("jid", jid))
		return []map[string]godbus.Variant{}, nil
	}

	return m.messageVariants(messages), nil
}

func (m *MessengerService) GetGroupInfo(jid string) (map[string]godbus.Variant, *godbus.Error) {
	info, err := m.loadGroupInfo(jid)
	if err != nil {
		return map[string]godbus.Variant{}, nil
	}
	return info, nil
}

func (m *MessengerService) GetProfilePicture(jid string) (string, *godbus.Error) {
	path, err := avatarCachePath(jid)
	if err != nil {
		return "", nil
	}

	info, err := os.Stat(path)
	if err == nil {
		if time.Since(info.ModTime()) < 24*time.Hour {
			return path, nil
		}
		m.scheduleAvatarFetch(jid)
		return path, nil
	}

	m.scheduleAvatarFetch(jid)
	return "", nil
}

func (m *MessengerService) SearchMessages(jid string, query string) ([]map[string]godbus.Variant, *godbus.Error) {
	if strings.TrimSpace(query) == "" {
		return []map[string]godbus.Variant{}, nil
	}

	messages, err := m.messageStore.SearchByJID(jid, strings.TrimSpace(query), 50)
	if err != nil {
		m.logger.Warn("failed to search messages", zap.Error(err), zap.String("jid", jid))
		return []map[string]godbus.Variant{}, nil
	}

	return m.messageVariants(messages), nil
}

func (m *MessengerService) DownloadMedia(messageID string, jid string) (string, *godbus.Error) {
	message, err := m.messageStore.GetByID(messageID)
	if err != nil {
		m.logger.Warn("failed to load message for media download", zap.Error(err), zap.String("message_id", messageID))
		return "", nil
	}
	if !message.HasMedia {
		return "", nil
	}
	if message.LocalPath != "" {
		return message.LocalPath, nil
	}

	mediaDir, err := mediaCacheDir(jid)
	if err != nil {
		m.logger.Warn("failed to resolve media cache path", zap.Error(err))
		return "", nil
	}
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		m.logger.Warn("failed to create media cache dir", zap.Error(err))
		return "", nil
	}

	data, err := m.client.DownloadStoredMedia(context.Background(), *message)
	if err != nil {
		m.logger.Warn("failed to download media", zap.Error(err), zap.String("message_id", messageID))
		return "", nil
	}

	filePath := filepath.Join(mediaDir, protocol.MediaFilename(*message))
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		m.logger.Warn("failed to write downloaded media", zap.Error(err), zap.String("path", filePath))
		return "", nil
	}

	if err := m.messageStore.UpdateLocalPath(messageID, filePath); err != nil {
		m.logger.Warn("failed to persist media local path", zap.Error(err), zap.String("message_id", messageID))
	}

	return filePath, nil
}

func (m *MessengerService) MarkAsRead(jid string) *godbus.Error {
	if err := m.messageStore.MarkJIDAsRead(jid); err != nil {
		m.logger.Warn("failed to mark chat as read", zap.Error(err), zap.String("jid", jid))
	}
	return nil
}

func (m *MessengerService) FetchLinkPreview(url string) (map[string]godbus.Variant, *godbus.Error) {
	preview := protocol.FetchLinkPreview(url)
	if preview.Title == "" && preview.Description == "" && preview.ImageURL == "" {
		return map[string]godbus.Variant{}, nil
	}

	return map[string]godbus.Variant{
		"title":       godbus.MakeVariant(preview.Title),
		"description": godbus.MakeVariant(preview.Description),
		"image_url":   godbus.MakeVariant(preview.ImageURL),
	}, nil
}

func (m *MessengerService) loadGroupInfo(jid string) (map[string]godbus.Variant, error) {
	m.cacheMu.Lock()
	if cached, ok := m.groupInfoCache[jid]; ok && time.Now().Before(cached.expiresAt) {
		m.cacheMu.Unlock()
		return cached.info, nil
	}
	m.cacheMu.Unlock()

	groupInfo, err := m.client.GetGroupInfo(context.Background(), jid)
	if err != nil {
		return nil, err
	}

	participants := make([]string, 0, len(groupInfo.Participants))
	for _, participant := range groupInfo.Participants {
		participants = append(participants, participant.JID.String())
	}

	info := map[string]godbus.Variant{
		"name":              godbus.MakeVariant(groupInfo.Name),
		"description":       godbus.MakeVariant(groupInfo.Topic),
		"participant_count": godbus.MakeVariant(int32(groupInfo.ParticipantCount)),
		"participants":      godbus.MakeVariant(participants),
	}

	m.cacheMu.Lock()
	m.groupInfoCache[jid] = cachedGroupInfo{
		info:      info,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	m.cacheMu.Unlock()

	return info, nil
}

func (m *MessengerService) invalidateGroupInfoCache(jid string) {
	m.cacheMu.Lock()
	delete(m.groupInfoCache, jid)
	m.cacheMu.Unlock()
}

func (m *MessengerService) invalidateAvatarCache(jid string) {
	path, err := avatarCachePath(jid)
	if err == nil {
		_ = os.Remove(path)
	}
}

func (m *MessengerService) scheduleAvatarFetch(jid string) {
	m.cacheMu.Lock()
	if m.avatarDownloads[jid] {
		m.cacheMu.Unlock()
		return
	}
	m.avatarDownloads[jid] = true
	m.cacheMu.Unlock()

	go func() {
		defer func() {
			m.cacheMu.Lock()
			delete(m.avatarDownloads, jid)
			m.cacheMu.Unlock()
		}()

		path, err := avatarCachePath(jid)
		if err != nil {
			return
		}

		info, err := m.client.GetProfilePictureInfo(context.Background(), jid)
		if err != nil || info == nil || info.URL == "" {
			return
		}

		data, err := m.client.DownloadFile(context.Background(), info.URL)
		if err != nil {
			m.logger.Warn("failed to download profile picture", zap.Error(err), zap.String("jid", jid))
			return
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return
		}

		_ = m.conn.Emit(ObjectPath, InterfaceName+".AvatarUpdated", jid, path)
	}()
}

func (m *MessengerService) messageVariants(messages []storage.StoredMessage) []map[string]godbus.Variant {
	items := make([]map[string]godbus.Variant, 0, len(messages))
	for _, message := range messages {
		item := map[string]godbus.Variant{
			"id":          godbus.MakeVariant(message.ID),
			"from_me":     godbus.MakeVariant(message.FromMe),
			"text":        godbus.MakeVariant(message.Text),
			"timestamp":   godbus.MakeVariant(message.Timestamp),
			"sender_jid":  godbus.MakeVariant(message.SenderJID),
			"sender_name": godbus.MakeVariant(message.SenderName),
			"receipt":     godbus.MakeVariant(int32(message.Receipt)),
		}
		if message.HasMedia {
			item["has_media"] = godbus.MakeVariant(message.HasMedia)
			item["media_type"] = godbus.MakeVariant(message.MediaType)
			item["media_mime"] = godbus.MakeVariant(message.MediaMIME)
			item["media_size"] = godbus.MakeVariant(message.MediaSize)
			item["local_path"] = godbus.MakeVariant(message.LocalPath)
		}
		if message.URLPreview != "" {
			item["url_preview"] = godbus.MakeVariant(message.URLPreview)
		}
		items = append(items, item)
	}
	return items
}

func (m *MessengerService) lookupContactName(senderJID string, fallback string) string {
	contacts, err := m.client.ContactNames(context.Background())
	if err != nil {
		return fallback
	}

	jid, err := types.ParseJID(senderJID)
	if err != nil {
		return fallback
	}

	return contactName(contacts[jid], fallback)
}

func (m *MessengerService) setStatus(status string) {
	m.mu.Lock()
	changed := m.status != status
	if changed {
		m.status = status
	}
	m.mu.Unlock()

	if !changed {
		return
	}

	if err := m.conn.Emit(ObjectPath, InterfaceName+".ConnectionStatusChanged", status); err != nil {
		m.logger.Warn("failed to emit status change signal", zap.Error(err))
	}
}

func (m *MessengerService) currentStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func generateQRCodeBase64(qrString string) (string, error) {
	pngBytes, err := qrcode.Encode(qrString, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pngBytes), nil
}

func extractDisplayText(event *waevents.Message) string {
	if event == nil {
		return ""
	}
	if text := event.Message.GetConversation(); text != "" {
		return text
	}
	if text := event.Message.GetExtendedTextMessage().GetText(); text != "" {
		return text
	}
	return ""
}

func contactName(info types.ContactInfo, fallbackJID string) string {
	switch {
	case info.BusinessName != "":
		return info.BusinessName
	case info.FullName != "":
		return info.FullName
	case info.FirstName != "":
		return info.FirstName
	case info.PushName != "":
		return info.PushName
	default:
		return fallbackJID
	}
}

func nowUnix() int64 {
	return time.Now().Unix()
}

func mediaCacheDir(jid string) (string, error) {
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "khaapp", "media", jid), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".cache", "khaapp", "media", jid), nil
}

func avatarCachePath(jid string) (string, error) {
	sanitized := strings.NewReplacer("@", "_", ".", "_", "/", "_").Replace(jid)
	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "khaapp", "avatars", sanitized+".png"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".cache", "khaapp", "avatars", sanitized+".png"), nil
}

func mapReceiptType(receiptType types.ReceiptType) int {
	switch receiptType {
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf, types.ReceiptTypePlayed, types.ReceiptTypePlayedSelf:
		return 2
	case types.ReceiptTypeDelivered:
		return 1
	default:
		return 0
	}
}

func variantString(values map[string]godbus.Variant, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	text, _ := value.Value().(string)
	return text
}
