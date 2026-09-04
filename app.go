package main

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	user32            = syscall.NewLazyDLL("user32.dll")
	procGetModuleHandle = kernel32.NewProc("GetModuleHandleW")
	procLoadImage     = user32.NewProc("LoadImageW")
	procSendMessage   = user32.NewProc("SendMessageW")
	procFindWindow    = user32.NewProc("FindWindowW")
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 设置窗口图标
	go func() {
		for i := 0; i < 20; i++ {
			time.Sleep(200 * time.Millisecond)
			hwnd := findWindow()
			if hwnd == 0 {
				continue
			}
			hInst, _, _ := procGetModuleHandle.Call(0)
			hIcon, _, _ := procLoadImage.Call(hInst, 1, 1, 0, 0, 0)
			if hIcon != 0 {
				procSendMessage.Call(hwnd, 0x0080, 1, hIcon)
				procSendMessage.Call(hwnd, 0x0080, 0, hIcon)
			}
			return
		}
	}()
}

func findWindow() uintptr {
	hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafePointer("DaoPowerplan")))
	return hwnd
}

func unsafePointer(s string) uintptr {
	ptr, _ := syscall.UTF16PtrFromString(s)
	return uintptr(unsafe.Pointer(ptr))
}

// 电源计划GUID
const (
	GUIDHighPerformance = "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"
	GUIDBalanced        = "381b4222-f694-41f0-9685-ff5bb260df2e"
	GUIDPowerSaver      = "a1841308-3541-4fab-bc81-f71556f20b4a"
)

type PowerPlan struct {
	GUID   string `json:"guid"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// GetCurrentPlan 获取当前电源计划
func (a *App) GetCurrentPlan() (string, error) {
	output, err := runPowercfg("/getactivescheme")
	if err != nil {
		return "", err
	}
	if idx := strings.Index(output, "("); idx > 0 {
		if end := strings.Index(output[idx:], ")"); end > 0 {
			return output[idx+1 : idx+end], nil
		}
	}
	return output, nil
}

// SetHighPerformance 切换高性能
func (a *App) SetHighPerformance() error {
	_, err := runPowercfg("/setactive", GUIDHighPerformance)
	return err
}

// SetBalanced 切换均衡
func (a *App) SetBalanced() error {
	_, err := runPowercfg("/setactive", GUIDBalanced)
	return err
}

// SetPowerSaver 切换节能
func (a *App) SetPowerSaver() error {
	_, err := runPowercfg("/setactive", GUIDPowerSaver)
	return err
}

// ListPlans 列出所有电源计划
func (a *App) ListPlans() ([]PowerPlan, error) {
	output, err := runPowercfg("/list")
	if err != nil {
		return nil, err
	}

	var plans []PowerPlan
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "GUID") {
			continue
		}
		plan := PowerPlan{}
		if idx := strings.Index(line, ":"); idx > 0 {
			rest := line[idx+1:]
			openIdx := strings.Index(rest, "(")
			closeIdx := strings.LastIndex(rest, ")")
			plan.GUID = strings.TrimSpace(rest[:openIdx])
			if openIdx >= 0 && closeIdx > openIdx {
				name := strings.TrimSpace(rest[openIdx+1 : closeIdx])
				plan.Name = name
				plan.Active = strings.Contains(rest[closeIdx:], "*")
			}
		}
		if plan.GUID != "" {
			plans = append(plans, plan)
		}
	}

	// 排序：系统默认计划（高性能、均衡、节能）排在最前面
	defaultOrder := map[string]int{
		GUIDHighPerformance: 0,
		GUIDBalanced:        1,
		GUIDPowerSaver:      2,
	}
	sort.SliceStable(plans, func(i, j int) bool {
		oi, ok1 := defaultOrder[plans[i].GUID]
		oj, ok2 := defaultOrder[plans[j].GUID]
		if ok1 && ok2 {
			return oi < oj
		}
		if ok1 {
			return true
		}
		if ok2 {
			return false
		}
		return plans[i].Name < plans[j].Name
	})

	return plans, nil
}

// SetPlanByGUID 按GUID切换电源计划
func (a *App) SetPlanByGUID(guid string) error {
	_, err := runPowercfg("/setactive", guid)
	return err
}

// CreateCustomPlan 创建定制电源计划
func (a *App) CreateCustomPlan(name string, baseGUID string, desc ...string) (string, error) {
	if baseGUID == "" {
		baseGUID = GUIDBalanced
	}
	output, err := runPowercfg("/duplicatescheme", baseGUID)
	if err != nil {
		return "", err
	}

	var newGUID string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "GUID") {
			if idx := strings.Index(line, ":"); idx > 0 {
				newGUID = strings.TrimSpace(line[idx+1:])
				if spaceIdx := strings.Index(newGUID, " "); spaceIdx > 0 {
					newGUID = newGUID[:spaceIdx]
				}
				break
			}
		}
	}

	if newGUID == "" {
		return "", nil
	}

	if len(desc) > 0 && desc[0] != "" {
		runPowercfg("/changename", newGUID, name, desc[0])
	} else {
		runPowercfg("/changename", newGUID, name)
	}
	return newGUID, nil
}

// DeletePlan 删除电源计划
func (a *App) DeletePlan(guid string) error {
	_, err := runPowercfg("/delete", guid)
	return err
}

// PowerSetting 电源设置项
type PossibleValue struct {
	Index string `json:"index"`
	Name  string `json:"name"`
}

type PowerSetting struct {
	GroupGUID     string          `json:"groupGuid"`
	GroupGUID2    string          `json:"groupGuid2"`
	SettingGUID   string          `json:"settingGuid"`
	SettingGUID2  string          `json:"settingGuid2"`
	Name          string          `json:"name"`
	CurrentAC     string          `json:"currentAC"`
	CurrentDC     string          `json:"currentDC"`
	PossibleValues []PossibleValue `json:"possibleValues"`
}

// GetPlanSettings 获取电源计划的所有设置
func (a *App) GetPlanSettings(planGUID string) ([]PowerSetting, error) {
	output, err := runPowercfg("/query", planGUID)
	if err != nil {
		return nil, err
	}

	var settings []PowerSetting
	var currentGroupGUID, currentGroupGUID2 string

	lines := strings.Split(output, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// 检测子组 GUID
		if strings.HasPrefix(line, "子组 GUID:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				rest := strings.TrimSpace(parts[1])
				guidParts := strings.SplitN(rest, "(", 2)
				currentGroupGUID = strings.TrimSpace(guidParts[0])
				// 尝试获取别名GUID
				if i+1 < len(lines) {
					next := strings.TrimSpace(lines[i+1])
					if strings.HasPrefix(next, "GUID 别名:") {
						currentGroupGUID2 = strings.TrimSpace(strings.SplitN(next, ":", 2)[1])
						i++
					} else {
						currentGroupGUID2 = ""
					}
				}
			}
			continue
		}

		// 检测电源设置 GUID
		if strings.HasPrefix(line, "电源设置 GUID:") {
			setting := PowerSetting{
				GroupGUID:  currentGroupGUID,
				GroupGUID2: currentGroupGUID2,
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				rest := strings.TrimSpace(parts[1])
				guidParts := strings.SplitN(rest, "(", 2)
				setting.SettingGUID = strings.TrimSpace(guidParts[0])
				if len(guidParts) > 1 {
					setting.Name = strings.TrimSuffix(guidParts[1], ")")
				}
			}

			// 查找别名、当前值和预设选项
			var pendingIndex string
			for j := i + 1; j < len(lines) && j < i+15; j++ {
				next := strings.TrimSpace(lines[j])
				if strings.HasPrefix(next, "GUID 别名:") {
					setting.SettingGUID2 = strings.TrimSpace(strings.SplitN(next, ":", 2)[1])
				}
				if strings.HasPrefix(next, "当前交流电源设置索引:") {
					setting.CurrentAC = strings.TrimSpace(strings.SplitN(next, ":", 2)[1])
				}
				if strings.HasPrefix(next, "当前直流电源设置索引:") {
					setting.CurrentDC = strings.TrimSpace(strings.SplitN(next, ":", 2)[1])
				}
				if strings.HasPrefix(next, "可能的设置索引:") {
					pendingIndex = strings.TrimSpace(strings.SplitN(next, ":", 2)[1])
				}
				if strings.HasPrefix(next, "可能的设置友好名称:") {
					name := strings.TrimSpace(strings.SplitN(next, ":", 2)[1])
					if pendingIndex != "" {
						setting.PossibleValues = append(setting.PossibleValues, PossibleValue{
							Index: pendingIndex,
							Name:  name,
						})
						pendingIndex = ""
					}
				}
				if strings.HasPrefix(next, "电源设置 GUID:") || strings.HasPrefix(next, "子组 GUID:") {
					break
				}
			}

			if setting.Name != "" {
				settings = append(settings, setting)
			}
		}
	}

	return settings, nil
}

// SetSettingValue 修改电源设置值
func (a *App) SetSettingValue(planGUID string, settingGUID string, acValue string, dcValue string) error {
	if acValue != "" {
		acInt, err := strconv.Atoi(acValue)
		if err == nil {
			_, _ = runPowercfg("/setacvalueindex", planGUID, settingGUID, strconv.Itoa(acInt))
		}
	}
	if dcValue != "" {
		dcInt, err := strconv.Atoi(dcValue)
		if err == nil {
			_, _ = runPowercfg("/setdcvalueindex", planGUID, settingGUID, strconv.Itoa(dcInt))
		}
	}
	// 应用设置
	_, err := runPowercfg("/setactive", planGUID)
	return err
}

// RenamePlan 重命名电源计划
func (a *App) RenamePlan(guid string, name string) error {
	_, err := runPowercfg("/changename", guid, name)
	return err
}

// 电源设置常量
const (
	// 子组
	SUB_PROCESSOR = "54533251-82be-4824-96c1-47b60b740d00"
	SUB_DISK      = "0012ee47-9041-4b5d-9b77-535fba8b1442"
	SUB_VIDEO     = "7516b95f-f776-4464-8c53-06167f40cc99"
	SUB_SLEEP     = "238c9fa8-0aad-41ed-83f4-97be242c8f20"
	SUB_PCIEXPRESS = "501a4d13-42af-4429-9fd1-a8218c268e20"
	SUB_USB       = "2a737441-1930-4402-8d77-b2bebba308a3"
	// 卓越性能计划模板
	GUIDUltimatePerformance = "e9a42b02-d5df-448d-aa00-03f14749eb61"
	// 处理器设置项
	PROCTHROTTLEMAX      = "bc5038f7-23e0-4960-96da-33abaf5935ec"
	PROCTHROTTLEMIN      = "893dee8e-2bef-41e0-89c6-b55d0929964c"
	PERFBOOSTMODE        = "be337238-0d82-4146-a960-4f3749d470c7"
	PERFIDLEDISABLE      = "5d76a2ca-e8c0-402f-a133-2158492d58ad"
	COREPARKMINCORES     = "0cc5b647-c1df-4637-891a-dec35c318583"
	COREPARKCONCURRENCY  = "3b04d4fd-1cc7-4f23-ab1c-d1337819c4bb"
	COREPARKMAXCORES     = "ea062031-0e34-4ff1-9b6d-eb1059334028"
	COREPARKMINCORESPERF = "36687f9e-e3a5-4dbf-b1dc-15eb381c6863"
	PERFINCREASETHRESH   = "45bcc044-d885-43e2-8605-ee0ec6e96b59"
	PERFDECREASETHRESH   = "06cadf0e-64ed-448a-8927-ce7bf90eb35d"
	PERFINCREASEPOLICY   = "984cf492-3bed-4488-a8f9-4286c97bf5aa"
	PERFDECREASEPOLICY   = "40fbefc7-2e9d-4d25-a185-0cfd8574bac6"
	LATENCYSENSITIVITY   = "4b92d758-5a24-4851-a470-815d78aee119"
	PERFTIMECHECK        = "7b224883-b3cc-4d79-819f-8374152cbe7c"
	PERFAUTONOMOUS       = "943c8cb6-6f93-4227-ad87-e9a3feec08d1"
	// 异类线程调度（大小核）
	HETEROPOLICY         = "7f2f5cfa-f10c-4823-b5e1-e93ae85f46b5"
	SCHEDPOLICY          = "93b8b6dc-0698-4d1c-9ee4-0644e900c85d"
	SHORTSCHEDPOLICY     = "bae08b81-2d5e-4688-ad6a-13243356654b"
	// 其他处理器设置
	SYSCOOLPOL           = "94d3a615-a899-4ac5-ae2b-e4d8f634367f"
	PERFHISTORY          = "7d24baa7-0b84-480f-840c-1b0743c00f5f"
	COREPARKMINCORESPERF2 = "36687f9e-e3a5-4dbf-b1dc-15eb381c6863"
	COREPARKMAXCORESPERF = "ea062031-0e34-4ff1-9b6d-eb1059334028"
	// 其他设置项
	DISKIDLE      = "6738e2c4-e8a5-4a42-b16a-e040e769756e"
	VIDEOIDLE     = "3c0bc021-c8a8-4e07-a973-6b14cbcb2b7e"
	STANDBYIDLE   = "29f6c1db-86da-48c5-9fdb-f2b67b1f44da"
	PCIELINKSTATE = "ee12f906-d277-404b-b6da-e5fa1a576df5"
	USBSELECTIVE  = "48e6b7a6-50f5-4782-a5d4-53bb8f07e226"
)

func setPlanValue(planGUID, subGUID, settingGUID, acVal, dcVal string) {
	if acVal != "" {
		runPowercfg("/setacvalueindex", planGUID, subGUID, settingGUID, acVal)
	}
	if dcVal != "" {
		runPowercfg("/setdcvalueindex", planGUID, subGUID, settingGUID, dcVal)
	}
}

// CreateGameMode 创建游戏模式电源计划（极致性能）
func (a *App) CreateGameMode(cpuType string) (string, error) {
	name := "游戏模式"
	if cpuType == "intel" {
		name = "游戏模式 (Intel)"
	} else if cpuType == "amd" {
		name = "游戏模式 (AMD)"
	}

	// 基于卓越性能计划创建
	gameDesc := "性能拉满，专为游戏优化"
	if cpuType == "intel" {
		gameDesc = "性能拉满，大核优先，专为游戏优化"
	} else if cpuType == "amd" {
		gameDesc = "全核满载，极致性能，专为游戏优化"
	}
	newGUID, err := a.CreateCustomPlan(name, GUIDUltimatePerformance, gameDesc)
	if err != nil || newGUID == "" {
		// 如果卓越性能不可用，回退到高性能
		newGUID, err = a.CreateCustomPlan(name, GUIDHighPerformance, gameDesc)
		if err != nil || newGUID == "" {
			return newGUID, err
		}
	}

	// === 处理器电源管理 - 极致性能 ===
	// 最小/最大处理器状态 100%
	setPlanValue(newGUID, SUB_PROCESSOR, PROCTHROTTLEMIN, "100", "100")
	setPlanValue(newGUID, SUB_PROCESSOR, PROCTHROTTLEMAX, "100", "100")
	// 处理器性能提升模式：激进 (2 = Aggressive)
	setPlanValue(newGUID, SUB_PROCESSOR, PERFBOOSTMODE, "2", "2")
	// 禁用处理器空闲
	setPlanValue(newGUID, SUB_PROCESSOR, PERFIDLEDISABLE, "1", "1")
	// 核心停车：最小核心 100%（禁用核心停车）
	setPlanValue(newGUID, SUB_PROCESSOR, COREPARKMINCORES, "100", "100")
	setPlanValue(newGUID, SUB_PROCESSOR, COREPARKMAXCORES, "100", "100")
	setPlanValue(newGUID, SUB_PROCESSOR, COREPARKMINCORESPERF, "100", "100")
	// 核心停车并发阈值 0
	setPlanValue(newGUID, SUB_PROCESSOR, COREPARKCONCURRENCY, "0", "0")
	// 性能提升阈值 100（立即提升）
	setPlanValue(newGUID, SUB_PROCESSOR, PERFINCREASETHRESH, "100", "100")
	// 性能降低阈值 0（不降低）
	setPlanValue(newGUID, SUB_PROCESSOR, PERFDECREASETHRESH, "0", "0")
	// 性能提升/降低策略：Rocket (2 = 火箭，直接跳到最高/最低状态)
	setPlanValue(newGUID, SUB_PROCESSOR, PERFINCREASEPOLICY, "2", "2")
	setPlanValue(newGUID, SUB_PROCESSOR, PERFDECREASEPOLICY, "2", "2")
	// 延迟敏感度提示：覆盖空闲 (0)
	setPlanValue(newGUID, SUB_PROCESSOR, LATENCYSENSITIVITY, "0", "0")
	// 性能时间检查间隔 0
	setPlanValue(newGUID, SUB_PROCESSOR, PERFTIMECHECK, "0", "0")
	// 禁用性能自主模式
	setPlanValue(newGUID, SUB_PROCESSOR, PERFAUTONOMOUS, "0", "0")
	// 系统散热方式：主动（1=Active，优先散热不降频）
	setPlanValue(newGUID, SUB_PROCESSOR, SYSCOOLPOL, "1", "1")
	// 处理器性能历史记录：1（最小历史采样，更快响应负载变化）
	setPlanValue(newGUID, SUB_PROCESSOR, PERFHISTORY, "1", "1")
	// 核心停车最小核心（性能）：100%
	setPlanValue(newGUID, SUB_PROCESSOR, COREPARKMINCORESPERF2, "100", "100")
	// 核心停车最大核心（性能）：100%
	setPlanValue(newGUID, SUB_PROCESSOR, COREPARKMAXCORESPERF, "100", "100")

	// === 异类线程调度策略（区分Intel大小核 / AMD全大核）===
	if cpuType == "intel" {
		// Intel 12代+ 大小核架构：大核优先
		// 生效的异类策略：1（严格执行手动策略）
		setPlanValue(newGUID, SUB_PROCESSOR, HETEROPOLICY, "1", "1")
		// 异类线程调度策略：2（首选高性能处理器，大核忙时用小核）
		setPlanValue(newGUID, SUB_PROCESSOR, SCHEDPOLICY, "2", "2")
		// 异类短运行线程调度策略：1（高性能处理器，短线程直接上大核）
		setPlanValue(newGUID, SUB_PROCESSOR, SHORTSCHEDPOLICY, "1", "1")
	} else {
		// AMD 桌面CPU多为全大核，无大小核区别
		// 异类线程调度策略：0（所有处理器平均分配）
		setPlanValue(newGUID, SUB_PROCESSOR, SCHEDPOLICY, "0", "0")
		// 异类短运行线程调度策略：0（所有处理器）
		setPlanValue(newGUID, SUB_PROCESSOR, SHORTSCHEDPOLICY, "0", "0")
	}

	// === PCI Express - 关闭链接状态电源管理 ===
	setPlanValue(newGUID, SUB_PCIEXPRESS, PCIELINKSTATE, "0", "0")

	// === USB - 禁用选择性暂停 ===
	setPlanValue(newGUID, SUB_USB, USBSELECTIVE, "0", "0")

	// === 硬盘 - 从不关闭 ===
	setPlanValue(newGUID, SUB_DISK, DISKIDLE, "0", "0")

	// === 显示器 - 从不关闭 ===
	setPlanValue(newGUID, SUB_VIDEO, VIDEOIDLE, "0", "0")

	// === 睡眠 - 从不睡眠 ===
	setPlanValue(newGUID, SUB_SLEEP, STANDBYIDLE, "0", "0")

	runPowercfg("/setactive", newGUID)
	return newGUID, nil
}

// CreateOfficeMode 创建办公模式电源计划
func (a *App) CreateOfficeMode(cpuType string) (string, error) {
	name := "办公模式"
	if cpuType == "intel" {
		name = "办公模式 (Intel)"
	} else if cpuType == "amd" {
		name = "办公模式 (AMD)"
	}

	officeDesc := "平衡省电，适合日常办公"
	if cpuType == "intel" {
		officeDesc = "平衡省电，智能调度，适合日常办公"
	} else if cpuType == "amd" {
		officeDesc = "均衡能效，安静办公，适合日常办公"
	}
	newGUID, err := a.CreateCustomPlan(name, GUIDBalanced, officeDesc)
	if err != nil || newGUID == "" {
		return newGUID, err
	}

	// 办公模式：平衡并节省电量
	// 处理器最大状态 70%（限制性能省电）
	setPlanValue(newGUID, SUB_PROCESSOR, PROCTHROTTLEMAX, "70", "50")
	// 处理器最小状态 5%
	setPlanValue(newGUID, SUB_PROCESSOR, PROCTHROTTLEMIN, "5", "5")
	// 硬盘10分钟后关闭（600秒）
	setPlanValue(newGUID, SUB_DISK, DISKIDLE, "600", "300")
	// 显示器5分钟后关闭（300秒）
	setPlanValue(newGUID, SUB_VIDEO, VIDEOIDLE, "300", "120")
	// 15分钟后睡眠（900秒）
	setPlanValue(newGUID, SUB_SLEEP, STANDBYIDLE, "900", "600")

	runPowercfg("/setactive", newGUID)
	return newGUID, nil
}

// IsSystemPlan 判断是否为系统默认计划（不可删除）
func (a *App) IsSystemPlan(guid string) bool {
	return guid == GUIDHighPerformance || guid == GUIDBalanced || guid == GUIDPowerSaver
}
