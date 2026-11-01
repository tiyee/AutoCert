import contextlib
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

import renewal_report as report


class RenewalReportTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        env = patch.dict(os.environ, {"AUTOCERT_REPORT_DIR": str(self.root / "results"),
                                     "GITHUB_STEP_SUMMARY": str(self.root / "summary.md")}, clear=True)
        env.start()
        self.addCleanup(env.stop)

    def run_program(self, code, task="renew_1", variable="DOMAINTOOLS"):
        command = [sys.executable, "-c", code, "--domain=tools.example.invalid",
                   "--dns-access-key-secret=fake+secret/", "--cdn-security-token=fake-token"]
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            result = report.run_task(task, variable, command)
        return result, output.getvalue()

    def test_failure_has_domain_code_and_safe_log(self):
        result, output = self.run_program(
            "import sys;print('fake+secret/ fake-token fake%2Bsecret%2F');"
            "print('-----BEGIN PRIVATE KEY-----');print('private-body');"
            "print('-----END PRIVATE KEY-----');print('::error::injected');"
            "print('apply: Aliyun DNS create failed (SignatureDoesNotMatch)');sys.exit(7)")
        self.assertEqual(result, 1)
        self.assertIn("tools.example.invalid", output)
        self.assertIn("SignatureDoesNotMatch", output)
        for private in ("fake+secret/", "fake-token", "fake%2Bsecret%2F", "private-body"):
            self.assertNotIn(private, output)
        self.assertEqual(sum(line.startswith("::error") for line in output.splitlines()), 1)
        pause = next(line for line in output.splitlines() if line.startswith("::stop-commands::"))
        resume = "::" + pause.split("::")[-1] + "::"
        self.assertLess(output.index(pause), output.index("::error::injected"))
        self.assertLess(output.index("::error::injected"), output.index(resume))
        record = json.loads((report.report_dir() / "renew_1.json").read_text())
        self.assertEqual(record["exit_code"], 7)
        self.assertIn("SignatureDoesNotMatch", "\n".join(record["log"]))
        self.assertNotIn("private-body", json.dumps(record))

    def test_success_and_missing_program(self):
        result, _ = self.run_program("print('Certificate no need renew')")
        self.assertEqual(result, 0)
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            result = report.run_task("renew_2", "DOMAINOTHER", [str(self.root / "missing"), "--domain=other.example.invalid"])
        self.assertEqual(result, 1)
        self.assertIn("无法启动", output.getvalue())
        self.assertEqual(json.loads((report.report_dir() / "renew_2.json").read_text())["exit_code"], 127)

    def test_multiline_error_in_summary_and_other_domains_continue(self):
        self.run_program("import sys;print('apply: validation failed');print('SignatureDoesNotMatch');sys.exit(1)")
        self.run_program("print('Successfully deployed')", task="renew_2", variable="DOMAINOTHER")
        os.environ["AUTOCERT_STEPS"] = json.dumps({"compile": {"outcome": "success"},
            "renew_1": {"outcome": "failure", "conclusion": "success"},
            "renew_2": {"outcome": "success"}, "renew_3": {"outcome": "skipped"}})
        with patch.object(report, "job_links", return_value={"tools.example.invalid": "https://example.invalid/job/1#step:5:1"}):
            with contextlib.redirect_stdout(io.StringIO()):
                result = report.summarize()
        self.assertEqual(result, 1)
        summary = Path(os.environ["GITHUB_STEP_SUMMARY"]).read_text()
        for text in ("DOMAINTOOLS", "renew_1", "SignatureDoesNotMatch", "DOMAINOTHER", "成功", "已跳过", "#step:5:1"):
            self.assertIn(text, summary)
        self.assertIn("失败项数：1", summary)

    def test_summary_success_and_preparation_failure(self):
        self.run_program("print('Certificate no need renew')")
        os.environ["AUTOCERT_STEPS"] = '{"renew_1":{"outcome":"success"}}'
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(report.summarize(), 0)
        os.environ["AUTOCERT_STEPS"] = '{"compile":{"outcome":"failure"},"renew_1":{"outcome":"skipped"}}'
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(report.summarize(), 1)
        self.assertIn("compile", Path(os.environ["GITHUB_STEP_SUMMARY"]).read_text())

    def test_missing_records_are_not_reported_as_success(self):
        for outcome in ("success", "failure", "cancelled", "unknown"):
            os.environ["AUTOCERT_STEPS"] = json.dumps({"renew_1": {"outcome": outcome}})
            with contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(report.summarize(), 1)
        os.environ["AUTOCERT_STEPS"] = '{}'
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(report.summarize(), 1)

    def test_corrupt_record_is_reported_as_failure(self):
        report.report_dir().mkdir()
        (report.report_dir() / "renew_1.json").write_text("incomplete JSON")
        os.environ["AUTOCERT_STEPS"] = '{"renew_1":{"outcome":"success"}}'
        with contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(report.summarize(), 1)
        self.assertIn("未生成执行记录", Path(os.environ["GITHUB_STEP_SUMMARY"]).read_text())

    def test_job_links_use_current_attempt_and_have_fallback(self):
        os.environ.update(GH_TOKEN="fake-gh-token", GITHUB_REPOSITORY="owner/repo",
                          GITHUB_RUN_ID="1", GITHUB_RUN_ATTEMPT="2", GITHUB_JOB="build")
        data = {"jobs": [{"name": "build", "html_url": "https://example.invalid/job/1",
                           "steps": [{"name": "tools.example.invalid", "number": 5}]}]}
        class Response(io.StringIO):
            pass
        with patch.object(report.urllib.request, "urlopen", return_value=Response(json.dumps(data))) as request:
            self.assertEqual(report.job_links()["tools.example.invalid"], "https://example.invalid/job/1#step:5:1")
            self.assertIn("/attempts/2/jobs", request.call_args.args[0].full_url)
        with patch.object(report.urllib.request, "urlopen", side_effect=OSError("offline")):
            self.assertEqual(report.job_links(), {})

    def test_escaped_pem_and_annotation_escaping(self):
        self.assertNotIn("private-value", report.Redactor().clean("request -----BEGIN PRIVATE KEY-----\\nprivate-value\\n-----END PRIVATE KEY-----"))
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            report.annotation("error\n::warning::data%", "title,:\n")
        self.assertEqual(len(output.getvalue().splitlines()), 1)
        self.assertIn("%0A", output.getvalue())
        self.assertIn("%25", output.getvalue())


if __name__ == "__main__":
    unittest.main()
