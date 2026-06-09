package deep_research

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
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

	// Prepend uploaded document content so the planner can treat them as primary source material.
	if len(s.Attachments) > 0 {
		var sb strings.Builder
		for i, a := range s.Attachments {
			if a == nil {
				continue
			}
			text := strings.TrimSpace(a.Text)
			if text == "" {
				continue
			}
			name := a.Name
			if name == "" {
				name = a.ID
			}
			sb.WriteString(fmt.Sprintf("[%d] %s:\n%s\n\n", i+1, name, util.SubString(text, 0, 2048)))
		}
		if sb.Len() > 0 {
			prompt = fmt.Sprintf("The user has uploaded the following documents. Treat them as primary source material when forming the research plan:\n<attachments>\n%s</attachments>\n\n%s", sb.String(), prompt)
		}
	}

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
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

			refinementQuery, err := llms.GenerateFromSinglePrompt(ctx, llm, refinementPrompt)
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

		completion, err := llms.GenerateFromSinglePrompt(ctx, synthesisLLM, findingsPrompt)
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

	// Strategy: Use the structured chapter approach if we have organized materials
	if len(s.ChapterOutline) > 0 && len(s.ChapterContents) > 0 {
		return s.generateChapterBasedReport(ctx, llm)
	}

	// Fallback: Use traditional approach
	return s.generateTraditionalReport(ctx, llm)
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
2. Chapters should progress logically, from basic to in-depth
3. Use %s for titles and descriptions
4. Accurately relate to relevant research steps`,
		s.Request.Query,
		strings.Join(s.Plan, "\n"),
		reportLang(s.Config.ReportLang))

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
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

// generateTraditionalReport provides fallback report generation when chapter structure is not available
func (s *State) generateTraditionalReport(ctx context.Context, llm llms.Model) (*State, error) {
	researchData := strings.Join(s.ResearchResults, "\n\n")

	prompt := fmt.Sprintf("You are a senior report writer. Based on the following research results, write a comprehensive final report. Use Markdown format. Write the report in %s:\n\n%s\n\nOriginal query was: %s",
		reportLang(s.Config.ReportLang), researchData, s.Request.Query)

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, err
	}

	// Clean up response
	completion = strings.TrimSpace(completion)
	completion = strings.TrimPrefix(completion, "```markdown")
	completion = strings.TrimPrefix(completion, "```")
	completion = strings.TrimSuffix(completion, "```")

	s.MarkdownReport = completion

	// Convert to HTML
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(completion))

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	s.FinalReport = string(markdown.Render(doc, renderer))

	return s, nil
}

// generateChapterContent generates comprehensive content for each chapter using allocated materials
func (s *State) generateChapterContent(ctx context.Context, llm llms.Model) map[string]*ChapterContent {
	log.Info("Starting chapter content generation...")

	for chapterID, content := range s.ChapterContents {
		content.Status = "generating"

		i18n := getReportI18n(s.Config.ReportLang)
		if len(content.Materials) == 0 {
			content.Content = fmt.Sprintf("# %s\n\n%s", content.Title, i18n.NoMaterials)
			content.Status = "completed"
			s.ChapterContents[chapterID] = content
			continue
		}

		// Build comprehensive material reference for this chapter
		materialsInfo := s.buildMaterialsInfo(content.Materials)
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

Generate the chapter content directly, do not add explanatory text.`,
			s.Request.Query,
			content.Title,
			materialsInfo,
			reportLang(s.Config.ReportLang))

		completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
		if err != nil {
			log.Warnf("Failed to generate chapter content for %s: %v", chapterID, err)
			content.Content = fmt.Sprintf("# %s\n\n"+i18n.GenerationFailed, content.Title, err)
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

// buildMaterialsInfo constructs a comprehensive description of all materials in a chapter
func (s *State) buildMaterialsInfo(materials []MaterialReference) string {
	var builder strings.Builder

	builder.WriteString("## Research Materials Summary\n\n")
	for i, material := range materials {
		if i > 10 { // Limit for clarity
			builder.WriteString(fmt.Sprintf("\n... and %d other related materials\n", len(materials)-10))
			break
		}

		builder.WriteString(fmt.Sprintf("### Material[%d]: %s\n", i+1, material.Title))
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

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
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

// generateChapterBasedReport creates a structured report based on the chapter outline and generated content
func (s *State) generateChapterBasedReport(ctx context.Context, llm llms.Model) (*State, error) {

	// Debug output for chapters

	contents := s.ChapterContents
	var reportBuilder strings.Builder

	// Title
	reportBuilder.WriteString(fmt.Sprintf("# %s\n\n", s.Request.Query))

	// Summary section
	i18n := getReportI18n(s.Config.ReportLang)
	reportBuilder.WriteString(fmt.Sprintf("## %s\n\n", i18n.Summary))
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
			reportBuilder.WriteString(fmt.Sprintf("- %s\n", point))
		}
		reportBuilder.WriteString("\n")
	} else {
		reportBuilder.WriteString(i18n.FallbackIntro + "\n\n")
	}
	reportBuilder.WriteString(fmt.Sprintf("## %s\n\n", i18n.TableOfContents))
	for _, chapter := range s.ChapterOutline {
		reportBuilder.WriteString(fmt.Sprintf("%d. [%s](#%s)\n", len(s.ChapterOutline), chapter.Title, strings.ToLower(strings.ReplaceAll(chapter.Title, " ", "-"))))
	}
	reportBuilder.WriteString("\n---\n\n")

	// Step 1: Check if we need to generate chapter content
	needsGeneration := false
	for _, chapter := range s.ChapterOutline {
		if content, exists := contents[chapter.ID]; exists {
			if content.Content == "" {
				needsGeneration = true
				break
			}
		}
	}

	// Generate chapter content if needed
	if needsGeneration {
		contents = s.generateChapterContent(ctx, llm)
	}

	// Step 2: Generate each chapter content
	for _, chapter := range s.ChapterOutline {
		if content, exists := contents[chapter.ID]; exists && content.Content != "" {

			reportBuilder.WriteString(fmt.Sprintf("## %s\n\n", chapter.Title))
			reportBuilder.WriteString(content.Content + "\n\n")

			// Add references or source citations if configured
			if s.Config != nil && s.Config.IncludeSources {
				if len(content.Materials) > 0 {
					reportBuilder.WriteString("### References\n\n")
					for j, material := range content.Materials {
						reportBuilder.WriteString(formatCitation(j+1, material.Title, material.URL, material.Source, s.Config.SourceFormat) + "\n")
					}
					reportBuilder.WriteString("\n")
				}
			}
		}
	}

	// Set the markdown report
	s.MarkdownReport = reportBuilder.String()

	// Convert to HTML for final report
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(s.MarkdownReport))

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	s.FinalReport = string(markdown.Render(doc, renderer))

	return s, nil
}

// reportI18n holds UI strings for generated report sections.
type reportI18n struct {
	Summary          string
	TableOfContents  string
	FallbackIntro    string
	NoMaterials      string
	GenerationFailed string // printf format — must contain one %v
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
			FallbackIntro:    "本研究按照系统性分析方法，对主题进行了深入调研。",
			NoMaterials:      "暂无可用的研究素材。",
			GenerationFailed: "内容生成失败：%v",
		}
	default:
		return reportI18n{
			Summary:          "Summary",
			TableOfContents:  "Table of Contents",
			FallbackIntro:    "This research conducted a systematic and in-depth analysis of the topic.",
			NoMaterials:      "No research materials available.",
			GenerationFailed: "Content generation failed: %v",
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

// formatCitation formats a reference citation based on the requested style.
// Supports "APA" and "MLA"; defaults to Markdown link format.
func formatCitation(n int, title, url, source, format string) string {
	switch strings.ToUpper(format) {
	case "APA":
		if url != "" {
			return fmt.Sprintf("[%d] %s. *%s*. %s", n, title, source, url)
		}
		return fmt.Sprintf("[%d] %s. *%s*.", n, title, source)
	case "MLA":
		if url != "" {
			return fmt.Sprintf("[%d] \"%s.\" *%s*, %s.", n, title, source, url)
		}
		return fmt.Sprintf("[%d] \"%s.\" *%s*.", n, title, source)
	default:
		if url != "" {
			return fmt.Sprintf("[%d] [%s](%s)", n, title, url)
		}
		return fmt.Sprintf("[%d] %s", n, title)
	}
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
