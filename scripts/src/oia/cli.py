from oia.config import Config
import oia.utils as utils
import oia.tests as tests
import sys

from oia.services import Database, Cms, Oia, All

COMMANDS = []


def command(prefix):
    def fun(f):
        COMMANDS.append((f, prefix))
        return f
    return fun


@command("up")
def _():
    All.down()
    Oia.build()
    Database.load_or_else(Database.populate_initial_data)
    Cms.start()
    Oia.start()


@command("db precompute")
def _():
    Database.populate_initial_data()


@command("db psql")
def _():
    Database.interact()


@command("db clear")
def _():
    Database.clear()


@command("test")
def _(*testnames):
    tests.run_tests(testnames)


@command("bridge reinstall")
def _():
    utils.run(
        "cp /workspaces/oiajudge/cms/cmsBridge.py /home/cms/cmscontrib/AddSubmission.py")
    utils.run("python3 setup.py install", cwd="/home/cms")


def is_prefix(a, b):
    if len(a) > len(b):
        return False
    for i in range(len(a)):
        if a[i] != b[i]:
            return False
    return True


def main():
    positional = []
    keyvalue = dict()
    for s in sys.argv[1:]:
        if not s.startswith('--'):
            positional.append(s)
        else:
            s = s[2:]
            v = s.split('=')
            if v[0] in keyvalue:
                raise Exception(f"Duplicated option {v[0]}")
            if len(v) == 2:
                keyvalue[v[0]] = v[1]
            else:
                keyvalue[v[0]] = True

    for f, prefix in COMMANDS:
        prefix = prefix.split(" ")
        if not is_prefix(prefix, positional):
            continue
        Config.from_flags(keyvalue)
        f(*positional[len(prefix):])
        return

    raise Exception(f"Unkown command {positional}")
