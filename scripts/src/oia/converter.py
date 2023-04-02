import csv
import json
import os
import shutil
import oia.utils as utils


# Converts a dump produced by cmsDumpExporter into ready-to-import directories
# in the argentina_loader format. Requires a tags.cvs file in the format
#     tag,problem
# which can be prodced directly from the database
def convert(path, dst):
    with open(path / 'contest.json') as f:
        contest = json.load(f)

    tags = dict()
    with open(path / 'tags.csv') as f:
        reader = csv.reader(f)
        for row in reader:
            problem = row[1]
            tag = row[0]
            if problem not in tags:
                tags[problem] = []
            tags[problem].append(tag)
    for k, v in contest.items():
        if k.startswith("_"):
            continue
        if v["_class"] == "Task":
            process_task(path, contest, k, dst, tags)


def process_task(src, contest, k, dst, tags):
    task = contest[k]
    print(task['name'])
    if task["name"] not in tags:
        tags[task["name"]] = []

    dst = dst / task["name"]
    os.makedirs(dst, exist_ok=True)
    os.makedirs(dst/'kits', exist_ok=True)
    os.makedirs(dst/'casos', exist_ok=True)
    os.makedirs(dst/'graders', exist_ok=True)

    dataset = contest[task["active_dataset"]]

    for codename, k in dataset["testcases"].items():
        testcase = contest[k]
        shutil.copy(src / 'files' / testcase['input'], dst/'casos'/f'{codename}.in')
        shutil.copy(src / 'files' / testcase['output'], dst/'casos'/f'{codename}.dat')

    utils.run('zip -r casos.zip *', cwd=dst/'casos')
    utils.run('mv casos/casos.zip .', cwd=dst)
    shutil.rmtree(dst/'casos')

    for filename, k in task["attachments"].items():
        attachment = contest[k]
        shutil.copy(src / 'files' / attachment['digest'], dst/'kits'/filename)

    for _, k in task["statements"].items():
        statement = contest[k]
        shutil.copy(src / 'files' / statement['digest'], dst/f'{task["name"]}.pdf')

    for name, k in dataset["managers"].items():
        fil = src / 'files' / contest[k]['digest']
        if name == "checker":
            shutil.copy(fil, dst / 'checker')
        else:
            shutil.copy(fil, dst / 'graders' / name)

    with open(dst / 'config.json', 'w') as f:
        if "token_max_number" in task and task["token_max_number"] is not None:
            multiplier = task["token_max_number"] / 100.0
        else:
            multiplier = 1
        f.write(json.dumps({
            "name": task["name"],
            "title": task["title"],
            "oiaj": {
                "tags": tags[task["name"]],
                "multiplier": multiplier,
            },
            "score_type": dataset["score_type"],
            "score_parameters": json.loads(dataset["score_type_parameters"]),
            "task_type": dataset["task_type"],
            "task_type_parameters": json.loads(dataset["task_type_parameters"]),
            "time_limit": dataset["time_limit"],
            "memory_limit": dataset["memory_limit"],
        }))
