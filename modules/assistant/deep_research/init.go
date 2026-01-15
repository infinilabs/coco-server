package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func RunDeepResearch(ctx context.Context, userQuery string, cfg *core.Assistant) error {

	config := cfg.DeepResearchConfig

	log.Error(util.ToJson(config, true))

	log.Trace("=== Open Deep Research ===")
	log.Debugf("Research Model: %s", config.ResearchModel)
	log.Debugf("Final Report Model: %s", config.ReportModel)
	log.Debugf("Max Researcher Iterations: %d", config.MaxResearcherIterations)
	log.Debugf("Max Concurrent Research Units: %d", config.MaxConcurrentResearchUnits)

	// Create deep researcher graph
	log.Debug("Initializing Deep Researcher...")
	deepResearcher, err := CreateDeepResearcherGraph(ctx, config)
	if err != nil {
		log.Errorf("Failed to create deep researcher: %v", err)
		return nil
	}

	// Define research query
	query := fmt.Sprintf(`%v?`, userQuery)

	log.Debugf("Research Query: %s\n\n", query)

	// Prepare initial state
	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, query),
		},
	}

	log.Debug("Starting research process...")

	result, err := deepResearcher.Invoke(ctx, initialState)
	if err != nil {
		log.Errorf("Research execution failed: %v", err)
		return nil
	}

	// Extract final report
	resultState := result.(map[string]interface{})
	finalReport, ok := resultState["final_report"].(string)
	if !ok {
		log.Error("No final report generated")
		return nil
	}

	// Display results
	log.Info("\n" + strings.Repeat("=", 80))
	log.Info("RESEARCH COMPLETE")
	log.Info(strings.Repeat("=", 80))
	fmt.Println()
	fmt.Println(finalReport)
	fmt.Println()
	log.Info(strings.Repeat("=", 80))

	// Display metadata
	notes, _ := resultState["notes"].([]string)
	rawNotes, _ := resultState["raw_notes"].([]string)

	log.Infof("\nMetadata:")
	log.Infof("- Research iterations: %v", resultState["research_iterations"])
	log.Infof("- Research findings collected: %d", len(notes))
	log.Infof("- Raw search results: %d", len(rawNotes))
	log.Infof("- Final report length: %d characters", len(finalReport))
	return nil
}
