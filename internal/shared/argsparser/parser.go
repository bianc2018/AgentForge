package argsparser

import (
	"fmt"
	"strconv"
	"strings"
)

// ErrUnknownFlag 表示遇到了未知的命令行参数。
type ErrUnknownFlag struct {
	Flag string
}

func (e *ErrUnknownFlag) Error() string {
	return fmt.Sprintf("未知参数: %s", e.Flag)
}

// ErrMissingValue 表示参数缺少必需的值。
type ErrMissingValue struct {
	Flag string
}

func (e *ErrMissingValue) Error() string {
	return fmt.Sprintf("参数 %s 缺少值", e.Flag)
}

// ErrInvalidValue 表示参数值格式错误。
type ErrInvalidValue struct {
	Flag   string
	Value  string
	Reason string
}

func (e *ErrInvalidValue) Error() string {
	return fmt.Sprintf("参数 %s 的值 %q 无效: %s", e.Flag, e.Value, e.Reason)
}

// isFlag 检查 token 是否以 - 或 -- 开头。
func isFlag(token string) bool {
	return strings.HasPrefix(token, "-") && len(token) > 1
}

// stripFlagPrefix 移除 - 或 -- 前缀，返回标志名称。
func stripFlagPrefix(token string) string {
	if strings.HasPrefix(token, "--") {
		return token[2:]
	}
	return token[1:]
}

// flagSet 表示一个标志是否已被设置（避免重复设置冲突检测）。
type flagSet map[string]bool

// ParseBuild 将原始命令行参数解析为 BuildParams。
//
// args 是原始参数切片，通常来自 os.Args[1:]。
// 函数只提取与 build 命令相关的已知参数，忽略不属于 build 命令的
// 子命令名称（如 "build" 本身）。
//
// 示例：
//
//	ParseBuild([]string{"-d", "all", "--max-retry", "3"})
//	ParseBuild([]string{"--no-cache", "-b", "centos:8", "--gh-proxy", "https://proxy.example.com"})
func ParseBuild(args []string) (*BuildParams, error) {
	params := DefaultBuildParams()
	seen := make(flagSet)
	i := 0

	for i < len(args) {
		token := args[i]

		if !isFlag(token) {
			// 跳过非 flag token（如子命令名称 "build"）
			i++
			continue
		}

		flagName := stripFlagPrefix(token)

		switch flagName {
		case "d", "deps":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Deps = args[i]
			seen["deps"] = true

		case "b", "base-image":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.BaseImage = args[i]
			seen["base-image"] = true

		case "c", "config":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Config = args[i]
			seen["config"] = true

		case "no-cache":
			params.NoCache = true
			seen["no-cache"] = true

		case "R", "rebuild":
			params.Rebuild = true
			seen["rebuild"] = true

		case "max-retry":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			val, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, &ErrInvalidValue{
					Flag:   token,
					Value:  args[i],
					Reason: "必须为整数",
				}
			}
			if val < 0 {
				return nil, &ErrInvalidValue{
					Flag:   token,
					Value:  args[i],
					Reason: "必须为非负整数",
				}
			}
			params.MaxRetry = val
			seen["max-retry"] = true

		case "gh-proxy":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.GHProxy = args[i]
			seen["gh-proxy"] = true

		default:
			// 检查是否是已知的 run 参数（跨命令参数给出友好提示）
			if isRunFlag(flagName) {
				return nil, &ErrUnknownFlag{
					Flag: fmt.Sprintf("%s（此参数属于 run 命令，不适用于 build）", token),
				}
			}
			return nil, &ErrUnknownFlag{Flag: token}
		}

		i++
	}

	return &params, nil
}

// ParseRun 将原始命令行参数解析为 RunParams。
//
// args 是原始参数切片，通常来自 os.Args[1:]。
//
// 示例：
//
//	ParseRun([]string{"-a", "claude", "-p", "3000:3000", "-p", "8080:8080"})
//	ParseRun([]string{"--docker", "--run", "npm test"})
func ParseRun(args []string) (*RunParams, error) {
	params := DefaultRunParams()
	seen := make(flagSet)
	i := 0

	for i < len(args) {
		token := args[i]

		if !isFlag(token) {
			// 跳过非 flag token（如子命令名称 "run"）
			i++
			continue
		}

		flagName := stripFlagPrefix(token)

		switch flagName {
		case "a", "agent":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Agent = args[i]
			seen["agent"] = true

		case "p", "port":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Ports = append(params.Ports, args[i])
			seen["port"] = true

		case "m", "mount":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Mounts = append(params.Mounts, args[i])
			seen["mount"] = true

		case "e", "env":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Envs = append(params.Envs, args[i])
			seen["env"] = true

		case "w", "workdir":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Workdir = args[i]
			seen["workdir"] = true

		case "r", "recall":
			params.Recall = true
			seen["recall"] = true

		case "docker", "dind":
			params.Docker = true
			seen["docker"] = true

		case "run":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.RunCmd = args[i]
			seen["run"] = true

		case "c", "config":
			if i+1 >= len(args) {
				return nil, &ErrMissingValue{Flag: token}
			}
			i++
			params.Config = args[i]
			seen["config"] = true

		default:
			// 检查是否是已知的 build 参数（跨命令参数给出友好提示）
			if isBuildFlag(flagName) {
				return nil, &ErrUnknownFlag{
					Flag: fmt.Sprintf("%s（此参数属于 build 命令，不适用于 run）", token),
				}
			}
			return nil, &ErrUnknownFlag{Flag: token}
		}

		i++
	}

	return &params, nil
}

// isBuildFlag 检查 flag 名是否是 build 命令的已知参数。
func isBuildFlag(name string) bool {
	switch name {
	case "d", "deps", "b", "base-image", "no-cache", "R", "rebuild", "max-retry", "gh-proxy":
		return true
	}
	return false
}

// isRunFlag 检查 flag 名是否是 run 命令的已知参数。
func isRunFlag(name string) bool {
	switch name {
	case "a", "agent", "p", "port", "m", "mount", "e", "env", "w", "workdir",
		"r", "recall", "docker", "dind", "run":
		return true
	}
	return false
}
