package app

import "github.com/hossein-repo/BaseProject/internal/messaging"

type Application struct {
	Consumer messaging.Consumer
	Repo     interface {
		Save(msg messaging.Message)
	}
}