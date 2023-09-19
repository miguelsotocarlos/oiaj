import json
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

    def post(self, url, can_fail=True, *args, **kwargs):
        resp = requests.post(f"{self.url()}{url}", *args, headers=self.make_headers(), **kwargs)
        if not can_fail and (resp.status_code < 200 or resp.status_code >= 300):
            raise Exception(f"POST {url}, failed ({resp.status_code}): {resp.content}")
        return resp

    def set_access_token(self, token):
        self.token = token

    def stop(self, extra_envs=None, old_version=False):
        utils.run('screen -S oiajudge -X quit')

    def start(self, extra_envs=None, old_version=False):
        env_vars = Config.env.copy()
        if extra_envs is not None:
            for k in extra_envs:
                env_vars[k] = str(extra_envs[k])
        binary = 'oiajudge_old' if old_version else 'oiajudge'
        if Config.debugger:
            if Config.autostart:
                run_command = f'/go/bin/dlv --listen :5050 --headless=true --continue --log=true --accept-multiclient --api-version=2 exec /workspaces/oiajudge/oiajudge/{binary}'
            else:
                run_command = f'/go/bin/dlv --listen :5050 --headless=true --log=true --accept-multiclient --api-version=2 exec /workspaces/oiajudge/oiajudge/{binary}'
        else:
            run_command = f'/workspaces/oiajudge/oiajudge/{binary}'
        utils.run_in_screen('oiajudge', run_command, env=env_vars)

        wait_for_service(lambda: self.get('/health'))

    def build(self):
        utils.run('go build -gcflags=\'all=-N -l\' -buildvcs=false',
                  cwd=Config.PROJECT_ROOT/'oiajudge')


class DatabaseService:
    def clear(self):
        utils.run('cmsDropDB -y')

    def interact(self):
        utils.run('psql -U postgres -d postgres -h db', env={
            "PGPASSWORD": "postgres"
        })

    def populate_with_contests(self, contests):
        Database.clear()
        Cms.init_db()
        Cms.add_empty_contest()
        for contest in contests:
            Cms.add_task(Config.TASK_PATH / contest)


class CmsService:
    def start(self):
        utils.run_in_screen('resource', "cmsResourceService -a ALL")
        utils.run_in_screen('log', "cmsLogService")

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
        utils.run(f"cp -r {utils.esc(task_dir)}/* /tmp/imported/")
        utils.run("chown -R cmsuser /tmp/imported")
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
