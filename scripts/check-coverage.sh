#!/usr/bin/env bash
# check-coverage.sh
# 校验每个 Go 包和总体覆盖率是否 ≥ 阈值。
#
# 用法：
#   go test -short -coverprofile=coverage.out -covermode=atomic ./...
#   bash scripts/check-coverage.sh
#
# 退出码：0 = 全部达标，1 = 存在未达标包

set -euo pipefail

COVER_FILE="${1:-coverage.out}"
THRESHOLD="${2:-90.0}"
FAIL=0
PASS=0

if [[ ! -f "$COVER_FILE" ]]; then
    echo "错误：覆盖率文件 $COVER_FILE 不存在"
    echo "请先运行： go test -short -coverprofile=$COVER_FILE -covermode=atomic ./..."
    exit 1
fi

echo "============================================"
echo "  覆盖率门禁检查（阈值：${THRESHOLD}%）"
echo "============================================"
echo ""

# 从 coverage.out 解析每个包的语句覆盖率
# 格式：module/file.go:line.col,line.col num_stmts exec_count
# 跳过 mode 行、mock 包
awk -v threshold="$THRESHOLD" '
/mode:/ { next }
/\/mock\// { next }

{
    # 提取包名：从文件路径中去掉最后的文件名
    file = $1
    sub(/:.*/, "", file)  # 去掉 :行号.列号,行号.列号
    sub(/\/[^\/]+$/, "", file)  # 去掉文件名，保留包目录

    stmts = $2
    count = $3

    pkg_stmts[file] += stmts
    if (count > 0) pkg_exec[file] += stmts  # count>=1 表示已覆盖
}

END {
    total_stmts = 0
    total_exec = 0

    for (pkg in pkg_stmts) {
        # 排除根模块目录（只包含 main.go）
        if (pkg == "github.com/agent-forge/cli") continue

        stmts = pkg_stmts[pkg]
        exec = pkg_exec[pkg]
        pct = (stmts > 0) ? (exec / stmts) * 100 : 0

        total_stmts += stmts
        total_exec += exec

        if (pct >= threshold) {
            status = "✓"
        } else {
            status = "✗ 未达标"
            fail++
        }
        pass++

        printf "  %-60s %6.1f%%  %s\n", pkg, pct, status
    }

    total_pct = (total_stmts > 0) ? (total_exec / total_stmts) * 100 : 0

    print ""
    print "============================================"
    printf "总体覆盖率（排除 mock）：%.1f%%", total_pct
    if (total_pct >= threshold) {
        print " ✓ 达标"
    } else {
        printf " ✗ 未达标（阈值 %.1f%%）\n", threshold
        fail++
    }

    print ""
    if (fail == 0) {
        printf "结果：全部通过 ✓ (%d 个包达标)\n", pass
        exit 0
    } else {
        printf "结果：%d 个检查未通过 ✗\n", fail
        print "请补充单元测试使覆盖率 ≥ " threshold "%"
        exit 1
    }
}
' "$COVER_FILE"
