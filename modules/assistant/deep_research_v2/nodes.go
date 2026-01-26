package deep_research

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

// PlannerNode generates a research plan based on the query.
func PlannerNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)

	state.(*State).Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchPlannerStart, "", 0)

	llm, err := getLLM()
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`你是一名研究规划师。请为以下查询创建一个分步研究计划：%s。
同时，请判断用户是否希望同时生成播客（Podcast）脚本（例如查询中包含"播客"、"podcast"、"对话"、"脚本"等意图，或者用户明确要求生成播客）。
请以 JSON 格式返回结果，格式如下：
{
    "plan": ["步骤1", "步骤2", ...],
    "generate_podcast": true/false
}
必须使用中文回复。`, s.Request.Query)

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
		Plan            []string `json:"plan"`
		GeneratePodcast bool     `json:"generate_podcast"`
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
		// Default to false if JSON parsing fails, unless we find keywords in query
		queryLower := strings.ToLower(s.Request.Query)
		s.GeneratePodcast = strings.Contains(queryLower, "播客") || strings.Contains(queryLower, "podcast")
	} else {
		s.Plan = output.Plan
		s.GeneratePodcast = output.GeneratePodcast
	}

	state.(*State).Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchPlannerEnd, util.MustToJSON(s.Plan), 0)

	return s, nil
}

// ResearcherNode executes the research plan using real search with feedback mechanism and chapter management.
func ResearcherNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)

	llm, err := getLLM()
	if err != nil {
		return nil, err
	}

	// Initialize state components if not present
	if s.StartTime == 0 {
		s.StartTime = time.Now().Unix()
		// Initialize material management
		//s.AllMaterials = []MaterialReference{}
		s.MaterialRegistry = make(map[string]bool)
		// Initialize chapter contents
		s.ChapterContents = make(map[string]*ChapterContent)
	}

	// Initialize search manager if not already present
	if s.SearchManager == nil {
		tavilyAPIKey := "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" // Hardcoded test key as fallback
		s.SearchManager = NewSearchToolManager(tavilyAPIKey)
	}

	// Initialize chapter outline if not present
	if len(s.ChapterOutline) == 0 {
		err := s.generateChapterOutline(ctx)
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
	var allImages []string

	// Process each research step with feedback-based search and chapter distribution
	for stepIndex, step := range s.Plan {
		stepStartTime := time.Now()

		payload := util.MapStr{}
		payload["plan"] = step
		state.(*State).Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchResearcherStart, util.MustToJSON(payload), 0)

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
			"name": "搜索资料",
			"payload": util.MapStr{
				"from":  0,
				"size":  10,
				"query": query,
			},
		} //TODO, convert to query
		state.(*State).Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchResearcherStepStart, util.MustToJSON(searchPayload), 0)

		// Step 1: Initial search for this research step
		initialSearchCollection, err := s.SearchManager.SearchWithFeedback(ctx, step, true) // internal first
		if err != nil {
			log.Warnf("Initial search failed for step '%s': %v", step, err)
			defErrorResult := fmt.Sprintf("Step: %s\nFindings: 搜索失败：%v", step, err)
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
			"name": "搜索资料",
			"payload": util.MapStr{
				"total": 10,
				"hits":  initialSearchCollection.Results,
			},
		} //TODO, convert to query
		state.(*State).Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchResearcherStepEnd, util.MustToJSON(searchPayload), 0)

		// Step 2: Analyze search results and potentially refine search
		if !initialSearchCollection.IsSufficient {

			// Generate refinement query based on initial results analysis
			refinementPrompt := fmt.Sprintf(`基于以下搜索结果分析，为这个研究步骤生成一个更具体的搜索查询：

研究步骤：%s
初始搜索结果：%s
当前置信度：%.2f%%

请生成一个更具体的搜索查询来获得更详细或相关的信息。只返回搜索查询，不要其他内容。`,
				step,
				initialSearchCollection.FormatResultsForLLM(),
				initialSearchCollection.Confidence*100)

			refinementQuery, err := llms.GenerateFromSinglePrompt(ctx, llm, refinementPrompt)
			if err == nil && strings.TrimSpace(refinementQuery) != "" {
				stepSearchQueries = append(stepSearchQueries, refinementQuery)

				// Perform refined search
				refinedCollection, err := s.SearchManager.SearchWithFeedback(ctx, refinementQuery, false) // external search
				if err == nil {
					// Combine initial and refined results
					initialSearchCollection.Results = append(initialSearchCollection.Results, refinedCollection.Results...)
					initialSearchCollection.evaluateSearchQuality() // Re-evaluate
				}
			}
		}

		stepSearchQueries = append(stepSearchQueries, step)

		// Step 3: Convert search results to material references and allocate to chapters
		stepMaterials = s.convertToMaterials(initialSearchCollection, stepIndex+1)

		// Step 4: Distribute materials to relevant chapters
		allocatedMaterials := s.distributeMaterialsToChapters(stepMaterials, stepIndex)

		// Step 5: Collect images from search results (extract from URLs or content)
		imagesInThisStep := extractImageURLs(initialSearchCollection)
		allImages = append(allImages, imagesInThisStep...)

		// Step 6: Analyze and synthesize findings with chapter-aware context
		findingsPrompt := s.generateChapterAwareAnalysisPrompt(step, allocatedMaterials, stepIndex)

		completion, err := llms.GenerateFromSinglePrompt(ctx, llm, findingsPrompt)
		if err != nil {
			log.Warnf("Analysis failed for step '%s': %v", step, err)
			errorResult := fmt.Sprintf("Step: %s\nFindings: 分析失败：%v\n\n搜索结果：%s",
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
				s.StepResults[stepIndex].Images = imagesInThisStep
				s.StepResults[stepIndex].ProcessingTime = time.Since(stepStartTime).String()
			}
		}

		// Step 7: Update chapter progress
		s.updateChapterProgress(stepIndex, allocatedMaterials)

		state.(*State).Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchResearcherEnd, util.MustToJSON(payload), 0)
	}

	s.ResearchResults = results
	s.Images = allImages

	return s, nil
}

// Replace image placeholders with actual image tags
// Regex matches [IMAGE_X：Title] or [IMAGE_X:Title]
var imgRe = regexp.MustCompile(`\[IMAGE_(\d+)[：:]([^\]]+)\]`)

// ReporterNode compiles the final report using organized chapter structure and materials.
func ReporterNode(ctx context.Context, state interface{}) (interface{}, error) {

	state.(*State).Sender.SendChunkMessage(core.MessageTypeAssistant, common.ResearchReporterStart, "", 0)

	s := state.(*State)

	llm, err := getLLM()
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

// PodcastNode generates a podcast script based on the research results.
func PodcastNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)

	llm, err := getLLM()
	if err != nil {
		return nil, err
	}

	researchData := strings.Join(s.ResearchResults, "\n\n")
	prompt := fmt.Sprintf(`你是一名专业的播客制作人。请根据以下研究结果，创作一段引人入胜的播客对话脚本。
对话应该由两名主持人（Host 1 和 Host 2）进行，风格轻松幽默，通俗易懂。
请深入讨论研究结果中的关键点，并加入一些生动的例子或类比。

请以 JSON 格式返回结果，格式如下：
{
    "title": "播客标题",
    "lines": [
        {"speaker": "Host 1", "content": "对话内容..."},
        {"speaker": "Host 2", "content": "对话内容..."}
    ]
}

研究结果：
%s

原始查询：%s
必须使用中文创作。`, researchData, s.Request.Query)

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

	var script struct {
		Title string `json:"title"`
		Lines []struct {
			Speaker string `json:"speaker"`
			Content string `json:"content"`
		} `json:"lines"`
	}

	if err := json.Unmarshal([]byte(completion), &script); err != nil {
		s.PodcastScript = fmt.Sprintf("<pre>%s</pre>", completion)
		return s, nil
	}

	// Serialize script back to JSON for export
	jsonBytes, _ := json.Marshal(script)
	jsonString := string(jsonBytes)
	jsonString = strings.ReplaceAll(jsonString, "</div>", "<\\/div>") // Escape for HTML embedding

	// Render HTML
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`
<div class="podcast-container" style="max-width: 800px; margin: 0 auto; font-family: 'Inter', sans-serif;">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
        <h2 style="margin: 0;">%s</h2>
        <button onclick="window.exportPodcastJson()" style="background-color: #28a745; color: white; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; display: flex; align-items: center; gap: 5px;">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
            导出 JSON 脚本
        </button>
    </div>
    <div id="podcastJsonData" style="display:none">%s</div>
`, script.Title, jsonString))

	for _, line := range script.Lines {
		speakerClass := "host-1"
		bgColor := "#e6f7ff"
		borderColor := "#1890ff"
		textColor := "#0050b3"

		if strings.Contains(strings.ToLower(line.Speaker), "2") {
			speakerClass = "host-2"
			bgColor = "#fff0f6"
			borderColor = "#eb2f96"
			textColor = "#9e1068"
		}

		sb.WriteString(fmt.Sprintf(`
    <div class="podcast-message %s" style="margin-bottom: 20px; padding: 20px; border-radius: 8px; border-left: 5px solid %s; background-color: %s; box-shadow: 0 2px 5px rgba(0,0,0,0.05);">
        <div class="speaker-name" style="font-weight: 700; margin-bottom: 8px; color: %s; text-transform: uppercase; letter-spacing: 0.5px;">%s</div>
        <div class="message-content" style="line-height: 1.6; color: #333; font-size: 16px;">%s</div>
    </div>
`, speakerClass, borderColor, bgColor, textColor, line.Speaker, line.Content))
	}

	sb.WriteString("</div>")

	s.PodcastScript = sb.String()
	return s, nil
}

func getLLM() (llms.Model, error) {
	// Use DeepSeek as per user preference
	// Ensure OPENAI_API_KEY and OPENAI_API_BASE are set in the environment
	return openai.New()
}

// extractImageURLs extracts image URLs from search results
func extractImageURLs(collection *SearchResultCollection) []string {
	var images []string
	imageExtensions := []string{".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp"}

	for _, result := range collection.Results {
		// Check URL field for image URLs
		if result.URL != "" {
			parsedURL, err := url.Parse(result.URL)
			if err == nil {
				for _, ext := range imageExtensions {
					if strings.HasSuffix(strings.ToLower(parsedURL.Path), ext) {
						images = append(images, result.URL)
						break
					}
				}
			}
		}

		// Check content field for image URLs using regex
		imgRegex := regexp.MustCompile(`https?://[^\s]+\.(?:jpg|jpeg|png|gif|svg|webp)`)
		if matches := imgRegex.FindAllString(result.Content, -1); len(matches) > 0 {
			images = append(images, matches...)
		}
	}

	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	var uniqueImages []string
	for _, img := range images {
		if !seen[img] {
			seen[img] = true
			uniqueImages = append(uniqueImages, img)
		}
	}

	return uniqueImages
}

// generateChapterOutline creates an intelligent chapter structure for the report
func (s *State) generateChapterOutline(ctx context.Context) error {
	if len(s.Plan) == 0 {
		return fmt.Errorf("no research plan available")
	}

	llm, err := getLLM()
	if err != nil {
		return err
	}

	prompt := fmt.Sprintf(`基于以下研究查询和研究计划，生成一份详细的报告章节大纲。章节应该逻辑清晰，覆盖研究的所有重要方面，每个章节都要有明确的重点和相关关键词。

研究查询：%s
研究计划：
%s

请生成 JSON 格式的章节大纲，格式如下：
[
  {
    "id": "chapter_1",
    "title": "章节标题",
    "description": "章节内容描述",
    "priority": 5,  // 1-5 重要程度
    "keywords": ["关键词1", "关键词2"],
    "related_steps": [1, 2, 3]  // 相关的研究步骤编号
  }
]

要求：
1. 生成 4-8 个章节
2. 章节逻辑递进，从基础到深入
3. 使用中文标题和描述
4. 准确关联相关的研究步骤`,
		s.Request.Query,
		strings.Join(s.Plan, "\n"))

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

	log.Info("生成报告章节：", util.ToJson(chapters, true))

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
		materialsInfo = fmt.Sprintf("\n已分配素材：\n")
		for _, material := range allocatedMaterials {
			materialsInfo += fmt.Sprintf("- %s (%s)\n", material.Title, material.Summary[:min(len(material.Summary), 100)])
		}
	}

	return fmt.Sprintf(`你是一名研究员。请基于以下搜索结果，为这个研究步骤提供详细发现和洞察。

研究步骤：%s
搜索结果详细信息：%s
%s

要求：
1. 提供详细的发现和洞察
2. 如果搜索结果不足，请明确指出
3. 使用中文回复
4. 按重要程度组织内容
5. 结合已分配的章节素材进行分析`,
		step,
		"已搜索到的结果", // For preview, not actual search results - replaced direct call
		materialsInfo)
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

	imageInfo := ""
	if len(s.Images) > 0 {
		imageInfo = fmt.Sprintf("\n\n注意：研究过程中收集到 %d 张相关图片。在报告中适当的位置，你可以使用 [IMAGE_X：图片标题] 占位符来标记应该插入图片的位置（X 为 1 到 %d）。", len(s.Images), len(s.Images))
	}

	prompt := fmt.Sprintf("你是一名资深报告撰写员。请根据以下研究结果撰写一份全面的最终报告。使用 Markdown 格式。%s必须使用中文撰写报告：\n\n%s\n\n原始查询是：%s",
		imageInfo, researchData, s.Request.Query)

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

	// Process image placeholders
	s.FinalReport = imgRe.ReplaceAllStringFunc(s.FinalReport, func(match string) string {
		parts := imgRe.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		idxStr := parts[1]
		title := strings.TrimSpace(parts[2])

		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 1 || idx > len(s.Images) {
			return match
		}

		imgURL := s.Images[idx-1]
		return fmt.Sprintf(`<img src="%s" alt="%s" style="max-width: 90%%; display: block; margin: 10px auto;" />`,
			imgURL, title)
	})

	return s, nil
}

// generateChapterContent generates comprehensive content for each chapter using allocated materials
func (s *State) generateChapterContent(ctx context.Context, llm llms.Model) map[string]*ChapterContent {
	log.Info("Starting chapter content generation...")

	for chapterID, content := range s.ChapterContents {
		content.Status = "generating"

		if len(content.Materials) == 0 {
			content.Content = fmt.Sprintf("# %s\n\n暂无可用的研究素材。", content.Title)
			content.Status = "completed"
			s.ChapterContents[chapterID] = content
			continue
		}

		// Build comprehensive material reference for this chapter
		materialsInfo := s.buildMaterialsInfo(content.Materials)
		log.Infof("Generating content for chapter %s with %d materials", content.Title, len(content.Materials))

		// Generate comprehensive chapter content from materials
		prompt := fmt.Sprintf(`你是专业的报告撰写员。

研究主题：%s
章节标题：%s

基于以下经过智能分类的研究素材，为这个章节撰写详细的专业报告：

%s

要求：
1. 内容必须基于提供的素材，不可加入虚构信息
2. 保持学术性和专业性风格
3. 整合所有相关素材，深度分析洞察
4. 使用清晰的Markdown格式，合理的层次结构
5. 按照逻辑结构组织内容，从基础到深入
6. 在每个重要观点后标注相关的素材来源（如[1]、[2]等）
7. 字数控制在1000-2000字
8. 使用中文撰写

直接生成章节正文，不要添加额外的说明文字。`,
			s.Request.Query,
			content.Title,
			materialsInfo)

		completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
		if err != nil {
			log.Warnf("Failed to generate chapter content for %s: %v", chapterID, err)
			content.Content = fmt.Sprintf("# %s\n\n内容生成失败：%v", content.Title, err)
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

	builder.WriteString("## 研究素材汇总\n\n")
	for i, material := range materials {
		if i > 10 { // Limit for clarity
			builder.WriteString(fmt.Sprintf("\n... 以及 %d 个其他相关素材\n", len(materials)-10))
			break
		}

		builder.WriteString(fmt.Sprintf("### 素材[%d]: %s\n", i+1, material.Title))
		builder.WriteString(fmt.Sprintf("- 来源: %s (相关度: %.2f)\n", material.Source, material.Relevance))
		if material.URL != "" {
			builder.WriteString(fmt.Sprintf("- 链接: %s\n", material.URL))
		}
		builder.WriteString(fmt.Sprintf("- 内容摘要: %s\n", material.Summary))
		builder.WriteString("\n")
	}
	return builder.String()
}

// extractKeyPoints extracts the main points from generated content
func (s *State) extractKeyPoints(ctx context.Context, llm llms.Model, content string) ([]string, error) {
	prompt := fmt.Sprintf(`请从以下内容中提取3-5个最重要的观点/要点，每个要点简短明了：

内容:
%s

返回格式：
- 要点1
- 要点2
- 要点3...

不要添加其他文字`, content)

	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return []string{"内容分析要点"}, err
	}

	lines := strings.Split(completion, "\n")
	var keyPoints []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") {
			keyPoints = append(keyPoints, strings.TrimPrefix(line, "-"))
		}
	}

	if len(keyPoints) == 0 {
		keyPoints = []string{"相关内容要点"}
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
	reportBuilder.WriteString("## 摘要\n\n")
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
		reportBuilder.WriteString("本研究按照系统性分析方法，对主题进行了深入调研。\n\n")
	}
	reportBuilder.WriteString("## 目录\n\n")
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
					reportBuilder.WriteString("### 参考资料\n\n")
					for j, material := range content.Materials {
						if material.URL != "" {
							reportBuilder.WriteString(fmt.Sprintf("[%d] [%s](%s)\n", j+1, material.Title, material.URL))
						} else {
							reportBuilder.WriteString(fmt.Sprintf("[%d] %s\n", j+1, material.Title))
						}
					}
					reportBuilder.WriteString("\n")
				}
			}
		}
	}

	// Add collected images at the end
	if len(s.Images) > 0 {
		reportBuilder.WriteString("## 相关图片\n\n")
		for i, imgURL := range s.Images {
			reportBuilder.WriteString(fmt.Sprintf("<img src=\"%s\" alt=\"图片 %d\" style=\"max-width: 90%%; display: block; margin: 10px auto;\" />\n\n", imgURL, i+1))
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

// GetChapterPreview returns a preview of chapter content
func (s *State) GetChapterPreview(ctx context.Context, chapterID string) (*ChapterContent, error) {
	if content, exists := s.ChapterContents[chapterID]; exists {
		return content, nil
	}
	return nil, fmt.Errorf("chapter %s not found", chapterID)
}

// getChapterDescription gets chapter description from outline
func (s *State) getChapterDescription(chapterID string) string {
	for _, chapter := range s.ChapterOutline {
		if chapter.ID == chapterID {
			return chapter.Description
		}
	}
	return ""
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
