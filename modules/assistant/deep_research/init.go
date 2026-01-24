package deep_research

import (
	"context"
	"errors"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

func RunDeepResearch(ctx context.Context, userQuery string, config *core.DeepResearchConfig, reqMsg, replyMsg *core.ChatMessage, sender core.MessageSender) error {

	log.Debug("deep research config: ", util.ToJson(config, true))

	log.Trace("=== Open Deep Research ===")
	log.Debugf("Research Model: %s", config.ResearchModel)
	log.Debugf("Final Report Model: %s", config.ReportModel)
	log.Debugf("Max Researcher Iterations: %d", config.MaxResearcherIterations)
	log.Debugf("Max Concurrent Research Units: %d", config.MaxConcurrentResearchUnits)

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}
	// note: we use defer to ensure that the response message is saved after processing
	// even if user cancels the task or if an error occurs
	defer func() {
		//save response message to system
		if messageBuffer.Len() > 0 {
			replyMsg.Message = messageBuffer.String()
		} else {
			log.Warnf("seems empty reply for query: %v", replyMsg)
		}
		if reasoningBuffer.Len() > 0 {
			detail := core.ProcessingDetails{Order: 50, Type: common.Think, Description: reasoningBuffer.String()}
			replyMsg.Details = append(replyMsg.Details, detail)
		}
	}()

	// Create deep researcher graph
	log.Debug("Initializing Deep Researcher...")
	deepResearcher, err := CreateDeepResearcherGraph(ctx, config, reqMsg, replyMsg, sender)
	if err != nil {
		log.Errorf("Failed to create deep researcher: %v", err)
		return err
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
		return err
	}

	// Extract final report
	resultState := result.(map[string]interface{})
	finalReport, ok := resultState["markdown_report"].(string)
	if !ok {
		return errors.New("No final report generated")
	}

	messageBuffer.WriteString(finalReport)

	// Display results
	log.Info("\n" + strings.Repeat("=", 80))
	log.Info("RESEARCH COMPLETE")
	log.Info(strings.Repeat("=", 80))
	log.Info(finalReport)
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
