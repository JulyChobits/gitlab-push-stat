package ai

import (
	"fmt"
	"strings"
)

// CommitInfo 提交信息（用于构建Prompt）
type CommitInfo struct {
	SHA         string
	Message     string
	CommittedAt string
	Additions   int
	Deletions   int
	Diffs       []DiffInfo
}

// DiffInfo 文件差异信息
type DiffInfo struct {
	OldPath     string
	NewPath     string
	NewFile     bool
	DeletedFile bool
	Diff        string
}

// ReviewContext 审核上下文
type ReviewContext struct {
	ReportDate  string
	ProjectName string
	AuthorName  string
	CommitCount int
	Commits     []CommitInfo
	ReviewType  string // project/author
}

// BuildProjectReviewPrompt 构建项目维度审核Prompt
func BuildProjectReviewPrompt(ctx *ReviewContext) []Message {
	systemMsg := Message{
		Role: "system",
		Content: `你是一位资深的代码审查专家和技术主管。你的任务是审查开发人员提交的代码变更，评估代码难度、工时和代码质量。

请从以下维度进行评估：

1. **代码难度**（简单/中等/复杂/高复杂）：
   - 分析代码逻辑复杂度、算法难度、架构层面影响
   - 是否涉及并发处理、性能优化、复杂业务逻辑
   - 考虑修改范围、影响面、风险程度

2. **预估工时**（小时）：
   - 编码时间：实际编写代码的时间
   - 调试时间：排查问题、调试代码的时间
   - 测试时间：自测和修复bug的时间
   - 参考标准：简单修改0.5-1h，普通功能2-4h，复杂功能4-8h，高复杂度8h+

3. **代码质量**（优秀/良好/一般/待改进）：
   - 代码规范性和命名规范
   - 逻辑清晰度和可读性
   - 是否遵循最佳实践
   - 潜在bug风险和边界条件处理
   - 代码可维护性

4. **代码亮点**：列出代码中的优秀实践

5. **改进建议**：指出潜在问题或可改进的地方

6. **工作总结**：用200字以内总结开发人员的工作内容和贡献

请严格以JSON格式返回结果，不要包含其他文字说明。`,
	}

	userContent := buildUserContent(ctx)
	userMsg := Message{
		Role:    "user",
		Content: userContent,
	}

	return []Message{systemMsg, userMsg}
}

// BuildAuthorReviewPrompt 构建人员汇总审核Prompt
func BuildAuthorReviewPrompt(ctx *ReviewContext, projectReviews []string) []Message {
	systemMsg := Message{
		Role: "system",
		Content: `你是一位资深的技术主管。你的任务是根据开发人员当天在各个项目的工作表现，给出综合评估。

请基于各项目的项目审核结果，给出该人员的综合评估：

1. **综合难度**（简单/中等/复杂/高复杂）：综合考虑所有项目的难度
2. **总工时**（小时）：所有项目工时的合理总和（考虑上下文切换成本）
3. **综合代码质量**（优秀/良好/一般/待改进）：综合所有项目的代码质量
4. **工作总结**：用300字以内总结该开发人员今天的整体工作表现和贡献
5. **改进建议**：针对整体工作的改进建议

请严格以JSON格式返回结果，不要包含其他文字说明。`,
	}

	userContent := fmt.Sprintf(`## 开发人员综合评估

**开发人员**：%s
**评估日期**：%s
**涉及项目数**：%d

### 各项目审核结果：

%s

请给出该人员的综合评估结果，JSON格式如下：
{
  "difficulty": "中等",
  "difficulty_reason": "涉及多个项目，整体难度适中",
  "work_hours": 6.5,
  "work_hours_detail": {"coding": 4.5, "debugging": 1.5, "testing": 0.5, "context_switch": 0.5},
  "code_quality": "良好",
  "quality_details": {"strengths": ["..."], "issues": ["..."]},
  "highlights": "今天在多个项目中都有贡献...",
  "suggestions": "建议减少多项目并行，提高专注度...",
  "summary": "该开发人员今天..."
}`, ctx.AuthorName, ctx.ReportDate, ctx.CommitCount, strings.Join(projectReviews, "\n\n"))

	userMsg := Message{
		Role:    "user",
		Content: userContent,
	}

	return []Message{systemMsg, userMsg}
}

// buildUserContent 构建用户消息内容
func buildUserContent(ctx *ReviewContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`## 代码审查任务

**开发人员**：%s
**项目**：%s
**审查日期**：%s
**提交数量**：%d个

### 代码提交详情

`, ctx.AuthorName, ctx.ProjectName, ctx.ReportDate, ctx.CommitCount))

	// 限制总diff长度，避免超过token限制（提高限制以容纳更多有效代码）
	maxTotalChars := 30000 // 约10000 tokens
	totalChars := 0

	for i, commit := range ctx.Commits {
		sb.WriteString(fmt.Sprintf("### Commit %d: %s\n", i+1, truncate(commit.Message, 100)))
		sb.WriteString(fmt.Sprintf("- 提交SHA: %s\n", commit.SHA[:8]))
		sb.WriteString(fmt.Sprintf("- 提交时间: %s\n", commit.CommittedAt))
		sb.WriteString(fmt.Sprintf("- 有效代码变更: +%d -%d\n", commit.Additions, commit.Deletions))
		sb.WriteString(fmt.Sprintf("- 变更文件: %d个\n\n", len(commit.Diffs)))

		if len(commit.Diffs) > 0 {
			sb.WriteString("**代码变更详情（已过滤自动生成文件）：**\n")
			for _, diff := range commit.Diffs {
				if totalChars > maxTotalChars {
					sb.WriteString("\n...(后续diff已省略，代码过长)\n")
					break
				}

				filePath := diff.NewPath
				if diff.NewFile {
					sb.WriteString(fmt.Sprintf("\n**[新增文件] %s**\n", filePath))
				} else if diff.DeletedFile {
					sb.WriteString(fmt.Sprintf("\n**[删除文件] %s**\n", filePath))
				} else {
					sb.WriteString(fmt.Sprintf("\n**[修改文件] %s**\n", filePath))
				}

				// 截断diff内容（提高单文件限制）
				diffContent := truncateDiff(diff.Diff, 2000)
				sb.WriteString("```diff\n")
				sb.WriteString(diffContent)
				sb.WriteString("\n```\n")

				totalChars += len(diffContent)
			}
		}
		sb.WriteString("\n---\n\n")

		if totalChars > maxTotalChars {
			break
		}
	}

	sb.WriteString(`请根据以上代码变更，返回JSON格式的审查结果：
{
  "difficulty": "中等",
  "difficulty_reason": "涉及业务逻辑调整，有一定复杂度",
  "work_hours": 4.5,
  "work_hours_detail": {"coding": 3.0, "debugging": 1.0, "testing": 0.5},
  "code_quality": "良好",
  "quality_details": {"strengths": ["命名规范", "逻辑清晰"], "issues": ["缺少部分边界检查"]},
  "highlights": "代码结构清晰，使用了合适的设计模式",
  "suggestions": "建议增加单元测试覆盖关键逻辑",
  "summary": "该开发人员今天主要完成了..."
}`)

	return sb.String()
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// truncateDiff 智能截断diff内容
func truncateDiff(diff string, maxLen int) string {
	if len(diff) <= maxLen {
		return diff
	}

	lines := strings.Split(diff, "\n")
	var result []string
	currentLen := 0

	for _, line := range lines {
		lineLen := len(line) + 1
		if currentLen+lineLen > maxLen-50 {
			result = append(result, "... (代码过长，已截断)")
			break
		}
		result = append(result, line)
		currentLen += lineLen
	}

	return strings.Join(result, "\n")
}
