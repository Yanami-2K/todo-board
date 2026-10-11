package seed

import (
	"fmt"

	"github.com/Yanami-2K/todo-board/internal/store"
)

// Run loads fictional example data. It is never called on a normal start.
func Run(st *store.Store, force bool) error {
	empty, err := st.DomainEmpty()
	if err != nil {
		return err
	}
	if !empty && !force {
		return fmt.Errorf("数据库里已经有数据。确认要换成示例数据再执行 seed --force")
	}
	if !empty {
		if err := st.ResetDomain(); err != nil {
			return err
		}
	}
	name, _ := st.GetSetting("owner_name")
	if name == "" {
		name = "我"
	}
	owner := store.Actor{Kind: "owner", ID: "owner", Name: name}

	type proj struct{ name, color string }
	projects := []proj{
		{"remote-dev", "mint"},
		{"log-center", "blue"},
		{"mini-infra", "yellow"},
	}
	pids := map[string]string{}
	for _, p := range projects {
		row, err := st.CreateProject(p.name, p.color)
		if err != nil {
			return err
		}
		pids[p.name] = row.ID
	}

	type ag struct{ name, role string }
	agents := []ag{
		{"remote-dev", "沙箱镜像"},
		{"miniinfra-ops", "运维"},
		{"harness-runner", "评测跑批"},
	}
	ids := map[string]store.Agent{}
	fmt.Println("示例 Agent token（只这一次）：")
	for _, a := range agents {
		row, token, err := st.CreateAgent(a.name, a.role, store.GenLook(a.name, 0))
		if err != nil {
			return err
		}
		ids[a.name] = row
		fmt.Printf("  %s  %s\n", a.name, token)
	}

	if _, err := st.CreateTodo(store.CreateTodo{
		Title: "整理这周要自己看的事", Note: "不用很长", Source: "self", Creator: owner, Lane: "self", ProjectID: pids["mini-infra"],
	}); err != nil {
		return err
	}

	rd := ids["remote-dev"]
	actor := store.Actor{Kind: "agent", ID: rd.ID, Name: rd.Name}
	t, err := st.CreateTodo(store.CreateTodo{
		Title: "构建沙箱基础镜像", Description: "把基础镜像推到自己的 registry。",
		Status: "in_progress", Steps: []string{"拉取基础层", "安装运行时", "跑冒烟测试"}, Step: 2,
		Source: "agent", Creator: actor, DelegateID: rd.ID, Lane: rd.ID, ProjectID: pids["remote-dev"],
	})
	if err != nil {
		return err
	}
	if _, _, err := st.AskOwner(store.Ask{
		TodoID: t.ID, Question: "扫出 2 个高危漏洞，先忽略还是先修？",
		Context: "上游已经有补丁。", Options: []string{"先忽略，下版修", "现在就修"}, Agent: actor,
	}); err != nil {
		return err
	}

	ops := ids["miniinfra-ops"]
	if _, err := st.CreateTodo(store.CreateTodo{
		Title: "节点磁盘快满了，确认能否清理旧镜像", Status: "todo",
		Source: "agent", Creator: store.Actor{Kind: "agent", ID: ops.ID, Name: ops.Name},
		DelegateID: ops.ID, Lane: ops.ID, ProjectID: pids["mini-infra"],
	}); err != nil {
		return err
	}

	if _, err := st.SubmitIntake(store.IntakeIn{
		Title: "申请两个沙箱测试账号", Body: "下周要压测，两个独立账号就够。",
		FromName: "陈默", FromKind: "human", Contact: "chenmo@example.com", Want: "周五前",
		Via: "公开登记页", SuggestID: rd.ID,
	}); err != nil {
		return err
	}
	if _, err := st.SubmitIntake(store.IntakeIn{
		Title: "帮忙看一下模块拆分的说明", Body: "组会前想听你的意见。",
		FromName: "张蔚", FromKind: "human", Contact: "zhangwei@example.com", Want: "周三前", Via: "公开登记页",
	}); err != nil {
		return err
	}
	fmt.Println("示例数据已写入。这些名字都是虚构的，只存在于这次 seed。")
	return nil
}
