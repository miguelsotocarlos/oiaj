import base64
import unittest

from oia.services import Database, Cms, Oia, All
from oia.config import Config
import oia.utils as utils


def run_tests(tests):
    loader = unittest.TestLoader()
    suite = unittest.TestSuite()
    if tests == ():
        suite.addTests(loader.loadTestsFromTestCase(OiaTests))
    else:
        suite.addTest(loader.loadTestsFromNames(
            [f"oia.tests.OiaTests.{testname}" for testname in tests]))
    runner = unittest.TextTestRunner()
    runner.run(suite)


class OiaTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        Oia.build()

    def setUp(self):
        All.down()
        Oia.set_access_token(None)

    def test_user_creation(self):
        Database.populate_with_contests([])
        Oia.start()
        resp = Oia.post(f'/user/create', json={
            "username": "test_user",
            "password": "test_pass",
            "school": "escuela",
            "email": "lala@lala.com",
            "name": "Carlos",
        })
        uid = resp.json()["user_id"]
        token = resp.json()["token"]

        # no token
        resp = Oia.post(f'/user/get', json={"user_id": uid})
        self.assertEqual(resp.status_code, 400)
        resp = Oia.post(f'/token/validate', json={"user_id": uid})
        self.assertEqual(resp.status_code, 400)

        # well-formed but invalid token
        Oia.set_access_token("1:AAAAAQID")
        resp = Oia.post(f'/user/get', json={"user_id": uid})
        self.assertEqual(resp.status_code, 401)
        resp = Oia.post(f'/token/validate', json={"user_id": uid})
        self.assertEqual(resp.status_code, 401)

        # correct token
        Oia.set_access_token(token)
        resp = Oia.post(f'/user/get', json={"user_id": uid})
        self.assertEqual(resp.status_code, 200)
        self.assertEqual(resp.json()["username"], "test_user")
        resp = Oia.post(f'/token/validate', json={"user_id": uid})
        self.assertEqual(resp.status_code, 200)

        Oia.set_access_token(None)
        resp = Oia.post(f'/user/login', json={
            "username": "test_user",
            "password": "wrong_pass",
        })
        self.assertEqual(resp.status_code, 401)

        resp = Oia.post(f'/user/login', json={
            "username": "test_user",
            "password": "test_pass",
        })
        self.assertEqual(resp.status_code, 200)
        Oia.set_access_token(resp.json()["token"])

        resp = Oia.post(f'/user/get', json={"user_id": uid})
        self.assertEqual(resp.status_code, 200)
        self.assertEqual(resp.json()["username"], "test_user")

    def test_submission_cooldown(self):
        Database.populate_with_contests(["envido"])
        Cms.start()
        Oia.start(extra_envs={"OIAJ_SUBMISSION_COOLDOWN_MS": 10*1000})

        resp = Oia.post(f'/user/create', json={
            "username": "test_user",
            "password": "test_pass",
            "school": "escuela",
            "email": "lala@lala.com",
            "name": "Carlos",
        }).json()
        uid = resp["user_id"]
        Oia.set_access_token(resp["token"])

        Oia.post(f'/mock/time/set', json={"time": "2000-01-01T00:00:00Z"}, can_fail=False)
        with open(Config.TASK_PATH / 'envido.cpp', "rb") as f:
            source = f.read()

        resp = Oia.post(f'/submission/create', json={
            "task_id": 1,
            "user_id": uid,
            "sources": {
                "envido.%l": base64.b64encode(source).decode('utf-8')
            }
        })
        self.assertEqual(resp.status_code, 200)

        Oia.post(f'/mock/time/set', json={"time": "2000-01-01T00:00:09Z"}, can_fail=False)
        resp = Oia.post(f'/submission/create', json={
            "task_id": 1,
            "user_id": uid,
            "sources": {
                "envido.%l": base64.b64encode(source).decode('utf-8')
            }
        })
        self.assertEqual(resp.status_code, 429)

        Oia.post(f'/mock/time/set', json={"time": "2000-01-01T00:00:11Z"}, can_fail=False)
        resp = Oia.post(f'/submission/create', json={
            "task_id": 1,
            "user_id": uid,
            "sources": {
                "envido.%l": base64.b64encode(source).decode('utf-8')
            }
        })
        self.assertEqual(resp.status_code, 200)

    def test_submission_envido(self):
        Database.populate_with_contests(["envido"])
        Cms.start()
        Oia.start()

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

    def test_submission_frutales(self):
        Database.populate_with_contests(["frutales"])
        Oia.start()
        Cms.start()

        with open(Config.TASK_PATH / 'frutales.cpp', "rb") as f:
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
                "frutales.%l": base64.b64encode(source).decode('utf-8')
            }
        })
        self.assertEqual(resp.status_code, 200)

        def submission_ready():
            submissions = Oia.post('/submissions/get', json={"user_id": uid, "task_id": 1}).json()["submissions"]
            return len(submissions) > 0 and submissions[0]["submission_status"] == "scored"

        utils.wait_for(submission_ready)

        submission = Oia.post('/submissions/get', json={"user_id": uid, "task_id": 1}).json()["submissions"][0]
        print(submission)

        self.assertEqual(submission["result"]["score"], {"score": 95, "max_score": 100})

        resp = Oia.post(f'/user/get', json={"user_id": uid}).json()
        # max_score * score_multiplier
        self.assertEqual(resp["score"], 95*4)

    def test_get_task(self):
        Database.populate_with_contests(["envido"])
        Oia.start()

        def task_ready():
            tasks = Oia.post('/task/get', json={}).json()["tasks"]
            return tasks is not None
        utils.wait_for(task_ready)
        task = Oia.post('/task/get', json={}).json()["tasks"][0]
        self.assertEqual(task["name"], "envido")
        self.assertEqual(task["max_score"], 2)
        self.assertEqual(task["tags"], ['año:2023', 'certamen:selectivo'])
        self.assertEqual(task["submission_format"], ["envido.%l"])

        task_statement = Oia.get(f'/task/statement/{task["id"]}').content

        actual_statement = (Config.TASK_PATH / 'envido' / 'envido.pdf').read_bytes()
        self.assertEqual(task_statement, actual_statement)
