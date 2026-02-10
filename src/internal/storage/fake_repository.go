package storage

import (
	"log"

	"github.com/hossein-repo/BaseProject/internal/messaging"
)

type FakeRepository struct {
	data map[string]messaging.NotamMessage
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		data: make(map[string]messaging.NotamMessage),
	}
}

func (r *FakeRepository) Save(msg messaging.Message) {
	notamMsg, ok := msg.(messaging.NotamMessage)
	if !ok {
		log.Println("❌ Invalid message type for save")
		return
	}

	r.data[notamMsg.ID()] = notamMsg
	log.Printf("💾 Saved NOTAM ID: %s | Location: %s | Text: %s", notamMsg.ID(), notamMsg.Event().ICAOLocation, notamMsg.Event().HumanReadableText)
}