import base64
import os
import time
import unittest

from oia.services import Database, Cms, Oia, All
from oia.config import Config
import oia.utils as utils


def run_tests(tests):
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()
    if tests == ():
        suite.addTests(loader.loadTestsFromTestCase(OiaMigrationTests))
    else:
        suite.addTest(loader.loadTestsFromNames(
            [f"oia.tests.OiaTests.{testname}" for testname in tests]))
    runner = unittest.TextTestRunner()
    runner.run(suite)


class OiaMigrationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        git_ref = Config.git_ref
        if git_ref is None:
            raise RuntimeError('Please specify git ref to test against with --git-ref')
        utils.run(f'git checkout {git_ref}')
        Oia.build()

        utils.run(f'git checkout main')
        os.rename(Config.PROJECT_ROOT / 'oiajudge' / 'oiajudge', Config.PROJECT_ROOT / 'oiajudge' / 'oiajudge_old')
        Oia.build()

    def setUp(self):
        All.down()
        Oia.set_access_token(None)

    def test_submission_permanence(self):
        Database.populate_with_contests(["envido"])
        Cms.start()
        Oia.start(old_version=True)

        with open(Config.TASK_PATH / 'envido.cpp', "rb") as f:
            source = f.read()

        resp = Oia.post(f'/user/create', json={
            "username": "test_user",
            "password": "test_pass",
            "school": "escuela",
            "email": "lala@lala.com",
            "name": "Carlos",
        }).json()
        uid = resp["user_id"]
        Oia.set_access_token(resp["token"])
        resp = Oia.post(f'/submission/create', json={
            "task_id": 1,
            "user_id": resp["user_id"],
            "sources": {
                "envido.%l": base64.b64encode(source).decode('utf-8')
            }
        })
        self.assertEqual(resp.status_code, 200)

        def submission_ready():
            submissions = Oia.post('/submissions/get', json={"user_id": uid, "task_id": 1}).json()["submissions"]
            return len(submissions) > 0 and submissions[0]["submission_status"] == "scored"

        utils.wait_for(submission_ready)

        submission = Oia.post('/submissions/get', json={"user_id": uid, "task_id": 1}).json()["submissions"][0]

        self.assertEqual(submission["result"]["score"], {"score": 2, "max_score": 2})

        resp = Oia.post(f'/user/get', json={"user_id": uid}).json()
        # max_score * score_multiplier
        self.assertEqual(resp["score"], 8)

        # Now update the server
        Oia.stop()
        Oia.start()

        submission = Oia.post('/submissions/get', json={"user_id": uid, "task_id": 1}).json()["submissions"][0]
        self.assertEqual(submission["result"]["score"], {"score": 2, "max_score": 2})
        resp = Oia.post(f'/user/get', json={"user_id": uid}).json()
        # max_score * score_multiplier
        self.assertEqual(resp["score"], 8)

        time.sleep(10)

        def task_updated():
            task = Oia.post('/task/get', json={}).json()["tasks"][0]
            return task["attachments"] != []

        utils.wait_for(task_updated)

        task = Oia.post('/task/get', json={}).json()["tasks"][0]
        self.assertEqual(task["name"], "envido")
        self.assertEqual(task["max_score"], 2)
        self.assertEqual(task["tags"], ['año:2023', 'certamen:selectivo'])
        self.assertEqual(task["submission_format"], ["envido.%l"])
        self.assertEqual(set(task["attachments"]), {"envido-cpp.zip", "envido-java.zip"})

        for name in {"envido-cpp.zip", "envido-java.zip"}:
            resp = Oia.get(f'/task/attachment', params={
                'task_id': 1,
                'filename': name
            })
            self.assertEqual(resp.status_code, 200)
            attachment = resp.content
            actual_attachment = (Config.TASK_PATH / 'envido' / 'kits' / name).read_bytes()
            self.assertEqual(attachment, actual_attachment)
