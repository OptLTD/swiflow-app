package agent

import (
	"fmt"
	"strings"
)

// CommitPoint commit point上下文管理器
type CommitPoint struct {
	points  map[string]string // commit point -> task title
	current string

	context string
}

// NewCommitPoint 创建新的commit point管理器
func NewCommitPoint(context string) *CommitPoint {
	return &CommitPoint{
		points:  make(map[string]string),
		context: context,
	}
}

// ParsePoints 解析任务上下文中的commit point结构
func (cm *CommitPoint) ParsePoints() {
	if cm.context == "" {
		return
	}

	cm.points = make(map[string]string)
	lines := strings.Split(cm.context, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是顶级任务（以"- [ ]"开头，没有额外缩进）
		if strings.HasPrefix(line, "- [ ]") {
			// 提取commit point ID
			if strings.Contains(line, "(commit point:") {
				parts := strings.Split(line, "(commit point:")
				if len(parts) >= 2 {
					commitPointPart := strings.TrimSpace(parts[1])
					commitPoint := strings.TrimSuffix(commitPointPart, ")")

					// 提取任务标题
					taskPart := strings.TrimSpace(parts[0])
					taskTitle := strings.TrimPrefix(taskPart, "- [ ]")
					taskTitle = strings.TrimSpace(taskTitle)

					cm.points[commitPoint] = taskTitle
				}
			}
		}
	}
}

// GetCommitPointContext 获取指定commit point的上下文
func (cm *CommitPoint) GetCommitPointContext(point string) string {
	if cm.points == nil {
		cm.ParsePoints()
	}

	var contextLines []string
	lines := strings.Split(cm.context, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 如果是顶级任务
		if strings.HasPrefix(line, "- [ ]") {
			// 检查是否包含commit point
			if strings.Contains(line, "(commit point:") {
				parts := strings.Split(line, "(commit point:")
				if len(parts) >= 2 {
					commitPointPart := strings.TrimSpace(parts[1])
					currentCommitPoint := strings.TrimSuffix(commitPointPart, ")")

					// 只包含当前commit point及之前的已完成任务
					if currentCommitPoint == point || cm.isCompleted(currentCommitPoint) {
						contextLines = append(contextLines, line)
					}
				}
			} else {
				// 没有commit point的任务，根据完成状态决定是否包含
				if cm.isTaskCompleted(line) {
					contextLines = append(contextLines, line)
				}
			}
		} else if strings.HasPrefix(line, "    - [ ]") {
			// 子任务，只有在当前commit point下才包含
			// 这里可以根据需要调整逻辑
			continue
		}
	}

	return strings.Join(contextLines, "\n")
}

// isCompleted 检查commit point是否已完成
func (cm *CommitPoint) isCompleted(point string) bool {
	lines := strings.Split(cm.context, "\n")

	for _, line := range lines {
		if strings.Contains(line, "(commit point:"+point+")") {
			// 检查是否标记为完成 [x]
			return strings.Contains(line, "- [x]")
		}
	}
	return false
}

// isTaskCompleted 检查任务是否已完成
func (cm *CommitPoint) isTaskCompleted(taskLine string) bool {
	return strings.Contains(taskLine, "- [x]")
}

// SetCurrent 设置当前commit point
func (cm *CommitPoint) SetCurrent(point string) {
	cm.current = point
}

// UpdateStatus 更新commit point状态
func (cm *CommitPoint) UpdateStatus(commitPoint string, completed bool) error {
	if cm.context == "" {
		return fmt.Errorf("task context is empty")
	}

	lines := strings.Split(cm.context, "\n")
	var newLines []string

	for _, line := range lines {
		if strings.Contains(line, "(commit point:"+commitPoint+")") {
			// 更新任务状态
			if completed {
				// 标记为完成
				line = strings.Replace(line, "- [ ]", "- [x]", 1)
			} else {
				// 标记为未完成
				line = strings.Replace(line, "- [x]", "- [ ]", 1)
			}
		}
		newLines = append(newLines, line)
	}

	cm.context = strings.Join(newLines, "\n")

	// 重新解析commit point结构
	cm.ParsePoints()

	return nil
}

// GetNextPoint 获取下一个commit point
func (cm *CommitPoint) GetNextPoint() string {
	if cm.points == nil {
		cm.ParsePoints()
	}

	commitPoints := make([]string, 0)
	for commitPoint := range cm.points {
		commitPoints = append(commitPoints, commitPoint)
	}

	// 简单的排序逻辑，可以根据需要调整
	// 这里假设commit point按字母顺序排列
	for _, commitPoint := range commitPoints {
		if !cm.isCompleted(commitPoint) {
			return commitPoint
		}
	}

	return ""
}

// GetProgress 获取commit point进度
func (cm *CommitPoint) GetProgress() (completed, total int) {
	if cm.points == nil {
		cm.ParsePoints()
	}

	total = len(cm.points)
	completed = 0

	for commitPoint := range cm.points {
		if cm.isCompleted(commitPoint) {
			completed++
		}
	}

	return completed, total
}

// GetCommitPoints 获取所有commit point
func (cm *CommitPoint) GetCommitPoints() map[string]string {
	if cm.points == nil {
		cm.ParsePoints()
	}
	return cm.points
}

// GetCurrent 获取当前commit point
func (cm *CommitPoint) GetCurrent() string {
	return cm.current
}

// Validate 验证commit point是否存在
func (cm *CommitPoint) Validate(point string) bool {
	if cm.points == nil {
		cm.ParsePoints()
	}
	_, exists := cm.points[point]
	return exists
}
