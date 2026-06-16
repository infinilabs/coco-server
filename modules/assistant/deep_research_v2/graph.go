package deep_research

import (
	"context"

	"github.com/smallnest/langgraphgo/graph"
	"infini.sh/coco/core"
)

// StepResult represents the result of a single research step
type StepResult struct {
	StepNumber     int      `json:"step_number"`
	StepQuery      string   `json:"step_query"`
	SearchResults  string   `json:"search_results"`
	Analysis       string   `json:"analysis"`        // Synthesized findings from a research step
	Status         string   `json:"status"`          // "pending", "in_progress", "completed", "failed"
	Confidence     float64  `json:"confidence"`      //  The quality/sufficiency score (0.0 to 1.0) of search results for a step
	SearchQueries  []string `json:"search_queries"`  // List of all search queries used for this step
	ProcessingTime string   `json:"processing_time"` // Time taken to process this step
	ErrorMessage   string   `json:"error_message"`   // Error details if failed
}

// ResearchProgress tracks overall research progress
type ResearchProgress struct {
	TotalSteps       int    `json:"total_steps"`
	CompletedSteps   int    `json:"completed_steps"`
	CurrentStep      int    `json:"current_step"`
	Status           string `json:"status"` // "planning", "researching", "completed", "failed"
	StartTime        string `json:"start_time"`
	EstTimeRemaining string `json:"est_time_remaining"`
}

// ChapterOutline represents a chapter in the report outline
type ChapterOutline struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Priority     int      `json:"priority"`      // 1-5, higher priority means more important
	Status       string   `json:"status"`        // "pending", "in_progress", "completed"
	Keywords     []string `json:"keywords"`      // Relevant keywords for research
	RelatedSteps []int    `json:"related_steps"` // Research step indices that contribute to this chapter
}

// MaterialReference represents a reference to research material for a chapter
type MaterialReference struct {
	ID         string  `json:"id"`
	ChapterID  string  `json:"chapter_id"`
	StepNumber int     `json:"step_number"`
	Source     string  `json:"source"` // "internal", "external"
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Summary    string  `json:"summary"`
	Relevance  float64 `json:"relevance"`  // Relevance score to this chapter
	Confidence float64 `json:"confidence"` // Search confidence score
	CreatedAt  string  `json:"created_at"`
}

// ChapterContent represents the compiled content for a chapter
type ChapterContent struct {
	ChapterID       string              `json:"chapter_id"`
	Title           string              `json:"title"`
	Content         string              `json:"content"`
	Materials       []MaterialReference `json:"materials"`
	ImageReferences []string            `json:"image_references"`
	Status          string              `json:"status"` // "draft", "review", "completed"
	LastUpdated     string              `json:"last_updated"`
	KeyPoints       []string            `json:"key_points"`
	SourceCount     int                 `json:"source_count"`
	InternalSources int                 `json:"internal_sources"`
	ExternalSources int                 `json:"external_sources"`
}

// Request represents the initial input to the research agent.
type Request struct {
	Query string `json:"query"`
}

// ChunkRecord is a lightweight representation of a streamed chunk,
// saved to ProcessingDetails so the frontend can replay the deep research UI from history.
type ChunkRecord struct {
	ChunkType    string `json:"chunk_type"`
	MessageChunk string `json:"message_chunk,omitempty"`
}

// State represents the state of the research agent.
type State struct {
	Request         Request      `json:"request"`
	Plan            []string     `json:"plan"`             // Research plan steps
	StepResults     []StepResult `json:"step_results"`     // Detailed results per step
	ResearchResults []string     `json:"research_results"` // Legacy format for backward compatibility

	// MarkdownReport, HTMLReport and PDFReport hold the three rendering formats
	// of the final report. All three are set inside the Reporter node (the last
	// graph node) and are read only after graph.Invoke returns — no downstream
	// node in the graph ever reads them. The json:"-" tag skips unnecessary
	// serialization between node transitions.
	MarkdownReport string `json:"-"`
	HTMLReport     string `json:"-"`
	PDFReport      []byte `json:"-"`

	//Step            int          `json:"step"`
	// Chapter Management
	ChapterOutline   []ChapterOutline           `json:"chapter_outline"`         // Report chapter structure
	ChapterContents  map[string]*ChapterContent `json:"chapter_contents"`        // Content per chapter
	AllMaterials     []MaterialReference        `json:"all_materials,omitempty"` // All collected materials (reserved)
	MaterialRegistry map[string]bool            `json:"-"`                       // Track material uniqueness
	// System
	Config      *core.DeepResearchConfig `json:"-"`
	Sender      core.MessageSender       `json:"-"`
	StartTime   int64                    `json:"-"` // Unix timestamp for timing
	Attachments []*core.Attachment       `json:"-"` // User-uploaded files; text is injected into the planner prompt.
	Chunks      []ChunkRecord            `json:"-"` // Collected streaming chunks for persistence
}

// sendAndCollect sends a chunk to the client and records it for later persistence.
func (s *State) sendAndCollect(chunkType, messageChunk string) {
	s.Sender.SendChunkMessage(core.MessageTypeAssistant, chunkType, messageChunk, 0)
	s.Chunks = append(s.Chunks, ChunkRecord{ChunkType: chunkType, MessageChunk: messageChunk})
}

// NewGraph creates and configures the research agent graph.
func NewGraph() (*graph.StateRunnable, error) {
	workflow := graph.NewStateGraph()

	// Add nodes
	workflow.AddNode("planner", "Research planning node", PlannerNode)
	workflow.AddNode("researcher", "Research execution node", ResearcherNode)
	workflow.AddNode("reporter", "Report generation node", ReporterNode)

	// Add edges
	// Start -> Planner
	workflow.SetEntryPoint("planner")

	// Planner -> Researcher
	workflow.AddEdge("planner", "researcher")

	// Researcher -> Reporter
	workflow.AddEdge("researcher", "reporter")

	// Reporter -> END
	workflow.AddConditionalEdge("reporter", func(ctx context.Context, state interface{}) string {
		return graph.END
	})

	return workflow.Compile()
}

// Define the node functions signatures here to avoid compilation errors in this file,
// but the actual implementation will be in nodes.go.
// Since they are in the same package (main), we don't need to declare them here if they are defined in nodes.go.
// But for clarity, I'll just rely on them being in nodes.go.
