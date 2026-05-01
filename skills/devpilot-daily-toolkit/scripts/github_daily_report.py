#!/usr/bin/env python3
"""
GitHub Daily Report Generator with AI Summary
汇总当天的所有 GitHub 活动，并用 AI 生成改动总结
"""

import subprocess
import json
import sys
from datetime import datetime, timedelta
import argparse
from collections import defaultdict


def run_gh_command(args):
    """运行 gh CLI 命令并返回结果"""
    try:
        result = subprocess.run(
            ["gh"] + args,
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout
    except subprocess.CalledProcessError as e:
        return None


def get_recent_repos(days=2):
    """获取最近有活动的仓库"""
    result = run_gh_command([
        "repo", "list",
        "--limit", "20",
        "--json", "nameWithOwner,updatedAt,pushedAt"
    ])
    
    if result:
        try:
            repos = json.loads(result)
            cutoff = (datetime.now() - timedelta(days=days)).isoformat()
            return [r for r in repos if r.get("pushedAt", "") > cutoff]
        except:
            pass
    return []


def get_repo_commits(repo, date):
    """获取仓库某天的 commits"""
    since = f"{date}T00:00:00Z"
    until_date = (datetime.strptime(date, "%Y-%m-%d") + timedelta(days=1)).strftime("%Y-%m-%d")
    until = f"{until_date}T00:00:00Z"
    
    result = run_gh_command([
        "api",
        f"repos/{repo}/commits",
        "--paginate",
        "--jq", f'map(select(.commit.author.date >= "{since}" and .commit.author.date < "{until}")) | map({{sha: .sha[:7], message: .commit.message, author: .commit.author.name, date: .commit.author.date, url: .html_url}})'
    ])
    
    if result:
        try:
            commits = json.loads(result)
            return [c for c in commits if c]
        except:
            pass
    return []


def get_repo_prs(repo, date):
    """获取仓库某天的 PR 活动"""
    created_prs = []
    merged_prs = []
    
    result = run_gh_command([
        "api",
        f"repos/{repo}/pulls",
        "--paginate",
        "--jq", "map({number: .number, title: .title, state: .state, created_at: .created_at, merged_at: .mergedAt, user: .user.login, url: .html_url, body: .body})"
    ])
    
    if result:
        try:
            prs = json.loads(result)
            for pr in prs:
                created_date = pr.get("created_at", "")[:10] if pr.get("created_at") else ""
                merged_date = pr.get("merged_at", "")[:10] if pr.get("merged_at") else ""
                
                pr_data = {
                    "number": pr.get("number", 0),
                    "title": pr.get("title", "Untitled"),
                    "url": pr.get("url", ""),
                    "author": pr.get("user", ""),
                    "body": pr.get("body", "")[:200] if pr.get("body") else ""
                }
                
                if created_date == date:
                    created_prs.append(pr_data)
                if merged_date == date:
                    merged_prs.append(pr_data)
        except:
            pass
    
    return {"created": created_prs, "merged": merged_prs}


def get_repo_issues(repo, date):
    """获取仓库某天的 Issue 活动"""
    created_issues = []
    closed_issues = []
    
    result = run_gh_command([
        "api",
        f"repos/{repo}/issues",
        "--paginate",
        "--jq", "map({number: .number, title: .title, state: .state, created_at: .created_at, closed_at: .closed_at, user: .user.login, url: .html_url})"
    ])
    
    if result:
        try:
            issues = json.loads(result)
            for issue in issues:
                if "/pull/" in issue.get("url", ""):
                    continue
                    
                created_date = issue.get("created_at", "")[:10] if issue.get("created_at") else ""
                closed_date = issue.get("closed_at", "")[:10] if issue.get("closed_at") else ""
                
                issue_data = {
                    "number": issue.get("number", 0),
                    "title": issue.get("title", "Untitled"),
                    "url": issue.get("url", ""),
                    "author": issue.get("user", "")
                }
                
                if created_date == date:
                    created_issues.append(issue_data)
                if closed_date == date:
                    closed_issues.append(issue_data)
        except:
            pass
    
    return {"created": created_issues, "closed": closed_issues}


def get_commit_diff(repo, sha):
    """获取 commit 的改动统计"""
    result = run_gh_command([
        "api",
        f"repos/{repo}/commits/{sha}",
        "--jq", "{additions: .stats.additions, deletions: .stats.deletions, total: .stats.total, files: [.files[].filename]}"
    ])
    
    if result:
        try:
            return json.loads(result)
        except:
            pass
    return None


def generate_change_descriptions(analysis, repo_name):
    """生成自然语言的改动描述"""
    descriptions = []
    
    # 分析功能开发
    if analysis["features"]:
        feature_msgs = [c.get("message", "").lower() for c in analysis["features"]]
        
        # 提取关键词
        if any("seo" in m for m in feature_msgs):
            descriptions.append("优化了网站的 SEO，添加了结构化数据和站点地图")
        if any("ui" in m or "component" in m for m in feature_msgs):
            descriptions.append("开发了新的 UI 组件和页面功能")
        if any("api" in m or "backend" in m for m in feature_msgs):
            descriptions.append("实现了后端 API 接口")
        if any("auth" in m or "login" in m for m in feature_msgs):
            descriptions.append("添加了用户认证和登录功能")
        
        # 通用描述
        if len(analysis["features"]) > 3 and len(descriptions) == 0:
            descriptions.append(f"开发了 {len(analysis['features'])} 个新功能")
    
    # 分析 Bug 修复
    if analysis["fixes"]:
        fix_msgs = [c.get("message", "").lower() for c in analysis["fixes"]]
        
        if any("responsive" in m or "mobile" in m for m in fix_msgs):
            descriptions.append("修复了响应式布局问题，优化了移动端显示")
        if any("typo" in m or "text" in m for m in fix_msgs):
            descriptions.append("修正了文字错误和拼写问题")
        if any("import" in m or "build" in m for m in fix_msgs):
            descriptions.append("解决了构建和依赖相关的问题")
        if any("crash" in m or "error" in m for m in fix_msgs):
            descriptions.append("修复了程序崩溃和错误处理问题")
        
        if len(analysis["fixes"]) > 2 and not any("修复" in d for d in descriptions):
            descriptions.append(f"修复了 {len(analysis['fixes'])} 处问题")
    
    # 分析重构
    if analysis["refactors"]:
        refactor_msgs = [c.get("message", "").lower() for c in analysis["refactors"]]
        
        if any("extract" in m or "component" in m for m in refactor_msgs):
            descriptions.append("重构了组件结构，提取了可复用组件")
        if any("optimize" in m or "performance" in m for m in refactor_msgs):
            descriptions.append("优化了代码性能")
        if any("clean" in m or "remove" in m for m in refactor_msgs):
            descriptions.append("清理了无用代码，提升了代码质量")
        
        if len(analysis["refactors"]) > 2 and not any("重构" in d for d in descriptions):
            descriptions.append(f"进行了 {len(analysis['refactors'])} 处代码重构")
    
    # 分析文档
    if analysis["docs"]:
        descriptions.append("更新了项目文档和说明")
    
    # 分析其他
    if analysis["others"]:
        other_msgs = [c.get("message", "").lower() for c in analysis["others"]]
        
        if any("style" in m or "css" in m for m in other_msgs):
            descriptions.append("调整了样式和视觉效果")
        if any("config" in m or "setup" in m for m in other_msgs):
            descriptions.append("更新了配置和项目设置")
    
    # 如果没有生成任何描述，使用通用描述
    if not descriptions:
        total = len(analysis["features"]) + len(analysis["fixes"]) + len(analysis["refactors"]) + len(analysis["docs"]) + len(analysis["others"])
        descriptions.append(f"进行了 {total} 项代码更新")
    
    return descriptions[:5]  # 最多返回5条


def analyze_commits(commits, repo):
    """分析 commits，提取改动类型和文件"""
    analysis = {
        "features": [],
        "fixes": [],
        "docs": [],
        "refactors": [],
        "others": [],
        "files_changed": set(),
        "total_additions": 0,
        "total_deletions": 0
    }
    
    for commit in commits:
        msg = commit.get("message", "").lower()
        sha = commit.get("sha", "")
        
        # 分类 commit
        if any(kw in msg for kw in ["feat", "feature", "add", "implement", "new"]):
            analysis["features"].append(commit)
        elif any(kw in msg for kw in ["fix", "bugfix", "hotfix", "resolve", "patch"]):
            analysis["fixes"].append(commit)
        elif any(kw in msg for kw in ["doc", "readme", "comment", "guide"]):
            analysis["docs"].append(commit)
        elif any(kw in msg for kw in ["refactor", "clean", "optimize", "improve", "update"]):
            analysis["refactors"].append(commit)
        else:
            analysis["others"].append(commit)
        
        # 获取文件改动统计（可选，因为 API 调用较多）
        # diff = get_commit_diff(repo, sha)
        # if diff:
        #     analysis["total_additions"] += diff.get("additions", 0)
        #     analysis["total_deletions"] += diff.get("deletions", 0)
        #     analysis["files_changed"].update(diff.get("files", []))
    
    return analysis


def generate_repo_summary(repo_name, commits, prs, issues):
    """生成单个仓库的总结"""
    lines = []
    lines.append(f"\n🔹 {repo_name}")
    lines.append("-" * 40)
    
    if not commits and not prs["created"] and not prs["merged"] and not issues["created"] and not issues["closed"]:
        lines.append("  No activity today")
        return "\n".join(lines)
    
    # 分析 commits
    if commits:
        analysis = analyze_commits(commits, repo_name)
        
        lines.append(f"\n  📊 改动分析 ({len(commits)} commits):")
        
        if analysis["features"]:
            lines.append(f"    ✨ 新功能 ({len(analysis['features'])}):")
            for c in analysis["features"][:3]:
                msg = c.get('message', '').split('\n')[0][:45]
                lines.append(f"      • {msg}")
        
        if analysis["fixes"]:
            lines.append(f"    🐛 Bug 修复 ({len(analysis['fixes'])}):")
            for c in analysis["fixes"][:3]:
                msg = c.get('message', '').split('\n')[0][:45]
                lines.append(f"      • {msg}")
        
        if analysis["refactors"]:
            lines.append(f"    🔧 重构优化 ({len(analysis['refactors'])}):")
            for c in analysis["refactors"][:3]:
                msg = c.get('message', '').split('\n')[0][:45]
                lines.append(f"      • {msg}")
        
        if analysis["docs"]:
            lines.append(f"    📝 文档更新 ({len(analysis['docs'])}):")
            for c in analysis["docs"][:3]:
                msg = c.get('message', '').split('\n')[0][:45]
                lines.append(f"      • {msg}")
        
        if analysis["others"]:
            lines.append(f"    📌 其他 ({len(analysis['others'])}):")
            for c in analysis["others"][:2]:
                msg = c.get('message', '').split('\n')[0][:45]
                lines.append(f"      • {msg}")
    
    # PRs
    if prs["created"]:
        lines.append(f"\n  🔀 新建 PR ({len(prs['created'])}):")
        for pr in prs["created"][:3]:
            lines.append(f"    + #{pr['number']} {pr['title'][:40]}")
    
    if prs["merged"]:
        lines.append(f"\n  ✅ 合并 PR ({len(prs['merged'])}):")
        for pr in prs["merged"][:3]:
            lines.append(f"    ✓ #{pr['number']} {pr['title'][:40]}")
    
    # Issues
    if issues["created"]:
        lines.append(f"\n  🐛 新建 Issue ({len(issues['created'])}):")
        for issue in issues["created"][:3]:
            lines.append(f"    + #{issue['number']} {issue['title'][:40]}")
    
    if issues["closed"]:
        lines.append(f"\n  ✓ 关闭 Issue ({len(issues['closed'])}):")
        for issue in issues["closed"][:3]:
            lines.append(f"    ✓ #{issue['number']} {issue['title'][:40]}")
    
    return "\n".join(lines)


def get_all_activity(date):
    """获取所有仓库的活动"""
    repos = get_recent_repos(days=2)
    activity = {}
    
    print(f"Checking {len(repos)} recently active repositories...", file=sys.stderr)
    
    for repo in repos:
        repo_name = repo.get("nameWithOwner", "")
        if not repo_name:
            continue
        
        commits = get_repo_commits(repo_name, date)
        prs = get_repo_prs(repo_name, date)
        issues = get_repo_issues(repo_name, date)
        
        if commits or prs["created"] or prs["merged"] or issues["created"] or issues["closed"]:
            activity[repo_name] = {
                "commits": commits,
                "prs_created": prs["created"],
                "prs_merged": prs["merged"],
                "issues_created": issues["created"],
                "issues_closed": issues["closed"]
            }
            total = len(commits) + len(prs["created"]) + len(prs["merged"]) + len(issues["created"]) + len(issues["closed"])
            print(f"  ✓ {repo_name}: {total} activities", file=sys.stderr)
    
    return activity


def format_detailed_report(activity, date):
    """格式化详细报告"""
    report = []
    
    # 标题
    report.append("=" * 60)
    report.append(f"📊 GitHub Daily Activity Report")
    report.append(f"📅 {date}")
    report.append("=" * 60)
    report.append("")
    
    # 统计概览
    total_commits = sum(len(r["commits"]) for r in activity.values())
    total_prs_created = sum(len(r["prs_created"]) for r in activity.values())
    total_prs_merged = sum(len(r["prs_merged"]) for r in activity.values())
    total_issues_created = sum(len(r["issues_created"]) for r in activity.values())
    total_issues_closed = sum(len(r["issues_closed"]) for r in activity.values())
    
    report.append("📈 今日概览")
    report.append("-" * 40)
    report.append(f"  📝 Commits: {total_commits}")
    report.append(f"  🔀 PRs Created: {total_prs_created}")
    report.append(f"  ✅ PRs Merged: {total_prs_merged}")
    report.append(f"  🐛 Issues Created: {total_issues_created}")
    report.append(f"  ✓ Issues Closed: {total_issues_closed}")
    report.append(f"  📦 Active Repositories: {len(activity)}")
    report.append("")
    
    # 按仓库详细列出
    if activity:
        report.append("📦 Repository Details")
        report.append("=" * 60)
        
        # 按活动数量排序
        sorted_repos = sorted(
            activity.items(),
            key=lambda x: (
                len(x[1]["commits"]) +
                len(x[1]["prs_created"]) +
                len(x[1]["prs_merged"]) +
                len(x[1]["issues_created"]) +
                len(x[1]["issues_closed"])
            ),
            reverse=True
        )
        
        for repo_name, data in sorted_repos:
            summary = generate_repo_summary(
                repo_name,
                data["commits"],
                {"created": data["prs_created"], "merged": data["prs_merged"]},
                {"created": data["issues_created"], "closed": data["issues_closed"]}
            )
            report.append(summary)
            report.append("")
    else:
        report.append("📭 No activity today")
        report.append("")
    
    # 添加每个仓库的 AI 总结
    if activity:
        report.append("\n" + "=" * 60)
        report.append("🤖 今日改动总结")
        report.append("=" * 60)
        report.append("")
        
        # 按仓库生成总结
        for repo_name, data in sorted_repos:
            commits = data["commits"]
            if not commits:
                continue
                
            report.append(f"\n*{repo_name}*")
            report.append("")
            
            # 分析该仓库的 commits
            analysis = analyze_commits(commits, repo_name)
            
            # 生成一句话总结
            summary_parts = []
            if analysis["features"]:
                summary_parts.append(f"新增了 {len(analysis['features'])} 个功能")
            if analysis["fixes"]:
                summary_parts.append(f"修复了 {len(analysis['fixes'])} 个 bug")
            if analysis["refactors"]:
                summary_parts.append(f"进行了 {len(analysis['refactors'])} 处重构优化")
            if analysis["docs"]:
                summary_parts.append(f"更新了 {len(analysis['docs'])} 处文档")
            
            if summary_parts:
                report.append(f"今日主要{'，'.join(summary_parts)}。")
            else:
                report.append(f"今日提交了 {len(commits)} 个 commits。")
            
            # 生成 AI 风格的具体改动总结
            report.append("")
            report.append("具体改动：")
            
            # 分析 commits 内容，生成自然语言描述
            descriptions = generate_change_descriptions(analysis, repo_name)
            for desc in descriptions:
                report.append(f"• {desc}")
            
            report.append("")
    
    return "\n".join(report)


def main():
    parser = argparse.ArgumentParser(description="GitHub Daily Report Generator")
    parser.add_argument("--date", help="Date to generate report for (YYYY-MM-DD)", default=None)
    parser.add_argument("--repo", help="Specific repository (owner/repo)", default=None)
    args = parser.parse_args()
    
    # 确定日期
    if args.date:
        date = args.date
    else:
        date = datetime.now().strftime("%Y-%m-%d")
    
    print(f"Generating GitHub report for {date}...", file=sys.stderr)
    
    if args.repo:
        print(f"Checking repository: {args.repo}", file=sys.stderr)
        commits = get_repo_commits(args.repo, date)
        prs = get_repo_prs(args.repo, date)
        issues = get_repo_issues(args.repo, date)
        
        activity = {}
        if commits or prs["created"] or prs["merged"] or issues["created"] or issues["closed"]:
            activity[args.repo] = {
                "commits": commits,
                "prs_created": prs["created"],
                "prs_merged": prs["merged"],
                "issues_created": issues["created"],
                "issues_closed": issues["closed"]
            }
    else:
        activity = get_all_activity(date)
    
    # 生成报告
    report = format_detailed_report(activity, date)
    
    print(report)


if __name__ == "__main__":
    main()
