package deep_research

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/infinilabs/picoloom/v2"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

// PlannerNode generates a research plan based on the query.
func PlannerNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)

	s.sendAndCollect(common.ResearchPlannerStart, "")

	planningModel, err := resolveStageModel(s.Config.PlanningModel, "planning")
	if err != nil {
		return nil, err
	}
	llm, err := langchain.GetLLMByConfig(planningModel)
	if err != nil {
		return nil, err
	}

	maxSteps := s.Config.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 10
	}

	depthHint := map[string]string{
		"basic":         "3-5",
		"comprehensive": "5-10",
		"exhaustive":    "10-15",
	}[s.Config.ResearchDepth]
	if depthHint == "" {
		depthHint = "5-10"
	}

	prompt := fmt.Sprintf(`You are a research planner. Create a step-by-step research plan for the following query: %s.

Generate no more than %d research steps. Aim for %s steps based on the %s research depth.

Return the result in JSON format:
{
    "plan": ["step 1", "step 2", ...]
}

Respond in %s.`, s.Request.Query, maxSteps, depthHint, s.Config.ResearchDepth, reportLang(s.Config.ReportLang))

	// Prepend uploaded document content so the planner can treat it as primary source material.
	if attachmentsSection := strings.TrimSpace(langchain.FormatAttachmentsSection(s.Attachments)); attachmentsSection != "" {
		prompt = fmt.Sprintf("The user has uploaded the following documents. Treat them as primary source material when forming the research plan:\n%s\n\n%s", attachmentsSection, prompt)
	}

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, langchain.PromptWithCurrentTime(prompt))
	if err != nil {
		return nil, err
	}

	// Clean up JSON
	completion = strings.TrimSpace(completion)
	completion = strings.TrimPrefix(completion, "```json")
	completion = strings.TrimPrefix(completion, "```")
	completion = strings.TrimSuffix(completion, "```")
	completion = strings.TrimSpace(completion)

	var output struct {
		Plan []string `json:"plan"`
	}

	if err := json.Unmarshal([]byte(completion), &output); err != nil {
		// Fallback: simple parsing
		lines := strings.Split(completion, "\n")
		var plan []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "}") {
				plan = append(plan, trimmed)
			}
		}
		s.Plan = plan
	} else {
		s.Plan = output.Plan
	}

	// Enforce MaxSteps limit
	if maxSteps > 0 && len(s.Plan) > maxSteps {
		s.Plan = s.Plan[:maxSteps]
	}

	s.sendAndCollect(common.ResearchPlannerEnd, util.MustToJSON(s.Plan))

	return s, nil
}

// ResearcherNode executes the research plan using real search with feedback mechanism and chapter management.
func ResearcherNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)

	researchModel, err := resolveStageModel(s.Config.ResearchModel, "research")
	if err != nil {
		return nil, err
	}
	llm, err := langchain.GetLLMByConfig(researchModel)
	if err != nil {
		return nil, err
	}

	// Initialize state components if not present
	if s.StartTime == 0 {
		s.StartTime = time.Now().Unix()
		s.MaterialRegistry = make(map[string]bool)
		s.ChapterContents = make(map[string]*ChapterContent)
	}

	// Initialize chapter outline if not present
	if len(s.ChapterOutline) == 0 {
		err := s.generateChapterOutline(ctx, llm)
		if err != nil {
			log.Warnf("Failed to generate chapter outline: %v", err)
		}
	}

	// Initialize step results if not present
	if len(s.StepResults) != len(s.Plan) {
		s.StepResults = make([]StepResult, len(s.Plan))
		for i := range s.StepResults {
			s.StepResults[i] = StepResult{
				StepNumber: i + 1,
				StepQuery:  s.Plan[i],
				Status:     "pending",
			}
		}
	}

	var results []string

	// Get synthesis LLM for the analysis step (falls back to ResearchModel if not configured)
	synthesisLLM := llm
	if s.Config.SynthesisModel.Name != "" {
		if sl, err := langchain.GetLLMByConfig(s.Config.SynthesisModel); err == nil {
			synthesisLLM = sl
		} else {
			log.Warnf("Failed to get synthesis model, falling back to research model: %v", err)
		}
	}

	// Respect MaxResearcherIterations limit
	maxIter := s.Config.MaxResearcherIterations
	if maxIter <= 0 {
		maxIter = len(s.Plan)
	}

	// Process each research step with feedback-based search and chapter distribution
	for stepIndex, step := range s.Plan {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if stepIndex >= maxIter {
			log.Infof("Reached MaxResearcherIterations (%d), stopping", maxIter)
			break
		}
		stepStartTime := time.Now()

		payload := util.MapStr{}
		payload["plan"] = step
		s.sendAndCollect(common.ResearchResearcherStart, util.MustToJSON(payload))

		// Update current step status
		if stepIndex < len(s.StepResults) {
			s.StepResults[stepIndex].Status = "in_progress"
		}

		var stepSearchQueries []string
		var stepMaterials []MaterialReference

		//TODO convert `step` to queries `keywords`
		query := step
		searchPayload := util.MapStr{}
		searchPayload["plan"] = step
		searchPayload["step"] = util.MapStr{
			"type": "search",
			"name": "Search Materials",
			"payload": util.MapStr{
				"from":  0,
				"size":  10,
				"query": query,
			},
		} //TODO, convert to query
		s.sendAndCollect(common.ResearchResearcherStepStart, util.MustToJSON(searchPayload))

		// Step 1: Initial search for this research step
		initialSearchCollection, err := SearchWithConfig(ctx, step, s.Config, true) // internal first
		if err != nil {
			log.Warnf("Initial search failed for step '%s': %v", step, err)
			defErrorResult := fmt.Sprintf("Step: %s\nFindings: Search failed: %v", step, err)
			results = append(results, defErrorResult)

			if stepIndex < len(s.StepResults) {
				s.StepResults[stepIndex].Status = "failed"
				s.StepResults[stepIndex].ErrorMessage = err.Error()
				s.StepResults[stepIndex].Analysis = defErrorResult
			}
			continue
		}

		searchPayload = util.MapStr{}
		searchPayload["plan"] = step
		searchPayload["step"] = util.MapStr{
			"type": "search",
			"name": "Search Materials",
			"payload": util.MapStr{
				"total": 10,
				"hits":  initialSearchCollection.Results,
			},
		} //TODO, convert to query

		// Step 2: Analyze search results and potentially refine search
		if !initialSearchCollection.IsSufficient {

			// Generate refinement query based on initial results analysis
			refinementPrompt := fmt.Sprintf(`Based on the following search results analysis, generate a more specific search query for this research step:

Research step: %s
Initial search results: %s
Current confidence: %.2f%%

Generate a more specific search query to obtain more detailed or relevant information. Only return the search query, nothing else.`,
				step,
				initialSearchCollection.FormatResultsForLLM(),
				initialSearchCollection.Confidence*100)

			refinementQuery, err := llms.GenerateFromSinglePrompt(ctx, llm, langchain.PromptWithCurrentTime(refinementPrompt))
			if err == nil && strings.TrimSpace(refinementQuery) != "" {
				stepSearchQueries = append(stepSearchQueries, refinementQuery)

				// Perform refined search
				refinedCollection, err := SearchWithConfig(ctx, refinementQuery, s.Config, false) // external first
				if err == nil {
					// Combine initial and refined results
					initialSearchCollection.Results = append(initialSearchCollection.Results, refinedCollection.Results...)
					initialSearchCollection.evaluateSearchQuality() // Re-evaluate
				}
			}
		}

		// Send step end with all combined results (initial + refined)
		searchPayload = util.MapStr{}
		searchPayload["plan"] = step
		searchPayload["step"] = util.MapStr{
			"type": "search",
			"name": "Search Materials",
			"payload": util.MapStr{
				"total": len(initialSearchCollection.Results),
				"hits":  initialSearchCollection.Results,
			},
		}
		s.sendAndCollect(common.ResearchResearcherStepEnd, util.MustToJSON(searchPayload))

		stepSearchQueries = append(stepSearchQueries, step)

		// Step 3: Convert search results to material references and allocate to chapters
		stepMaterials = s.convertToMaterials(initialSearchCollection, stepIndex+1)

		// Step 4: Distribute materials to relevant chapters
		allocatedMaterials := s.distributeMaterialsToChapters(stepMaterials, stepIndex)

		// Step 5: Analyze and synthesize findings with chapter-aware context
		findingsPrompt := s.generateChapterAwareAnalysisPrompt(step, allocatedMaterials, stepIndex)

		completion, err := llms.GenerateFromSinglePrompt(ctx, synthesisLLM, langchain.PromptWithCurrentTime(findingsPrompt))
		if err != nil {
			log.Warnf("Analysis failed for step '%s': %v", step, err)
			errorResult := fmt.Sprintf("Step: %s\nFindings: Analysis failed: %v\n\nSearch results: %s",
				step, err, initialSearchCollection.FormatResultsForLLM())
			results = append(results, errorResult)

			if stepIndex < len(s.StepResults) {
				s.StepResults[stepIndex].Status = "failed"
				s.StepResults[stepIndex].ErrorMessage = err.Error()
				s.StepResults[stepIndex].Analysis = errorResult
				s.StepResults[stepIndex].SearchQueries = stepSearchQueries
				s.StepResults[stepIndex].SearchResults = initialSearchCollection.FormatResultsForLLM()
				s.StepResults[stepIndex].Confidence = initialSearchCollection.Confidence
			}
		} else {
			//log.Info(fmt.Sprintf("Step: %s\nFindings: %s", step, completion))
			successResult := fmt.Sprintf("Step: %s\nFindings: %s", step, completion)
			results = append(results, successResult)

			if stepIndex < len(s.StepResults) {
				s.StepResults[stepIndex].Status = "completed"
				s.StepResults[stepIndex].Analysis = successResult
				s.StepResults[stepIndex].SearchQueries = stepSearchQueries
				s.StepResults[stepIndex].SearchResults = initialSearchCollection.FormatResultsForLLM()
				s.StepResults[stepIndex].Confidence = initialSearchCollection.Confidence
				s.StepResults[stepIndex].ProcessingTime = time.Since(stepStartTime).String()
			}
		}

		// Step 7: Update chapter progress
		s.updateChapterProgress(stepIndex, allocatedMaterials)

		s.sendAndCollect(common.ResearchResearcherEnd, util.MustToJSON(payload))
	}

	s.ResearchResults = results

	return s, nil
}

// ReporterNode compiles the final report using organized chapter structure and materials.
func ReporterNode(ctx context.Context, state interface{}) (interface{}, error) {

	s := state.(*State)

	s.sendAndCollect(common.ResearchReporterStart, "")

	reportModel, err := resolveStageModel(s.Config.ReportModel, "report")
	if err != nil {
		return nil, err
	}
	llm, err := langchain.GetLLMByConfig(reportModel)
	if err != nil {
		return nil, err
	}

	// Strategy: Use the structured chapter approach; require organized materials.
	if len(s.ChapterOutline) == 0 {
		return nil, fmt.Errorf("no chapter outline available for report generation")
	}
	return s.generateChapterBasedReport(ctx, llm)
}

// generateChapterOutline creates an intelligent chapter structure for the report
func (s *State) generateChapterOutline(ctx context.Context, llm llms.Model) error {
	if len(s.Plan) == 0 {
		return fmt.Errorf("no research plan available")
	}

	prompt := fmt.Sprintf(`Based on the following research query and research plan, generate a detailed report chapter outline. Chapters should be logically structured, covering all important aspects of the research, with clear focus and relevant keywords for each chapter.

Research query: %s
Research plan:
%s

Generate JSON format chapter outline:
[
  {
    "id": "chapter_1",
    "title": "Chapter Title",
    "description": "Chapter content description",
    "priority": 5,  // 1-5 importance level
    "keywords": ["keyword1", "keyword2"],
    "related_steps": [1, 2, 3]  // Related research step numbers
  }
]

Requirements:
1. Generate 4-8 chapters
2. Each chapter must have a non-empty "title" and "description"
3. The chapter array must not be empty
4. Chapters should progress logically, from basic to in-depth
5. Use %s for titles and descriptions
6. Accurately relate to relevant research steps`,
		s.Request.Query,
		strings.Join(s.Plan, "\n"),
		reportLang(s.Config.ReportLang))

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, langchain.PromptWithCurrentTime(prompt))
	if err != nil {
		return err
	}

	// Parse JSON response
	completion = strings.TrimSpace(completion)
	completion = strings.TrimPrefix(completion, "```json")
	completion = strings.TrimPrefix(completion, "```")
	completion = strings.TrimSuffix(completion, "```")
	completion = strings.TrimSpace(completion)

	var chapters []ChapterOutline
	if err := json.Unmarshal([]byte(completion), &chapters); err != nil {
		return fmt.Errorf("failed to parse chapter outline: %v", err)
	}

	log.Info("Generated report chapters: ", util.ToJson(chapters, true))

	s.ChapterOutline = chapters
	return nil
}

// convertToMaterials converts search results to MaterialReference objects
func (s *State) convertToMaterials(collection *SearchResultCollection, stepNumber int) []MaterialReference {
	var materials []MaterialReference

	for _, result := range collection.Results {
		materialID := fmt.Sprintf("material_%v", result.URL)

		// Check if material already exists
		if _, exists := s.MaterialRegistry[materialID]; exists {
			continue
		}

		material := MaterialReference{
			ID:         materialID,
			ChapterID:  "", // Will be assigned during distribution
			StepNumber: stepNumber,
			Source:     result.Source,
			Title:      result.Title,
			URL:        result.URL,
			Content:    result.Content,
			CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
			Confidence: result.Score,
		}

		// Generate summary using first 300 characters
		if len(result.Content) > 300 {
			material.Summary = result.Content[:300] + "..."
		} else {
			material.Summary = result.Content
		}

		materials = append(materials, material)
		s.MaterialRegistry[materialID] = true
	}

	//s.AllMaterials = append(s.AllMaterials, materials...)
	return materials
}

// distributeMaterialsToChapters intelligently assigns materials to relevant chapters
func (s *State) distributeMaterialsToChapters(materials []MaterialReference, stepIndex int) []MaterialReference {
	if len(s.ChapterOutline) == 0 {
		log.Warn("No chapter outline available, skipping material distribution")
		return materials
	}

	var allocatedMaterials []MaterialReference

	for i, material := range materials {
		// Find best matching chapter based on content relevance
		bestChapterID := ""
		bestScore := 0.0

		for _, chapter := range s.ChapterOutline {
			score := s.calculateRelevance(material.Summary, chapter.Keywords, []string{chapter.Title})
			if score > bestScore {
				bestScore = score
				bestChapterID = chapter.ID
			}
		}

		// Assign material to best chapter or default chapter
		if bestChapterID != "" {
			materials[i].ChapterID = bestChapterID
			materials[i].Relevance = bestScore
			allocatedMaterials = append(allocatedMaterials, materials[i])

			// Initialize chapter content if needed
			if s.ChapterContents[bestChapterID] == nil {
				s.initializeChapterContent(bestChapterID)
			}

			// Add material to chapter
			s.ChapterContents[bestChapterID].Materials = append(
				s.ChapterContents[bestChapterID].Materials,
				materials[i],
			)
		}
	}

	return allocatedMaterials
}

// TODO move it to LLM
// calculateRelevance calculates relevance score between content and keywords
func (s *State) calculateRelevance(content string, keywords []string, additionalTerms []string) float64 {
	if len(keywords) == 0 && len(additionalTerms) == 0 {
		return 0.0
	}

	score := 0.0
	contentLower := strings.ToLower(content)

	// Check keywords
	for _, keyword := range keywords {
		if strings.Contains(contentLower, strings.ToLower(keyword)) {
			score += 1.0
		}
	}

	// Check additional terms
	for _, term := range additionalTerms {
		if strings.Contains(contentLower, strings.ToLower(term)) {
			score += 0.8
		}
	}

	maxPossible := float64(len(keywords) + len(additionalTerms))
	if maxPossible == 0 {
		return 0.0
	}

	return score / maxPossible
}

// initializeChapterContent initializes a chapter content structure
func (s *State) initializeChapterContent(chapterID string) {
	// Find chapter info
	var chapterInfo *ChapterOutline
	for _, chapter := range s.ChapterOutline {
		if chapter.ID == chapterID {
			chapterInfo = &chapter
			break
		}
	}

	if chapterInfo == nil {
		log.Warnf("Chapter %s not found in outline", chapterID)
		return
	}

	s.ChapterContents[chapterID] = &ChapterContent{
		ChapterID:       chapterID,
		Title:           chapterInfo.Title,
		Materials:       []MaterialReference{},
		ImageReferences: []string{},
		Status:          "draft",
		LastUpdated:     time.Now().Format("2006-01-02 15:04:05"),
		KeyPoints:       []string{},
	}
}

// generateChapterAwareAnalysisPrompt creates analysis prompt with chapter context
func (s *State) generateChapterAwareAnalysisPrompt(step string, allocatedMaterials []MaterialReference, stepIndex int) string {
	materialsInfo := ""
	if len(allocatedMaterials) > 0 {
		materialsInfo = "\nAllocated materials:\n"
		for _, material := range allocatedMaterials {
			materialsInfo += fmt.Sprintf("- %s (%s)\n", material.Title, material.Summary[:min(len(material.Summary), 100)])
		}
	}

	return fmt.Sprintf(`You are a researcher. Based on the following search results, provide detailed findings and insights for this research step.

Research step: %s
Search results details: %s
%s

Requirements:
1. Provide detailed findings and insights
2. If search results are insufficient, clearly state so
3. Respond in %s
4. Organize content by importance
5. Combine analysis with assigned chapter materials`,
		step,
		"searched results", // For preview, not actual search results - replaced direct call
		materialsInfo,
		reportLang(s.Config.ReportLang))
}

// updateChapterProgress updates chapter progress and status
func (s *State) updateChapterProgress(stepIndex int, materials []MaterialReference) {
	for _, material := range materials {
		if chapter, exists := s.ChapterContents[material.ChapterID]; exists {
			// Update source counters
			if material.Source == "internal" {
				chapter.InternalSources++
			} else {
				chapter.ExternalSources++
			}
			chapter.SourceCount = chapter.InternalSources + chapter.ExternalSources

			// Update status based on progress
			if chapter.SourceCount > 3 {
				chapter.Status = "well_researched"
			} else if chapter.SourceCount > 0 {
				chapter.Status = "some_material"
			}

			chapter.LastUpdated = time.Now().Format("2006-01-02 15:04:05")
		}
	}

	// Update related steps for chapters
	for _, chapter := range s.ChapterOutline {
		if containsInt(chapter.RelatedSteps, stepIndex+1) {
			if content, exists := s.ChapterContents[chapter.ID]; exists {
				content.LastUpdated = time.Now().Format("2006-01-02 15:04:05")
			}
		}
	}
}

// generateChapterContent generates comprehensive chapter content for each chapter using allocated materials.
// The indexMap assigns global indices to materials so that LLM citations like [5] are
// consistent with the unified References section at the end of the report.
func (s *State) generateChapterContent(ctx context.Context, llm llms.Model, indexMap map[string]int) map[string]*ChapterContent {
	log.Info("Starting chapter content generation...")

	for _, chapter := range s.ChapterOutline {
		chapterID := chapter.ID
		content, exists := s.ChapterContents[chapterID]
		if !exists {
			s.initializeChapterContent(chapterID)
			content = s.ChapterContents[chapterID]
		}
		content.Status = "generating"

		i18n := getReportI18n(s.Config.ReportLang)

		// Build comprehensive material reference for this chapter
		materialsInfo := s.buildMaterialsInfo(content.Materials, indexMap)
		log.Infof("Generating content for chapter %s with %d materials", content.Title, len(content.Materials))

		// Generate comprehensive chapter content from materials
		prompt := fmt.Sprintf(`You are a professional report writer.

Research topic: %s
Chapter title: %s

Based on the following intelligently categorized research materials, write a detailed professional report for this chapter:

%s

Requirements:
1. Content must be based on provided materials, do not add fictional information
2. Maintain academic and professional style
3. Integrate all relevant materials with in-depth analysis and insights
4. Use clear Markdown format with reasonable hierarchy
5. Organize content logically, from basic to in-depth
6. Cite relevant material sources after each key point (e.g., [1], [2])
7. Word count: 1000-2000 words
8. Write in %s
9. Do NOT append a references or bibliography section at the end of the chapter; citations in the body are sufficient

Generate the chapter content directly, do not add explanatory text.`,
			s.Request.Query,
			content.Title,
			materialsInfo,
			reportLang(s.Config.ReportLang))

		completion, err := llms.GenerateFromSinglePrompt(ctx, llm, langchain.PromptWithCurrentTime(prompt))
		if err != nil {
			log.Warnf("Failed to generate chapter content for %s: %v", chapterID, err)
			content.Content = fmt.Sprintf(i18n.GenerationFailed, content.Title, err)
			content.Status = "error"
		} else {
			completion = strings.TrimSpace(completion)
			completion = strings.TrimPrefix(completion, "```markdown")
			completion = strings.TrimPrefix(completion, "```")
			completion = strings.TrimSuffix(completion, "```")
			completion = strings.TrimSpace(completion)

			content.Content = completion
			content.Status = "completed"

			// Generate key points from the content
			if overview, err := s.extractKeyPoints(ctx, llm, completion); err == nil {
				content.KeyPoints = overview
			}
		}

		content.LastUpdated = time.Now().Format("2006-01-02 15:04:05")
		s.ChapterContents[chapterID] = content
	}
	log.Info("Chapter content generation completed")
	return s.ChapterContents
}

// buildMaterialIndexMap scans all chapter materials and assigns each unique URL a
// global index [1]-[N] based on first appearance across the outline. The returned
// map is used by buildMaterialsInfo so that LLM citations like [5] refer to the
// same source in the unified References section at the end of the report.
func (s *State) buildMaterialIndexMap() map[string]int {
	indexMap := make(map[string]int)
	idx := 1
	for _, chapter := range s.ChapterOutline {
		if content, exists := s.ChapterContents[chapter.ID]; exists {
			for _, m := range content.Materials {
				if _, ok := indexMap[m.URL]; !ok {
					indexMap[m.URL] = idx
					idx++
				}
			}
		}
	}
	return indexMap
}

// buildMaterialsInfo constructs a comprehensive description of all materials in a chapter.
// It uses the global indexMap so that Material[N] identifiers are consistent across
// the entire report and match the unified References section.
func (s *State) buildMaterialsInfo(materials []MaterialReference, indexMap map[string]int) string {
	var builder strings.Builder

	builder.WriteString("## Research Materials Summary\n\n")
	for i, material := range materials {
		if i > 10 { // Limit for clarity
			builder.WriteString(fmt.Sprintf("\n... and %d other related materials\n", len(materials)-10))
			break
		}

		globalIdx := indexMap[material.URL]
		builder.WriteString(fmt.Sprintf("### Material[%d]: %s\n", globalIdx, material.Title))
		builder.WriteString(fmt.Sprintf("- Source: %s (Relevance: %.2f)\n", material.Source, material.Relevance))
		if material.URL != "" {
			builder.WriteString(fmt.Sprintf("- Link: %s\n", material.URL))
		}
		builder.WriteString(fmt.Sprintf("- Content Summary: %s\n", material.Summary))
		builder.WriteString("\n")
	}
	return builder.String()
}

// extractKeyPoints extracts the main points from generated content
func (s *State) extractKeyPoints(ctx context.Context, llm llms.Model, content string) ([]string, error) {
	prompt := fmt.Sprintf(`Extract 3-5 of the most important points/key takeaways from the following content, each point should be concise and clear:

Content:
%s

Return format:
- Point 1
- Point 2
- Point 3...

Do not add other text. Respond in %s.`, content, reportLang(s.Config.ReportLang))

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, langchain.PromptWithCurrentTime(prompt))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(completion, "\n")
	var keyPoints []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") {
			keyPoints = append(keyPoints, strings.TrimPrefix(line, "-"))
		}
	}

	return keyPoints, nil
}

// generateReportTitle produces a concise title based on the assembled report
// body. It falls back to the original query if title generation fails.
func (s *State) generateReportTitle(ctx context.Context, llm llms.Model, body string) string {
	prompt := fmt.Sprintf(`You are a professional report title writer. Based on the following research report content, generate a concise, professional title (5-10 words). Return ONLY the title text, no quotes, no markdown, no extra explanation.

Report content:
%s

Write the title in %s.`, body, reportLang(s.Config.ReportLang))

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, langchain.PromptWithCurrentTime(prompt))
	if err != nil {
		log.Warnf("failed to generate report title: %v, falling back to query", err)
		return s.Request.Query
	}

	title := strings.TrimSpace(completion)
	title = strings.TrimPrefix(title, "#")
	title = strings.TrimPrefix(title, "```markdown")
	title = strings.TrimPrefix(title, "```")
	title = strings.TrimSuffix(title, "```")
	title = strings.TrimSpace(title)

	if title == "" {
		return s.Request.Query
	}
	return title
}

// generateChapterBasedReport creates a structured report based on the chapter outline and generated content
func (s *State) generateChapterBasedReport(ctx context.Context, llm llms.Model) (*State, error) {

	// Debug output for chapters

	contents := s.ChapterContents
	i18n := getReportI18n(s.Config.ReportLang)

	// Assign global material indices across all chapters before generating content
	// so that LLM citations like [5] are consistent with the unified References section.
	indexMap := s.buildMaterialIndexMap()

	// Generate chapter content if needed
	needsGeneration := false
	for _, chapter := range s.ChapterOutline {
		if content, exists := contents[chapter.ID]; !exists || content.Content == "" {
			needsGeneration = true
			break
		}
	}
	if needsGeneration {
		contents = s.generateChapterContent(ctx, llm, indexMap)
	}

	// Build the summary section.
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString(fmt.Sprintf("## %s\n\n", i18n.Summary))
	allKeyPoints := []string{}
	for _, chapter := range s.ChapterOutline {
		if content, exists := contents[chapter.ID]; exists {
			if len(content.KeyPoints) > 0 {
				allKeyPoints = append(allKeyPoints, content.KeyPoints...)
			}
		}
	}
	if len(allKeyPoints) > 0 {
		for _, point := range allKeyPoints {
			summaryBuilder.WriteString(fmt.Sprintf("- %s\n", point))
		}
		summaryBuilder.WriteString("\n")
	} else {
		summaryBuilder.WriteString(i18n.FallbackIntro + "\n\n")
	}
	summary := summaryBuilder.String()

	// Build the chapter contents section.
	var chaptersBuilder strings.Builder
	for _, chapter := range s.ChapterOutline {
		if content, exists := contents[chapter.ID]; exists && content.Content != "" {
			chaptersBuilder.WriteString(fmt.Sprintf("## %s\n\n", chapter.Title))
			chaptersBuilder.WriteString(content.Content + "\n\n")
		}
	}

	// Append a unified References section at the end of the report.
	// Collect all unique materials and sort by their global index so that
	// the order matches the [N] citations in the chapter bodies.
	seen := make(map[string]bool)
	var allMaterials []MaterialReference
	for _, chapter := range s.ChapterOutline {
		if content, exists := contents[chapter.ID]; exists {
			for _, m := range content.Materials {
				if !seen[m.URL] && indexMap[m.URL] > 0 {
					seen[m.URL] = true
					allMaterials = append(allMaterials, m)
				}
			}
		}
	}
	if len(allMaterials) > 0 {
		// Sort by global index to maintain citation order.
		for i := 0; i < len(allMaterials); i++ {
			for j := i + 1; j < len(allMaterials); j++ {
				if indexMap[allMaterials[i].URL] > indexMap[allMaterials[j].URL] {
					allMaterials[i], allMaterials[j] = allMaterials[j], allMaterials[i]
				}
			}
		}
		chaptersBuilder.WriteString(fmt.Sprintf("## %s\n\n", i18n.References))
		for _, m := range allMaterials {
			chaptersBuilder.WriteString(formatCitation(indexMap[m.URL], m.Title, m.URL) + "\n")
		}
		chaptersBuilder.WriteString("\n")
	}
	chapters := chaptersBuilder.String()

	// Body excludes the H1 title and the manual TOC; picoloom renders its own
	// cover and table of contents for PDF output.
	body := summary + chapters

	// Generate the report title from the assembled content.
	s.ReportTitle = s.generateReportTitle(ctx, llm, body)

	// Build the full markdown report with an H1 title and a manual TOC for
	// HTML/Markdown output.
	var reportBuilder strings.Builder
	reportBuilder.WriteString(fmt.Sprintf("# %s\n\n", s.ReportTitle))
	reportBuilder.WriteString(summary)
	reportBuilder.WriteString(fmt.Sprintf("## %s\n\n", i18n.TableOfContents))
	for i, chapter := range s.ChapterOutline {
		reportBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, chapter.Title))
	}
	reportBuilder.WriteString("\n---\n\n")
	reportBuilder.WriteString(chapters)
	s.MarkdownReport = reportBuilder.String()

	// DEBUG: write raw markdown to a temp file for inspection.
	if tmpFile, err := os.CreateTemp("", "deep_research_*.md"); err == nil {
		_, _ = tmpFile.WriteString(s.MarkdownReport)
		_ = tmpFile.Close()
		fmt.Printf("DBG markdown report written to %s\n\n", tmpFile.Name())
	}

	switch s.Config.ReportFormat {
	case "html":
		// Convert to HTML for final report
		extensions := parser.CommonExtensions | parser.AutoHeadingIDs
		p := parser.NewWithExtensions(extensions)
		doc := p.Parse([]byte(s.MarkdownReport))

		htmlFlags := html.CommonFlags | html.HrefTargetBlank
		opts := html.RendererOptions{Flags: htmlFlags}
		renderer := html.NewRenderer(opts)
		s.HTMLReport = string(markdown.Render(doc, renderer))
	case "pdf":
		s.PDFReport = renderPDF(ctx, body, s.ReportTitle, i18n.TableOfContents)
	}

	return s, nil
}

// reportI18n holds UI strings for generated report sections.
type reportI18n struct {
	Summary          string
	TableOfContents  string
	References       string
	FallbackIntro    string
	GenerationFailed string // printf format — must contain %s for title and %v for error
}

// getReportI18n returns UI strings for the primary language subtag derived
// from a BCP 47 tag (e.g. "zh-CN" → "zh"). Defaults to English.
func getReportI18n(lang string) reportI18n {
	primary := strings.ToLower(lang)
	if idx := strings.IndexByte(primary, '-'); idx >= 0 {
		primary = primary[:idx]
	}
	switch primary {
	case "zh":
		return reportI18n{
			Summary:          "摘要",
			TableOfContents:  "目录",
			References:       "参考文献",
			FallbackIntro:    "本研究按照系统性分析方法，对主题进行了深入调研。",
			GenerationFailed: "'%s' 内容生成失败：%v",
		}
	default:
		return reportI18n{
			Summary:          "Summary",
			TableOfContents:  "Table of Contents",
			References:       "References",
			FallbackIntro:    "This research conducted a systematic and in-depth analysis of the topic.",
			GenerationFailed: "Content generation failed for '%s': %v",
		}
	}
}

// reportLang returns the configured report language, defaulting to en-US when
// unset. This mirrors the fallback used throughout the other node functions.
func reportLang(lang string) string {
	if lang == "" {
		return "en-US"
	}
	return lang
}

// helper function to convert markdown content to a PDF byte slice using picoloom.
const deepResearchBrowserRevision = 1625079

func renderPDF(ctx context.Context, markdown string, title string, tocTitle string) []byte {
	conv, err := picoloom.NewConverter(
		picoloom.WithBrowserRevision(deepResearchBrowserRevision),
		picoloom.WithStyle("academic"),
		picoloom.WithKaTeXPath("./config/katex"),
	)
	if err != nil {
		log.Errorf("failed to create PDF converter: %v", err)
		return nil
	}
	defer conv.Close()

	result, err := conv.Convert(ctx, picoloom.Input{
		Markdown: markdown,
		Cover: &picoloom.Cover{
			Title: title,
			Date:  time.Now().Format("2006-01-02"),
		},
		TOC: &picoloom.TOC{
			Title:    tocTitle,
			NoNumber: true,
		},
		Footer: &picoloom.Footer{
			Position:       "right",
			ShowPageNumber: true,
		},
	})
	if err != nil {
		log.Errorf("failed to render PDF: %v", err)
		return nil
	}
	return result.PDF
}

// Utility functions
func containsInt(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// formatCitation returns a Markdown link citation for a reference.
func formatCitation(n int, title, url string) string {
	if url != "" {
		return fmt.Sprintf("[%d] [%s](%s)", n, title, url)
	}
	return fmt.Sprintf("[%d] %s", n, title)
}

// GetChapterPreview returns a preview of chapter content
func (s *State) GetChapterPreview(ctx context.Context, chapterID string) (*ChapterContent, error) {
	if content, exists := s.ChapterContents[chapterID]; exists {
		return content, nil
	}
	return nil, fmt.Errorf("chapter %s not found", chapterID)
}

// GetAllChaptersPreview returns preview for all chapters
func (s *State) GetAllChaptersPreview() map[string]*ChapterContent {
	return s.ChapterContents
}

// GetChapterMaterials returns materials for specific chapter
func (s *State) GetChapterMaterials(chapterID string) []MaterialReference {
	if content, exists := s.ChapterContents[chapterID]; exists {
		return content.Materials
	}
	return []MaterialReference{}
}
