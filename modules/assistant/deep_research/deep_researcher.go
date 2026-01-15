package deep_research

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
)

func CreateDeepResearcherGraph(ctx context.Context, config *core.DeepResearchConfig, reqMsg, replyMsg *core.ChatMessage, sender core.MessageSender) (*graph.StateRunnable, error) {

	// Create researcher subgraph
	researcherGraph, err := CreateResearcherGraph(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create researcher graph: %w", err)
	}

	// Create supervisor subgraph
	supervisorGraph, err := CreateSupervisorGraph(ctx, config, researcherGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to create supervisor graph: %w", err)
	}

	// Create main workflow
	workflow := graph.NewStateGraph()

	// Define state schema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("messages", graph.AppendReducer)
	schema.RegisterReducer("supervisor_messages", graph.AppendReducer)
	schema.RegisterReducer("notes", graph.AppendReducer)
	schema.RegisterReducer("raw_notes", graph.AppendReducer)
	workflow.SetSchema(schema)

	// Initialize research node - creates research brief
	workflow.AddNode("init_research", "init_research", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState := state.(map[string]interface{})
		messages, ok := mState["messages"].([]llms.MessageContent)
		if !ok || len(messages) == 0 {
			return nil, fmt.Errorf("no messages in state")
		}

		// Get user's query from the first message
		userQuery := ""
		for _, part := range messages[0].Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				userQuery = textPart.Text
				break
			}
		}

		if userQuery == "" {
			return nil, fmt.Errorf("could not extract user query")
		}

		// Create research brief (simplified - in production, could use LLM to refine)
		researchBrief := fmt.Sprintf("研究以下主题：%s", userQuery)

		log.Debug("[Init Research] Research brief created: %s", researchBrief)

		return map[string]interface{}{
			"research_brief":      researchBrief,
			"supervisor_messages": []llms.MessageContent{},
			"notes":               []string{},
			"raw_notes":           []string{},
			"research_iterations": 0,
		}, nil
	})

	// Supervisor subgraph node
	workflow.AddNode("supervisor", "supervisor", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState := state.(map[string]interface{})

		// Compile supervisor graph
		supervisorRunnable, err := supervisorGraph.Compile()
		if err != nil {
			return nil, fmt.Errorf("failed to compile supervisor: %w", err)
		}

		// Prepare supervisor state
		supervisorState := map[string]interface{}{
			"supervisor_messages": mState["supervisor_messages"],
			"research_brief":      mState["research_brief"],
			"notes":               mState["notes"],
			"raw_notes":           mState["raw_notes"],
			"research_iterations": mState["research_iterations"],
		}

		// Invoke supervisor
		result, err := supervisorRunnable.Invoke(ctx, supervisorState)
		if err != nil {
			return nil, fmt.Errorf("supervisor execution failed: %w", err)
		}

		resultState := result.(map[string]interface{})

		return map[string]interface{}{
			"supervisor_messages": resultState["supervisor_messages"],
			"notes":               resultState["notes"],
			"raw_notes":           resultState["raw_notes"],
			"research_iterations": resultState["research_iterations"],
		}, nil
	})

	// Final report generation node
	workflow.AddNode("final_report", "final_report", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState := state.(map[string]interface{})

		researchBrief, _ := mState["research_brief"].(string)
		notes, _ := mState["notes"].([]string)
		messages, _ := mState["messages"].([]llms.MessageContent)

		if len(notes) == 0 {
			return map[string]interface{}{
				"final_report": "没有可用的研究结果来生成报告。",
				"messages": []llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeAI, "没有可用的研究结果来生成报告。"),
				},
			}, nil
		}

		// Create final report prompt
		userMessages := GetMessagesString(messages)
		findings := JoinNotes(notes)
		prompt := GetFinalReportPrompt(researchBrief, userMessages, findings)

		log.Infof("[Final Report] Generating report with %d research findings", len(notes))

		resp, err := langchain.DirectGenerate(ctx, &config.ReportModel, langchain.GetPromptMessages(&config.ReportModel, "", prompt, nil, nil), func(chunk []byte, seq int) {
			msg := core.NewMessageChunk(reqMsg.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Think, string(chunk), seq)
			err = sender.SendMessage(msg)
			if err != nil {
				panic(err)
			}
		}, func(chunk []byte, seq int) {
			msg := core.NewMessageChunk(reqMsg.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Response, string(chunk), seq)
			err = sender.SendMessage(msg)
			if err != nil {
				panic(err)
			}
		})
		if err != nil {
			return nil, fmt.Errorf("report generation failed: %w", err)
		}

		finalReport := resp.Choices[0].Content

		log.Info("research report generated: ", finalReport)

		// Clean up markdown code blocks if present
		finalReport = strings.TrimPrefix(finalReport, "```markdown")
		finalReport = strings.TrimPrefix(finalReport, "```")
		finalReport = strings.TrimSuffix(finalReport, "```")

		finalReportInMarkdown := finalReport

		// Convert Markdown to HTML
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
		p := parser.NewWithExtensions(extensions)
		doc := p.Parse([]byte(finalReport))

		htmlFlags := html.CommonFlags | html.HrefTargetBlank
		opts := html.RendererOptions{Flags: htmlFlags}
		renderer := html.NewRenderer(opts)
		htmlContent := markdown.Render(doc, renderer)
		finalReportHTML := string(htmlContent)

		log.Infof("[Final Report] Report generated successfully (%d characters)", len(finalReportHTML))

		return map[string]interface{}{
			"markdown_report": finalReportInMarkdown,
			"html_report":     finalReportHTML,
			"messages": []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeAI, finalReport),
			},
			"notes": []string{}, // Clear notes after final report
		}, nil
	})

	// Define workflow edges
	workflow.SetEntryPoint("init_research")
	workflow.AddEdge("init_research", "supervisor")
	workflow.AddEdge("supervisor", "final_report")
	workflow.AddEdge("final_report", graph.END)

	// Compile the complete workflow
	return workflow.Compile()
}
