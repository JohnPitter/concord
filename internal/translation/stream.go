package translation

import (
	"context"
	"sync"

	"github.com/concord-chat/concord/internal/config"
	"github.com/rs/zerolog"
)

// Pipeline connects the voice engine audio stream to PersonaPlex for real-time translation.
// It reads audio frames from the voice engine, sends them to PersonaPlex, and injects the
// translated result back into the output. On failure, it gracefully degrades by passing
// the original audio through (never blocks voice).
type Pipeline struct {
	mu     sync.RWMutex
	client *Client
	cfg    config.TranslationConfig
	logger zerolog.Logger
	cancel context.CancelFunc
	active bool
}

// NewPipeline creates a new streaming translation pipeline.
// Complexity: O(1)
func NewPipeline(client *Client, cfg config.TranslationConfig, logger zerolog.Logger) *Pipeline {
	return &Pipeline{
		client: client,
		cfg:    cfg,
		logger: logger.With().Str("component", "translation-pipeline").Logger(),
	}
}

// Start begins the streaming translation pipeline.
// It reads audio frames from audioIn, translates them via PersonaPlex, and writes
// translated frames to the returned channel. If translation fails at any point,
// original audio is passed through (graceful degradation).
// Complexity: O(n) where n = number of audio frames processed
func (p *Pipeline) Start(ctx context.Context, audioIn <-chan []byte, sourceLang, targetLang string) (<-chan []byte, error) {
	p.mu.Lock()
	if p.active {
		p.mu.Unlock()
		return nil, ErrPipelineAlreadyActive
	}
	p.active = true
	p.mu.Unlock()

	pipeCtx, cancel := context.WithCancel(ctx)
	p.mu.Lock()
	p.cancel = cancel
	p.mu.Unlock()

	out := make(chan []byte, 64)

	// Try to start the streaming translation
	translatedCh, err := p.client.TranslateStream(pipeCtx, audioIn, sourceLang, targetLang)
	if err != nil {
		// Graceful degradation: pass-through original audio
		p.logger.Warn().Err(err).Msg("translation stream failed to start, using pass-through")
		go p.passthrough(pipeCtx, audioIn, out)
		return out, nil
	}

	// Bridge goroutine: read translated audio and forward to output.
	// If the translated channel closes unexpectedly, fall back to pass-through.
	go func() {
		defer close(out)
		defer func() {
			p.mu.Lock()
			p.active = false
			p.mu.Unlock()
		}()

		for {
			select {
			case <-pipeCtx.Done():
				return
			case frame, ok := <-translatedCh:
				if !ok {
					// Translation stream ended — fall back to pass-through
					p.logger.Warn().Msg("translated stream closed, falling back to pass-through")
					p.drainPassthrough(pipeCtx, audioIn, out)
					return
				}
				select {
				case out <- frame:
				case <-pipeCtx.Done():
					return
				}
			}
		}
	}()

	p.logger.Info().
		Str("source_lang", sourceLang).
		Str("target_lang", targetLang).
		Msg("translation pipeline started")

	return out, nil
}

// Stop gracefully stops the translation pipeline.
// Complexity: O(1)
func (p *Pipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.active = false
	p.logger.Info().Msg("translation pipeline stopped")
}

// IsActive returns whether the pipeline is currently running.
// Complexity: O(1)
func (p *Pipeline) IsActive() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.active
}

// passthrough forwards original audio frames directly to output when translation is unavailable.
func (p *Pipeline) passthrough(ctx context.Context, audioIn <-chan []byte, out chan<- []byte) {
	defer close(out)
	defer func() {
		p.mu.Lock()
		p.active = false
		p.mu.Unlock()
	}()

	p.drainPassthrough(ctx, audioIn, out)
}

// drainPassthrough reads from audioIn and writes to out until ctx is done or audioIn closes.
func (p *Pipeline) drainPassthrough(ctx context.Context, audioIn <-chan []byte, out chan<- []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-audioIn:
			if !ok {
				return
			}
			select {
			case out <- frame:
			case <-ctx.Done():
				return
			}
		}
	}
}
