package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// EventType represents the type of pipeline event.
type EventType string

const (
	EventPipelineStart   EventType = "pipeline_start"
	EventPipelineEnd     EventType = "pipeline_end"
	EventTreeSnapshot    EventType = "tree_snapshot"
	EventTreeNodeAdded   EventType = "tree_node_added"
	EventNodeAnalyzing   EventType = "node_analyzing"
	EventNodeAnalyzed    EventType = "node_analyzed"
	EventNodeFailed      EventType = "node_failed"
	EventGraphUpdating   EventType = "graph_updating"
	EventGraphUpdated    EventType = "graph_updated"
	EventScenePlanning   EventType = "scene_planning"
	EventScenePlanned    EventType = "scene_planned"
	EventSceneWriting    EventType = "scene_writing"
	EventSceneWritten    EventType = "scene_written"
	EventEditorReviewing EventType = "editor_reviewing"
	EventEditorReviewed  EventType = "editor_reviewed"
	EventValidation      EventType = "validation"
	EventError           EventType = "error"
)

// PipelineEvent represents a single event in the conversion pipeline.
type PipelineEvent struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Step      string                 `json:"step,omitempty"`
	NodeID    string                 `json:"node_id,omitempty"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  *time.Duration         `json:"duration,omitempty"`
	Error     string                 `json:"error,omitempty"`
	ProjectID string                 `json:"projectId,omitempty"`
	JobID     string                 `json:"jobId,omitempty"`
	RunID     string                 `json:"runId,omitempty"`
	TraceID   string                 `json:"traceId,omitempty"`
	Progress  *int                   `json:"progress,omitempty"`
}

// EventBus manages SSE event distribution to connected clients.
type EventBus struct {
	mu         sync.RWMutex
	clients    map[chan PipelineEvent]struct{}
	eventLog   []PipelineEvent
	maxLogSize int
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		clients:    make(map[chan PipelineEvent]struct{}),
		eventLog:   make([]PipelineEvent, 0, 100),
		maxLogSize: 1000,
	}
}

// Publish sends an event to all connected clients.
func (b *EventBus) Publish(event PipelineEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	event.Timestamp = time.Now()

	// Store in log
	b.eventLog = append(b.eventLog, event)
	if len(b.eventLog) > b.maxLogSize {
		b.eventLog = b.eventLog[1:]
	}

	// Send to all clients
	for client := range b.clients {
		select {
		case client <- event:
		default:
			// Client too slow, skip
		}
	}
}

// Subscribe creates a new channel for receiving events.
func (b *EventBus) Subscribe() chan PipelineEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	client := make(chan PipelineEvent, 100)
	b.clients[client] = struct{}{}
	return client
}

// Unsubscribe removes a client channel.
func (b *EventBus) Unsubscribe(client chan PipelineEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.clients, client)
	close(client)
}

// GetEventLog returns recent events.
func (b *EventBus) GetEventLog() []PipelineEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]PipelineEvent, len(b.eventLog))
	copy(result, b.eventLog)
	return result
}

// Clear removes all events from the log.
func (b *EventBus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.eventLog = b.eventLog[:0]
}

// SSEHandler returns an HTTP handler for Server-Sent Events.
func (b *EventBus) SSEHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Subscribe before replaying recent events. Events published after this point are
		// delivered through the live channel, so the replay must only include the snapshot
		// captured before subscription to avoid duplicating in-flight events.
		recentEvents := b.GetEventLog()
		client := b.Subscribe()
		defer b.Unsubscribe(client)

		// Send recent events first for page refresh recovery.
		for _, event := range recentEvents {
			if !eventMatchesRequest(event, r) {
				continue
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		flusher.Flush()
		// Listen for new events or client disconnect
		for {
			select {
			case event, ok := <-client:
				if !ok {
					return
				}
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

func eventMatchesRequest(event PipelineEvent, r *http.Request) bool {
	projectID := r.URL.Query().Get("projectId")
	if projectID != "" && event.ProjectID != projectID {
		return false
	}
	jobID := r.URL.Query().Get("jobId")
	if jobID != "" && event.JobID != jobID {
		return false
	}
	return true
}

// EventHandler returns an HTTP handler for querying events.
func (b *EventBus) EventHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		events := b.GetEventLog()
		json.NewEncoder(w).Encode(events)
	}
}
