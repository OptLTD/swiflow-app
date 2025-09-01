package ability

import (
	"strings"
	"testing"
	"time"
)

func TestDevAsyncCmdAbility_Basic(t *testing.T) {
	tempDir := t.TempDir()
	session := "testsession"
	ability := &DevAsyncCmdAbility{
		Home: tempDir, Name: session,
	}

	// 启动异步命令
	err := ability.Start("echo hello-async && sleep 1 && echo done")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 等待命令执行
	time.Sleep(2 * time.Second)

	// 查询日志
	log, err := ability.Query()
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if !strings.Contains(log, "hello-async") || !strings.Contains(log, "done") {
		t.Errorf("Log content incorrect: %s", log)
	}

	// 中止 session（即使已结束也应无异常）
	if err = ability.Abort(); err != nil {
		t.Errorf("Abort failed: %v", err)
	}
}
