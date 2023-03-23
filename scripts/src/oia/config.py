from pathlib import Path


class Configuration:
    def from_flags(self, flags):
        def read_with_default(flagname, default):
            if flagname in flags:
                return flags[flagname]
            else:
                return default

        self.debugger = read_with_default('debugger', False)
        self.autostart = read_with_default('autostart', True)


Config = Configuration()


Config.PROJECT_ROOT = Path('/workspaces/oiajudge')
Config.DUMP_PATH = Path('/workspaces/oiajudge/testdata/dumps')
Config.TASK_PATH = Path('/workspaces/oiajudge/testdata/tasks')

Config.env = {
    "OIAJ_SUBMITTER_PORT": "1366",
    "OIAJ_DB_CONNECTION_STRING": "postgresql://postgres:postgres@db:5432/postgres",
    "OIAJ_SERVER_PORT": "1367",
    "OIAJ_CMS_BRIDGE_ADDRESS": "http://localhost:1366",
}
