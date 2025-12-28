package nelchanbot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// MessageCreateHandler handles new messages from Discord
type MessageCreateHandler func(s *discordgo.Session, m *discordgo.MessageCreate)

// MessageUpdateHandler handles message updates from Discord
type MessageUpdateHandler func(s *discordgo.Session, m *discordgo.MessageUpdate)

// MessageDeleteHandler handles message deletions from Discord
type MessageDeleteHandler func(s *discordgo.Session, m *discordgo.MessageDelete)

// handleMessageCreate stores a new message in the database
func (n *Nelchan) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author == nil || m.Author.ID == s.State.User.ID {
		return
	}

	// Build mention user IDs
	mentionUserIDs := make([]string, 0, len(m.Mentions))
	for _, user := range m.Mentions {
		mentionUserIDs = append(mentionUserIDs, user.ID)
	}

	// Get reference message ID if this is a reply
	var referenceMessageID *string
	if m.MessageReference != nil && m.MessageReference.MessageID != "" {
		referenceMessageID = &m.MessageReference.MessageID
	}

	// Get display name
	var displayName *string
	if m.Author.GlobalName != "" {
		displayName = &m.Author.GlobalName
	}

	// Format timestamp
	timestamp := m.Timestamp.Format(time.RFC3339)

	request := StoreMessageAPIRequest{
		ID:                 m.ID,
		ChannelID:          m.ChannelID,
		UserID:             m.Author.ID,
		Content:            m.Content,
		Timestamp:          timestamp,
		ReferenceMessageID: referenceMessageID,
		MentionUserIDs:     mentionUserIDs,
		MentionRoleIDs:     m.MentionRoles,
		HasAttachments:     len(m.Attachments) > 0,
		Username:           m.Author.Username,
		DisplayName:        displayName,
	}

	// Store message asynchronously
	go func() {
		result, err := n.CommandAPIClient.StoreMessage(request)
		if err != nil {
			fmt.Printf("[messageHandler] error storing message %s: %v\n", m.ID, err)
			return
		}

		if result != nil && result.Stored {
			fmt.Printf("[messageHandler] stored message %s (vectorized: %v)\n", m.ID, result.Vectorized)
		}
	}()
}

// handleMessageUpdate updates a message in the database
func (n *Nelchan) handleMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) {
	// Ignore if no author (system messages, etc.)
	if m.Author == nil {
		return
	}

	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Get edited timestamp
	var editedTimestamp string
	if m.EditedTimestamp != nil {
		editedTimestamp = m.EditedTimestamp.Format(time.RFC3339)
	} else {
		editedTimestamp = time.Now().UTC().Format(time.RFC3339)
	}

	request := UpdateMessageAPIRequest{
		ID:              m.ID,
		Content:         m.Content,
		EditedTimestamp: editedTimestamp,
	}

	// Update message asynchronously
	go func() {
		result, err := n.CommandAPIClient.UpdateMessage(request)
		if err != nil {
			fmt.Printf("[messageHandler] error updating message %s: %v\n", m.ID, err)
			return
		}

		if result != nil && result.Stored {
			fmt.Printf("[messageHandler] updated message %s (vectorized: %v)\n", m.ID, result.Vectorized)
		}
	}()
}

// handleMessageDelete removes a message from the database
func (n *Nelchan) handleMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	request := DeleteMessageAPIRequest{
		ID: m.ID,
	}

	// Delete message asynchronously
	go func() {
		result, err := n.CommandAPIClient.DeleteMessage(request)
		if err != nil {
			fmt.Printf("[messageHandler] error deleting message %s: %v\n", m.ID, err)
			return
		}

		if result != nil && result.Success {
			fmt.Printf("[messageHandler] deleted message %s\n", m.ID)
		}
	}()
}
