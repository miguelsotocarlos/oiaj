import subprocess


def esc(s):
    escaped = s.replace('\\', '\\\\').replace('"', '\\\"')
    return f'"{escaped}"'


def run(command, env=None, cwd=None):
    print("RUNNING: " + command)
    subprocess.run(["bash", "-c", command], env=env, cwd=cwd)


def clear_screens(command):
    run("kilall screen")
    run("screen -wipe")


def run_in_screen(screen_name, command, env=None):
    run(f"screen -Smd {esc(screen_name)} sh -c {esc(command + '; sleep infinity')}", env=env)
