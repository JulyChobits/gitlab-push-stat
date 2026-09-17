package gitlab

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client GitLab API客户端
type Client struct {
	baseURL    string
	token      string
	httpClient *resty.Client
}

// Project GitLab项目信息
type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	DefaultBranch     string `json:"default_branch"`
}

// Commit GitLab提交信息
type Commit struct {
	ID            string    `json:"id"`
	ShortID       string    `json:"short_id"`
	Title         string    `json:"title"`
	Message       string    `json:"message"`
	AuthorName    string    `json:"author_name"`
	AuthorEmail   string    `json:"author_email"`
	CommittedDate time.Time `json:"committed_date"`
	CreatedAt     time.Time `json:"created_at"`
	ParentIDs     []string  `json:"parent_ids"`
	Stats         *Stats    `json:"stats,omitempty"`
}

// Stats 提交统计信息
type Stats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

// Diff 提交差异信息
type Diff struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
	Diff        string `json:"diff"`
}

// User GitLab用户信息
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Event GitLab事件信息
type Event struct {
	ID         int         `json:"id"`
	ActionName string      `json:"action_name"`
	Author     EventAuthor `json:"author"`
	PushData   *PushData   `json:"push_data"`
	CreatedAt  time.Time   `json:"created_at"`
}

// EventAuthor 事件作者信息
type EventAuthor struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	State    string `json:"state"`
}

// PushData Push事件数据
type PushData struct {
	CommitCount int    `json:"commit_count"`
	Action      string `json:"action"`
	RefType     string `json:"ref_type"`
	CommitFrom  string `json:"commit_from"`
	CommitTo    string `json:"commit_to"`
	Ref         string `json:"ref"`
	CommitTitle string `json:"commit_title"`
}

// CommitDetail 提交详情（包含diff）
type CommitDetail struct {
	Commit
	Diffs []Diff `json:"diffs"`
	Stats Stats  `json:"stats"`
}

// NewClient 创建GitLab客户端
func NewClient(baseURL, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	client := resty.New().
		SetBaseURL(baseURL+"/api/v4").
		SetHeader("PRIVATE-TOKEN", token).
		SetHeader("Content-Type", "application/json").
		SetTimeout(30 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(5 * time.Second)

	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: client,
	}
}

// ListProjects 获取项目列表
func (c *Client) ListProjects(search string, page, perPage int) ([]Project, error) {
	var projects []Project

	resp, err := c.httpClient.R().
		SetQueryParams(map[string]string{
			"search":   search,
			"page":     fmt.Sprintf("%d", page),
			"per_page": fmt.Sprintf("%d", perPage),
			"order_by": "name",
		}).
		SetResult(&projects).
		Get("/projects")

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return projects, nil
}

// GetProject 获取单个项目
func (c *Client) GetProject(projectID int) (*Project, error) {
	var project Project

	resp, err := c.httpClient.R().
		SetResult(&project).
		Get(fmt.Sprintf("/projects/%d", projectID))

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode())
	}

	return &project, nil
}

// ListCommits 获取提交列表
func (c *Client) ListCommits(projectID int, since, until time.Time, page, perPage int) ([]Commit, error) {
	var commits []Commit

	url := fmt.Sprintf("/projects/%d/repository/commits", projectID)
	sinceStr := since.Format(time.RFC3339)
	untilStr := until.Format(time.RFC3339)
	log.Printf("请求GitLab: GET %s?since=%s&until=%s&page=%d&per_page=%d",
		c.baseURL+"/api/v4"+url, sinceStr, untilStr, page, perPage)

	resp, err := c.httpClient.R().
		SetQueryParams(map[string]string{
			"since":      sinceStr,
			"until":      untilStr,
			"page":       fmt.Sprintf("%d", page),
			"per_page":   fmt.Sprintf("%d", perPage),
			"with_stats": "true",
			"all":        "true",
		}).
		SetResult(&commits).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode())
	}

	return commits, nil
}

// GetCommitDetail 获取提交详情（包含diff）
func (c *Client) GetCommitDetail(projectID int, commitSHA string) (*CommitDetail, error) {
	// 1. 先获取commit基本信息
	var commit Commit
	resp1, err := c.httpClient.R().
		SetResult(&commit).
		SetQueryParams(map[string]string{"stats": "true"}).
		Get(fmt.Sprintf("/projects/%d/repository/commits/%s", projectID, commitSHA))

	if err != nil {
		return nil, fmt.Errorf("请求commit详情失败: %w", err)
	}
	if resp1.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d", resp1.StatusCode())
	}

	// 2. 再获取diff列表（该端点返回数组）
	var diffs []Diff
	resp2, err := c.httpClient.R().
		SetResult(&diffs).
		Get(fmt.Sprintf("/projects/%d/repository/commits/%s/diff", projectID, commitSHA))

	if err != nil {
		return nil, fmt.Errorf("请求commit diff失败: %w", err)
	}
	if resp2.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d", resp2.StatusCode())
	}

	detail := &CommitDetail{
		Commit: commit,
		Diffs:  diffs,
	}
	if commit.Stats != nil {
		detail.Stats = *commit.Stats
	}

	log.Printf("获取提交详情: %s, diff文件数: %d", commitSHA[:8], len(diffs))
	return detail, nil
}

// GetCommitStats 获取提交统计
func (c *Client) GetCommitStats(projectID int, commitSHA string) (*Stats, error) {
	var commit struct {
		Stats Stats `json:"stats"`
	}

	resp, err := c.httpClient.R().
		SetResult(&commit).
		SetQueryParams(map[string]string{
			"stats": "true",
		}).
		Get(fmt.Sprintf("/projects/%d/repository/commits/%s", projectID, commitSHA))

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode())
	}

	return &commit.Stats, nil
}

// SearchProjects 搜索项目（支持分页获取全部）
func (c *Client) SearchProjects(search string) ([]Project, error) {
	var allProjects []Project
	page := 1
	perPage := 100

	for {
		projects, err := c.ListProjects(search, page, perPage)
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, projects...)

		if len(projects) < perPage {
			break
		}
		page++
	}

	return allProjects, nil
}

// SearchUserByEmail 通过邮箱搜索GitLab用户
func (c *Client) SearchUserByEmail(email string) (*User, error) {
	var users []User

	resp, err := c.httpClient.R().
		SetQueryParams(map[string]string{
			"search":   email,
			"per_page": "10",
		}).
		SetResult(&users).
		Get("/users")

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode())
	}

	// 精确匹配邮箱
	for _, u := range users {
		if strings.EqualFold(u.Email, email) {
			return &u, nil
		}
	}

	return nil, nil // 未找到
}

// SearchUserByUsername 通过用户名搜索GitLab用户
func (c *Client) SearchUserByUsername(username string) (*User, error) {
	var users []User

	resp, err := c.httpClient.R().
		SetQueryParams(map[string]string{
			"search":   username,
			"per_page": "10",
		}).
		SetResult(&users).
		Get("/users")

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d", resp.StatusCode())
	}

	// 精确匹配用户名
	for _, u := range users {
		if strings.EqualFold(u.Username, username) {
			return &u, nil
		}
	}

	return nil, nil // 未找到
}

// ListProjectEvents 获取项目事件列表（push事件）
func (c *Client) ListProjectEvents(projectID int, after, before string, page, perPage int) ([]Event, error) {
	var events []Event

	url := fmt.Sprintf("/projects/%d/events", projectID)
	log.Printf("请求GitLab Events: GET %s?action=pushed&after=%s&before=%s&page=%d&per_page=%d",
		c.baseURL+"/api/v4"+url, after, before, page, perPage)

	resp, err := c.httpClient.R().
		SetQueryParams(map[string]string{
			"action":   "pushed",
			"after":    after,
			"before":   before,
			"page":     fmt.Sprintf("%d", page),
			"per_page": fmt.Sprintf("%d", perPage),
		}).
		SetResult(&events).
		Get(url)

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API返回错误状态: %d, body: %s", resp.StatusCode(), resp.String())
	}

	log.Printf("Events原始响应(%d条): %s", len(events), resp.String())

	return events, nil
}
