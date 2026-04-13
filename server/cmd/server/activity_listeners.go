package main

import (
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/handler"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func registerActivityListeners(bus *events.Bus, queries *db.Queries) {
	handler.RegisterActivityListeners(bus, queries)
}
