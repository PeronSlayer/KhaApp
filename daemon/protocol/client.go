package protocol

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/khaapp/khaapp-daemon/storage"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waevents "go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// WAClient wraps the whatsmeow client and forwards selected events to an internal channel.
type WAClient struct {
	client       *whatsmeow.Client
	container    *sqlstore.Container
	messageStore *storage.MessageStore
	eventsCh     chan interface{}
	logger       *zap.Logger
	reconnectMu  sync.Mutex
	reconnectCtx context.Context
	cancelRetry  context.CancelFunc
}

// NewWAClient creates a new whatsmeow client using the first available device store.
func NewWAClient(container *sqlstore.Container, messageStore *storage.MessageStore, logger *zap.Logger) (*WAClient, error) {
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, err
	}

	wrapped := &WAClient{
		container:    container,
		messageStore: messageStore,
		eventsCh:     make(chan interface{}, 32),
		logger:       logger,
	}

	wrapped.replaceClient(deviceStore)
	return wrapped, nil
}

func (w *WAClient) replaceClient(deviceStore *store.Device) {
	w.client = whatsmeow.NewClient(deviceStore, nil)
	w.client.EnableAutoReconnect = false
	w.client.AddEventHandler(func(evt interface{}) {
		switch event := evt.(type) {
		case *waevents.Message:
			storedMessage := w.buildStoredMessage(event)
			if storedMessage.Text != "" || storedMessage.HasMedia {
				if err := w.messageStore.Save(storedMessage); err != nil {
					w.logger.Warn("failed to persist incoming message", zap.Error(err))
				}
			}

			select {
			case w.eventsCh <- event:
			default:
				w.logger.Warn("dropping message event because channel is full")
			}
		case *waevents.Connected:
			w.StopReconnect()
			if err := w.client.SendPresence(context.Background(), types.PresenceAvailable); err != nil {
				w.logger.Warn("failed to announce online presence", zap.Error(err))
			}
			select {
			case w.eventsCh <- event:
			default:
			}
		case *waevents.Disconnected:
			w.startReconnectLoop()
			select {
			case w.eventsCh <- event:
			default:
			}
		case *waevents.LoggedOut:
			w.StopReconnect()
			select {
			case w.eventsCh <- event:
			default:
			}
		case *waevents.Receipt:
			receipt := mapReceiptType(event.Type)
			if receipt > 0 {
				for _, messageID := range event.MessageIDs {
					if err := w.messageStore.UpdateReceipt(string(messageID), receipt); err != nil {
						w.logger.Warn("failed to persist receipt", zap.Error(err), zap.String("message_id", string(messageID)))
					}
				}
			}
			select {
			case w.eventsCh <- event:
			default:
			}
		case *waevents.ChatPresence, *waevents.GroupInfo, *waevents.Picture:
			select {
			case w.eventsCh <- event:
			default:
			}
		}
	})
}

// Events returns the internal event channel used by the D-Bus service.
func (w *WAClient) Events() <-chan interface{} {
	return w.eventsCh
}

// Connect connects the client to WhatsApp.
func (w *WAClient) Connect() error {
	return w.client.Connect()
}

// Disconnect disconnects the client from WhatsApp.
func (w *WAClient) Disconnect() {
	w.StopReconnect()
	w.client.Disconnect()
}

// Logout requests a remote logout when possible.
func (w *WAClient) Logout(ctx context.Context) error {
	return w.client.Logout(ctx)
}

// GetQRChannel returns the QR channel used during login.
func (w *WAClient) GetQRChannel(ctx context.Context) (<-chan whatsmeow.QRChannelItem, error) {
	return w.client.GetQRChannel(ctx)
}

// SendText sends a text message and returns the WhatsApp message ID.
func (w *WAClient) SendText(jid string, text string) (string, error) {
	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return "", err
	}

	response, err := w.client.SendMessage(context.Background(), parsedJID, &waProto.Message{
		Conversation: &text,
	})
	if err != nil {
		return "", err
	}

	return string(response.ID), nil
}

// IsConnected returns whether the websocket is currently connected.
func (w *WAClient) IsConnected() bool {
	return w.client != nil && w.client.IsConnected()
}

// HasSession returns whether the current device store already contains a paired session.
func (w *WAClient) HasSession() bool {
	return w.client != nil && w.client.Store != nil && w.client.Store.ID != nil
}

// DeviceStore returns the underlying device store.
func (w *WAClient) DeviceStore() *store.Device {
	return w.client.Store
}

// RenderQRInTerminal prints the current QR string to the terminal for debugging.
func (w *WAClient) RenderQRInTerminal(qr string) {
	qrterminal.GenerateHalfBlock(qr, qrterminal.L, os.Stdout)
}

// PhoneNumber returns the currently connected phone number, if known.
func (w *WAClient) PhoneNumber() string {
	if w.client.Store == nil || w.client.Store.ID == nil {
		return ""
	}

	return fmt.Sprintf("%s", w.client.Store.ID.User)
}

// ContactNames returns all locally cached contacts.
func (w *WAClient) ContactNames(ctx context.Context) (map[types.JID]types.ContactInfo, error) {
	if w.client == nil || w.client.Store == nil || w.client.Store.Contacts == nil {
		return map[types.JID]types.ContactInfo{}, nil
	}

	return w.client.Store.Contacts.GetAllContacts(ctx)
}

// ResetSession recreates the underlying whatsmeow client with a fresh device store.
func (w *WAClient) ResetSession(ctx context.Context) error {
	w.StopReconnect()
	deviceStore, err := w.container.GetFirstDevice(ctx)
	if err != nil {
		return err
	}

	w.replaceClient(deviceStore)
	return nil
}

// DownloadStoredMedia downloads one previously stored media attachment.
func (w *WAClient) DownloadStoredMedia(ctx context.Context, message storage.StoredMessage) ([]byte, error) {
	return w.client.DownloadMediaWithPath(
		ctx,
		message.DirectPath,
		message.FileEncSHA256,
		message.FileSHA256,
		message.MediaKey,
		mapMediaType(message.MediaType),
		mapMMSType(message.MediaType),
		false,
	)
}

// MediaFilename returns a filesystem-safe filename for one stored media attachment.
func MediaFilename(message storage.StoredMessage) string {
	extension := ".bin"
	if message.MediaMIME != "" {
		if extensions, _ := mime.ExtensionsByType(message.MediaMIME); len(extensions) > 0 {
			extension = extensions[0]
		}
	}
	return message.ID + extension
}

func (w *WAClient) GetGroupInfo(ctx context.Context, jid string) (*types.GroupInfo, error) {
	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return nil, err
	}

	return w.client.GetGroupInfo(ctx, parsedJID)
}

func (w *WAClient) GetProfilePictureInfo(ctx context.Context, jid string) (*types.ProfilePictureInfo, error) {
	parsedJID, err := types.ParseJID(jid)
	if err != nil {
		return nil, err
	}

	params := &whatsmeow.GetProfilePictureParams{}
	if strings.HasSuffix(jid, "@g.us") {
		params.IsCommunity = false
	}

	return w.client.GetProfilePictureInfo(ctx, parsedJID, params)
}

func (w *WAClient) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}

	return io.ReadAll(response.Body)
}

func (w *WAClient) StopReconnect() {
	w.reconnectMu.Lock()
	defer w.reconnectMu.Unlock()
	if w.cancelRetry != nil {
		w.cancelRetry()
		w.cancelRetry = nil
		w.reconnectCtx = nil
	}
}

func (w *WAClient) startReconnectLoop() {
	if !w.HasSession() {
		return
	}

	w.reconnectMu.Lock()
	if w.cancelRetry != nil {
		w.reconnectMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.reconnectCtx = ctx
	w.cancelRetry = cancel
	w.reconnectMu.Unlock()

	go func() {
		defer w.StopReconnect()

		delay := 2 * time.Second
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			if w.IsConnected() {
				return
			}

			if err := w.client.Connect(); err != nil {
				w.logger.Warn("failed to reconnect WhatsApp client", zap.Error(err), zap.Duration("retry_in", delay))
				delay *= 2
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
				continue
			}
			return
		}
	}()
}

func (w *WAClient) buildStoredMessage(event *waevents.Message) storage.StoredMessage {
	messageText := extractMessageText(event.Message)
	senderJID := event.Info.Chat.String()
	if !event.Info.Sender.IsEmpty() {
		senderJID = event.Info.Sender.String()
	}
	storedMessage := storage.StoredMessage{
		ID:        string(event.Info.ID),
		JID:       event.Info.Chat.String(),
		FromMe:    event.Info.IsFromMe,
		Text:      messageText,
		Timestamp: event.Info.Timestamp.Unix(),
		IsRead:    event.Info.IsFromMe,
		SenderJID: senderJID,
		SenderName: w.lookupContactName(senderJID, event.Info.PushName),
		IsGroup:   strings.HasSuffix(event.Info.Chat.String(), "@g.us"),
		Receipt:   0,
	}

	switch {
	case event.Message.GetImageMessage() != nil:
		media := event.Message.GetImageMessage()
		storedMessage.HasMedia = true
		storedMessage.MediaType = "image"
		storedMessage.MediaMIME = media.GetMimetype()
		storedMessage.MediaSize = int64(media.GetFileLength())
		storedMessage.MediaKey = media.GetMediaKey()
		storedMessage.DirectPath = media.GetDirectPath()
		storedMessage.MediaURL = media.GetURL()
		storedMessage.FileEncSHA256 = media.GetFileEncSHA256()
		storedMessage.FileSHA256 = media.GetFileSHA256()
		if storedMessage.Text == "" {
			storedMessage.Text = media.GetCaption()
		}
	case event.Message.GetAudioMessage() != nil:
		media := event.Message.GetAudioMessage()
		storedMessage.HasMedia = true
		storedMessage.MediaType = "audio"
		storedMessage.MediaMIME = media.GetMimetype()
		storedMessage.MediaSize = int64(media.GetFileLength())
		storedMessage.MediaKey = media.GetMediaKey()
		storedMessage.DirectPath = media.GetDirectPath()
		storedMessage.MediaURL = media.GetURL()
		storedMessage.FileEncSHA256 = media.GetFileEncSHA256()
		storedMessage.FileSHA256 = media.GetFileSHA256()
	case event.Message.GetDocumentMessage() != nil:
		media := event.Message.GetDocumentMessage()
		storedMessage.HasMedia = true
		storedMessage.MediaType = "document"
		storedMessage.MediaMIME = media.GetMimetype()
		storedMessage.MediaSize = int64(media.GetFileLength())
		storedMessage.MediaKey = media.GetMediaKey()
		storedMessage.DirectPath = media.GetDirectPath()
		storedMessage.MediaURL = media.GetURL()
		storedMessage.FileEncSHA256 = media.GetFileEncSHA256()
		storedMessage.FileSHA256 = media.GetFileSHA256()
		if storedMessage.Text == "" {
			storedMessage.Text = media.GetCaption()
		}
	case event.Message.GetVideoMessage() != nil:
		media := event.Message.GetVideoMessage()
		storedMessage.HasMedia = true
		storedMessage.MediaType = "video"
		storedMessage.MediaMIME = media.GetMimetype()
		storedMessage.MediaSize = int64(media.GetFileLength())
		storedMessage.MediaKey = media.GetMediaKey()
		storedMessage.DirectPath = media.GetDirectPath()
		storedMessage.MediaURL = media.GetURL()
		storedMessage.FileEncSHA256 = media.GetFileEncSHA256()
		storedMessage.FileSHA256 = media.GetFileSHA256()
		if storedMessage.Text == "" {
			storedMessage.Text = media.GetCaption()
		}
	case event.Message.GetStickerMessage() != nil:
		media := event.Message.GetStickerMessage()
		storedMessage.HasMedia = true
		storedMessage.MediaType = "sticker"
		storedMessage.MediaMIME = media.GetMimetype()
		storedMessage.MediaSize = int64(media.GetFileLength())
		storedMessage.MediaKey = media.GetMediaKey()
		storedMessage.DirectPath = media.GetDirectPath()
		storedMessage.MediaURL = media.GetURL()
		storedMessage.FileEncSHA256 = media.GetFileEncSHA256()
		storedMessage.FileSHA256 = media.GetFileSHA256()
	}

	return storedMessage
}

func (w *WAClient) lookupContactName(senderJID string, pushName string) string {
	if pushName != "" {
		return pushName
	}

	if w.client == nil || w.client.Store == nil || w.client.Store.Contacts == nil {
		return senderJID
	}

	parsedJID, err := types.ParseJID(senderJID)
	if err != nil {
		return senderJID
	}

	info, err := w.client.Store.Contacts.GetContact(context.Background(), parsedJID)
	if err != nil {
		return senderJID
	}

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
		return senderJID
	}
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

func extractMessageText(message *waProto.Message) string {
	if message == nil {
		return ""
	}

	if text := message.GetConversation(); text != "" {
		return text
	}

	if text := message.GetExtendedTextMessage().GetText(); text != "" {
		return text
	}

	return ""
}

func mapMediaType(mediaType string) whatsmeow.MediaType {
	switch mediaType {
	case "image":
		return whatsmeow.MediaImage
	case "audio":
		return whatsmeow.MediaAudio
	case "document":
		return whatsmeow.MediaDocument
	case "video":
		return whatsmeow.MediaVideo
	case "sticker":
		return whatsmeow.MediaImage
	default:
		return ""
	}
}

func mapMMSType(mediaType string) string {
	switch mediaType {
	case "image":
		return "image"
	case "audio":
		return "audio"
	case "document":
		return "document"
	case "video":
		return "video"
	case "sticker":
		return "image"
	default:
		return strings.TrimPrefix(filepath.Ext(mediaType), ".")
	}
}
