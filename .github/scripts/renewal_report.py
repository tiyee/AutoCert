#!/usr/bin/env python3
"""为独立续期步骤提供脱敏日志、错误标注和运行汇总。"""

import argparse
import contextlib
from collections import deque
import html
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import urllib.request
import urllib.parse
import uuid


PREPARATION_STEPS = {
    "checkout": "Checkout", "setup_go": "Set up Go",
    "dependencies": "Depedency", "compile": "Build",
}


def annotation(message, title="AutoCert"):
    def escape(value):
        return str(value).replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")

    title = escape(title).replace(",", "%2C").replace(":", "%3A")
    print(f"::error title={title}::{escape(message)}", flush=True)


@contextlib.contextmanager
def plain_log():
    # Actions 可能识别行内命令，按官方协议暂停解析外部输出。
    token = uuid.uuid4().hex
    print(f"::stop-commands::{token}", flush=True)
    try:
        yield
    finally:
        print(f"::{token}::", flush=True)


def argument(command, name, default=""):
    for index, value in enumerate(command):
        if value in ("--" + name, "-" + name) and index + 1 < len(command):
            return command[index + 1]
        if value.startswith(("--" + name + "=", "-" + name + "=")):
            return value.split("=", 1)[1]
    return default


class Redactor:
    def __init__(self, command=()):
        values = [value for key, value in os.environ.items()
                  if re.search(r"ACCESS.*KEY|SECRET|TOKEN", key, re.I) and value]
        for side in ("dns", "cdn"):
            for field in ("access-key-id", "access-key-secret", "security-token"):
                value = argument(command, f"{side}-{field}")
                if value:
                    values.append(value)
        variants = set()
        for value in values:
            for item in (value, value.strip()):
                if item:
                    variants.update((item, urllib.parse.quote(item, safe=""), urllib.parse.quote_plus(item)))
        self.values = sorted(variants, key=len, reverse=True)
        self.in_pem = False

    def clean(self, line):
        # 同时处理逐行 PEM 和 SDK 错误中包含的单行、转义 PEM。
        if self.in_pem:
            if "-----END " in line:
                self.in_pem = False
            return "[证书/私钥内容已隐藏]"
        start = line.find("-----BEGIN ")
        if start >= 0:
            self.in_pem = "-----END " not in line[start:]
            line = line[:start] + "[证书/私钥内容已隐藏]"
        for value in self.values:
            line = line.replace(value, "***")
        # 过滤终端控制符；工作流命令解析由 plain_log 暂停。
        return re.sub(r"[\x00-\x08\x0b-\x1f\x7f]", "", line).rstrip()[:8000]


def report_dir():
    configured = os.environ.get("AUTOCERT_REPORT_DIR")
    if configured:
        return Path(configured)
    return Path(os.environ.get("RUNNER_TEMP", "/tmp")) / "autocert-renewals"


def run_task(task, variable, command):
    if not re.fullmatch(r"renew_[0-9]+", task):
        raise ValueError("invalid renewal task ID")
    if command and command[0] == "--":
        command = command[1:]
    if not command:
        raise ValueError("missing renewal command")
    redactor = Redactor(command)
    domain = redactor.clean(argument(command, "domain")).replace("\n", " ") or "[域名未配置]"
    variable = redactor.clean(variable).replace("\n", " ")
    tail = deque(maxlen=30)
    with plain_log():
        print(f"[{task}] 开始续期检查：{domain}（变量 {variable}）", flush=True)
        try:
            with subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                  text=True, errors="replace") as process:
                for raw in process.stdout:
                    line = redactor.clean(raw)
                    if line:
                        tail.append(line)
                        print(f"[{task} {domain}] {line}", flush=True)
                code = process.wait()
        except OSError as error:
            code = 127
            line = redactor.clean(f"无法启动续期程序：{error}")
            tail.append(line)
            print(f"[{task} {domain}] {line}", flush=True)
    record = {"task": task, "variable": variable, "domain": domain,
              "exit_code": code, "log": list(tail)}
    directory = report_dir()
    directory.mkdir(parents=True, exist_ok=True, mode=0o700)
    pending = directory / f"{task}.tmp"
    pending.write_text(json.dumps(record, ensure_ascii=False), encoding="utf-8")
    pending.replace(directory / f"{task}.json")
    if code:
        detail = "\n".join(tail)[-12000:] or "程序未输出错误内容，请查看此步骤日志。"
        annotation(f"{domain}（{variable}）续期失败，退出码 {code}\n{detail}", title=task)
    else:
        with plain_log():
            print(f"[{task}] {domain} 检查完成", flush=True)
    return 1 if code else 0


def job_links():
    """链接查询失败不影响本地错误汇总；不输出 API 响应或 Token。"""
    token = os.environ.get("GH_TOKEN")
    repo = os.environ.get("GITHUB_REPOSITORY")
    run = os.environ.get("GITHUB_RUN_ID")
    attempt = os.environ.get("GITHUB_RUN_ATTEMPT", "1")
    if not token or not repo or not run:
        return {}
    url = (f"{os.environ.get('GITHUB_API_URL', 'https://api.github.com')}/repos/{repo}"
           f"/actions/runs/{run}/attempts/{attempt}/jobs?per_page=100")
    request = urllib.request.Request(url, headers={"Authorization": f"Bearer {token}",
                                                  "Accept": "application/vnd.github+json"})
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            jobs = json.load(response)["jobs"]
        for job in jobs:
            if job["name"] == os.environ.get("GITHUB_JOB", "build"):
                return {step["name"]: f"{job['html_url']}#step:{step['number']}:1"
                        for step in job.get("steps", [])}
    except (OSError, ValueError, KeyError, TypeError):
        pass
    return {}


def summarize():
    steps = json.loads(os.environ.get("AUTOCERT_STEPS", "{}"))
    links = job_links()
    rows, failures = [], []
    for task, status in steps.items():
        if not task.startswith("renew_"):
            if status.get("outcome") in ("failure", "cancelled"):
                label = PREPARATION_STEPS.get(task, task)
                failures.append((f"{label} ({task})", "准备步骤失败或取消，请查看该步骤日志。",
                                 links.get(label)))
            continue
        path = report_dir() / f"{task}.json"
        outcome = status.get("outcome", "unknown")
        try:
            record = json.loads(path.read_text(encoding="utf-8")) if path.exists() and outcome != "skipped" else {}
        except (OSError, ValueError):
            record = {}
        domain = record.get("domain", "[未取得域名]")
        variable = record.get("variable", task)
        failed = outcome not in ("success", "skipped") or bool(record.get("exit_code"))
        if outcome == "success" and not record:
            failed = True
        state = "失败" if failed else ("已跳过" if outcome == "skipped" else "成功")
        title = f"{domain}（{variable} / {task}）"
        # 步骤名称保留域名，缺少变量时保留变量标识。
        link = links.get(domain) or links.get(variable) or links.get(task)
        rows.append((domain, variable, task, state, record.get("exit_code", "—"), link))
        if failed:
            detail = "\n".join(record.get("log", []))[-12000:] or "未生成执行记录，请打开对应步骤日志。"
            failures.append((title, detail, link))
    text = ["## AutoCert 续期结果", "", "| 域名 | 配置变量 / 步骤 | 结果 | 退出码 | 日志 |",
            "| --- | --- | --- | --- | --- |"]
    def cell(value):
        return html.escape(str(value)).replace("|", "&#124;").replace("\n", " ")
    for domain, variable, task, state, code, link in rows:
        log = f"[查看日志]({link})" if link else "按步骤名查看运行日志"
        text.append(f"| {cell(domain)} | {cell(variable)} / {cell(task)} | {state} | {code} | {log} |")
    for title, detail, link in failures:
        text.extend(["", f"### {cell(title)}", "", f"<pre>{html.escape(detail)}</pre>"])
        annotation(f"{title}\n{detail}" + (f"\n日志：{link}" if link else ""))
    if not rows and not failures:
        failures.append(("summary", "没有取得续期步骤结果。", None))
        annotation("没有取得续期步骤结果，请检查工作流配置。")
    text.extend(["", f"失败项数：{len(failures)}。"])
    content = "\n".join(text) + "\n"
    destination = os.environ.get("GITHUB_STEP_SUMMARY")
    if destination:
        with open(destination, "a", encoding="utf-8") as output:
            output.write(content)
    with plain_log():
        print(content)
    return 1 if failures else 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="mode", required=True)
    runner = commands.add_parser("run")
    runner.add_argument("--task", required=True)
    runner.add_argument("--variable", required=True)
    runner.add_argument("command", nargs=argparse.REMAINDER)
    commands.add_parser("summary")
    args = parser.parse_args()
    if args.mode == "run":
        return run_task(args.task, args.variable, args.command)
    return summarize()


if __name__ == "__main__":
    sys.exit(main())
