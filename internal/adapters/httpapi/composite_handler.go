package httpapi

// CompositeHandler composes every per-resource handler into the single
// StrictServerInterface value cmd/api's composition root wires up - each
// handler still only knows about its own operations.
type CompositeHandler struct {
	*GreetingHandler
	*ConversationHandler
}

func NewCompositeHandler(greeting *GreetingHandler, conversation *ConversationHandler) *CompositeHandler {
	return &CompositeHandler{GreetingHandler: greeting, ConversationHandler: conversation}
}
