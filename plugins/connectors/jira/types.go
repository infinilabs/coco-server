package jira  
  
import "time"  
  
// Config 定义 Jira 连接器配置  
type Config struct {  
	BaseURL           string   `config:"base_url" json:"base_url"`  
	AuthType          string   `config:"auth_type" json:"auth_type"` // basic_auth, api_token, oauth  
	Username          string   `config:"username" json:"username"`  
	Password          string   `config:"password" json:"password"`  
	APIToken          string   `config:"api_token" json:"api_token"`  
	Projects          []string `config:"projects" json:"projects"`  
	IssueTypes        []string `config:"issue_types" json:"issue_types"`  
	Fields            []string `config:"fields" json:"fields"`  
	IncludeComments   bool     `config:"include_comments" json:"include_comments"`  
	IncludeAttachments bool    `config:"include_attachments" json:"include_attachments"`  
	MaxResults        int      `config:"max_results" json:"max_results"`  
	JQLFilter         string   `config:"jql_filter" json:"jql_filter"`  
}  
  
// SearchResult Jira 搜索结果  
type SearchResult struct {  
	Expand     string  `json:"expand"`  
	StartAt    int     `json:"startAt"`  
	MaxResults int     `json:"maxResults"`  
	Total      int     `json:"total"`  
	Issues     []Issue `json:"issues"`  
}  
  
// Issue Jira 问题  
type Issue struct {  
	Expand string      `json:"expand"`  
	ID     string      `json:"id"`  
	Self   string      `json:"self"`  
	Key    string      `json:"key"`  
	Fields IssueFields `json:"fields"`  
}  
  
// IssueFields 问题字段  
type IssueFields struct {  
	Summary     string      `json:"summary"`  
	Description string      `json:"description"`  
	IssueType   *IssueType  `json:"issuetype"`  
	Project     *Project    `json:"project"`  
	Status      *Status     `json:"status"`  
	Priority    *Priority   `json:"priority"`  
	Reporter    *User       `json:"reporter"`  
	Assignee    *User       `json:"assignee"`  
	Created     *time.Time  `json:"created"`  
	Updated     *time.Time  `json:"updated"`  
	Labels      []string    `json:"labels"`  
	Components  []Component `json:"components"`  
}  
  
// IssueType 问题类型  
type IssueType struct {  
	Self        string `json:"self"`  
	ID          string `json:"id"`  
	Description string `json:"description"`  
	IconURL     string `json:"iconUrl"`  
	Name        string `json:"name"`  
	Subtask     bool   `json:"subtask"`  
}  
  
// Project 项目  
type Project struct {  
	Self       string `json:"self"`  
	ID         string `json:"id"`  
	Key        string `json:"key"`  
	Name       string `json:"name"`  
	ProjectTypeKey string `json:"projectTypeKey"`  
}  
  
// Status 状态  
type Status struct {  
	Self           string `json:"self"`  
	Description    string `json:"description"`  
	IconURL        string `json:"iconUrl"`  
	Name           string `json:"name"`  
	ID             string `json:"id"`  
	StatusCategory StatusCategory `json:"statusCategory"`  
}  
  
// StatusCategory 状态分类  
type StatusCategory struct {  
	Self      string `json:"self"`  
	ID        int    `json:"id"`  
	Key       string `json:"key"`  
	ColorName string `json:"colorName"`  
	Name      string `json:"name"`  
}  

// Priority 优先级  
type Priority struct {  
	Self    string `json:"self"`  
	IconURL string `json:"iconUrl"`  
	Name    string `json:"name"`  
	ID      string `json:"id"`  
}  
  
// User 用户  
type User struct {  
	Self         string `json:"self"`  
	AccountID    string `json:"accountId"`  
	DisplayName  string `json:"displayName"`  
	EmailAddress string `json:"emailAddress"`  
	Active       bool   `json:"active"`  
	TimeZone     string `json:"timeZone"`  
	AccountType  string `json:"accountType"`  
}  
  
// Component 组件  
type Component struct {  
	Self        string `json:"self"`  
	ID          string `json:"id"`  
	Name        string `json:"name"`  
	Description string `json:"description"`  
}  
  
// CommentsResponse 评论响应  
type CommentsResponse struct {  
	StartAt    int       `json:"startAt"`  
	MaxResults int       `json:"maxResults"`  
	Total      int       `json:"total"`  
	Comments   []Comment `json:"comments"`  
}  
  
// Comment 评论  
type Comment struct {  
	Self     string     `json:"self"`  
	ID       string     `json:"id"`  
	Author   *User      `json:"author"`  
	Body     string     `json:"body"`  
	Created  *time.Time `json:"created"`  
	Updated  *time.Time `json:"updated"`  
	IssueKey string     `json:"-"` // 手动设置，用于关联问题  
}