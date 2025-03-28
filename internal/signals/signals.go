package signals

import (
	"context"

	"github.com/maniartech/signals"
)

// TokenSetupData contains data associated with token setup signal
type TokenSetupData struct {
	// You can add additional fields here if needed
	Success bool
}

// Signal definitions using generics
var TokenSetup = signals.New[TokenSetupData]()

// EmitTokenSetup emits a signal when a token is successfully set up
func EmitTokenSetup(ctx context.Context, success bool) {
	TokenSetup.Emit(ctx, TokenSetupData{
		Success: success,
	})
}

// OnTokenSetup registers a handler for token setup events
func OnTokenSetup(handler func(ctx context.Context, data TokenSetupData), key ...string) {
	if len(key) > 0 {
		TokenSetup.AddListener(handler, key[0])
	} else {
		TokenSetup.AddListener(handler)
	}
}
