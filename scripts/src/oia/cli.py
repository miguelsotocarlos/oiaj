from pathlib import Path
from oia.config import Config
import oia.tests as tests
import oia.migration_tests as migration_tests
import sys

from oia.services import Database, Cms, Oia, All
import oia.utils as utils
from oia.converter import convert

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
    if Config.clear_db:
        Database.populate_with_contests([])
    if Config.add_test_data:
        Database.populate_with_contests(['frutales', 'envido'])
    Cms.start()
    Oia.start()


@command("task add")
def _(*dirs):
    for dir in dirs:
        Cms.add_task(dir)


@command("convert")
def _(src, dst):
    convert(Path(src), Path(dst))


@command("db psql")
def _():
    Database.interact()


@command("reinstall")
def _():
    utils.run(
        "cp /workspaces/oiajudge/cms/argentina_loader.py /home/cms/cmscontrib/loaders/argentina_loader.py")
    utils.run("python3 setup.py install", cwd="/home/cms")


@command("db clear")
def _():
    Database.clear()


@command("test")
def _(*testnames):
    tests.run_tests(testnames)


@command("migration test")
def _(*testnames):
    migration_tests.run_tests(testnames)


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
