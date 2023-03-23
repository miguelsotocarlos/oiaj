import json
import os
from oia.config import Config
from oia import utils
import requests
import time


def wait_for_service(service_healthy):
    start_time = time.time()
    while True:
        try:
            service_healthy()
            break
        except requests.ConnectionError:
            pass
        time.sleep(0.01)
        now = time.time()
        if now - start_time > 30:
            print("Warning, waited for service more than 30 seconds")


class OiaService:
    def __init__(self):
        self.token = None

    def url(self):
        return 'http://localhost:1367'

    def make_headers(self):
        if self.token is None:
            return dict()
        else:
            return {
                "Authorization": f"Bearer {self.token}"
            }

    def get(self, url, *args, **kwargs):
        return requests.get(f"{self.url()}{url}", *args, headers=self.make_headers(), **kwargs)

    def post(self, url, *args, **kwargs):
        return requests.post(f"{self.url()}{url}", *args, headers=self.make_headers(), **kwargs)

    def set_access_token(self, token):
        self.token = token

    def start(self):
        if Config.debugger:
            if Config.autostart:
                run_command = '/go/bin/dlv --listen :5050 --headless=true --continue --log=true --accept-multiclient --api-version=2 exec /workspaces/oiajudge/oiajudge/oiajudge'
            else:
                run_command = '/go/bin/dlv --listen :5050 --headless=true --log=true --accept-multiclient --api-version=2 exec /workspaces/oiajudge/oiajudge/oiajudge'
        else:
            run_command = '/workspaces/oiajudge/oiajudge/oiajudge'
        utils.run_in_screen('oiajudge', run_command, env=Config.env)

        wait_for_service(lambda: self.get('/health'))

    def build(self):
        utils.run('go build -gcflags=\'all=-N -l\' -buildvcs=false',
                  cwd=Config.PROJECT_ROOT/'oiajudge')


class DatabaseService:
    def dump_exists(self, name="dump") -> bool:
        path = str(Config.DUMP_PATH/f"{name}.sql")
        try:
            os.stat(path)
        except OSError:
            return False
        else:
            return True

    def clear(self):
        utils.run('cmsDropDB -y')

    def save(self, name="dump"):
        os.makedirs(Config.DUMP_PATH, exist_ok=True)
        path = str(Config.DUMP_PATH/f"{name}.sql")
        utils.run(f'pg_dump -U postgres -d postgres -h db > {utils.esc(path)}', env={
            "PGPASSWORD": "postgres"
        })

    def load(self, name="dump"):
        self.clear()
        path = str(Config.DUMP_PATH/f"{name}.sql")
        utils.run(f'psql -U postgres -d postgres -h db < {utils.esc(path)} > /dev/null', env={
            "PGPASSWORD": "postgres"
        })

    def load_or_else(self, compute, name="dump"):
        if not self.dump_exists(name):
            compute()
        self.load(name)

    def interact(self):
        utils.run('psql -U postgres -d postgres -h db', env={
            "PGPASSWORD": "postgres"
        })

    def populate_initial_data(self):
        Database.clear()
        Cms.init_db()
        Cms.add_empty_contest()
        Cms.add_task(Config.TASK_PATH / 'envido.zip')
        Database.save()


class CmsService:
    def start(self):
        utils.run_in_screen('resource', "cmsResourceService -a ALL")
        utils.run_in_screen('log', "cmsLogService")
        utils.run_in_screen('bridge', "cmsAddSubmission", env=Config.env)
        wait_for_service(lambda: requests.get('http://localhost:1366/health'))

    def init_db(self):
        utils.run("cmsInitDB")

    def add_empty_contest(self):
        utils.run('rm /tmp/imported -r')
        utils.run('mkdir -p /tmp/imported')
        contest = json.dumps({"name": "oiaj", "description": "oiaj", "tasks": [
        ], "users": [], "token_mode": "disabled"})
        utils.run(f'echo {utils.esc(contest)} > /tmp/imported/contest.yaml')
        utils.run(f'cmsImportContest -L italy_yaml /tmp/imported')

    def add_task(self, task_dir):
        task_dir = str(task_dir)
        utils.run('rm /tmp/imported -r')
        utils.run('mkdir -p /tmp/imported')
        utils.run(f"cp {utils.esc(task_dir)} /tmp/imported/task.zip")
        utils.run("chown -R cmsuser /tmp/imported")
        utils.run("unzip /tmp/imported/task.zip -d /tmp/imported")
        utils.run("cmsImportTask -c 1 -L argentina /tmp/imported")


class AllService:
    def down(self):
        utils.run('pkill screen')
        utils.run('pkill dlv')
        utils.run('screen -wipe')


Database = DatabaseService()
Cms = CmsService()
Oia = OiaService()
All = AllService()
