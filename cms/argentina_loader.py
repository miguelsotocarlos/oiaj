#!/usr/bin/env python
# -*- coding: utf-8 -*-

# Contest Management System - http://cms-dev.github.io/
# Copyright © 2010-2014 Giovanni Mascellani <mascellani@poisson.phc.unipi.it>
# Copyright © 2010-2018 Stefano Maggiolo <s.maggiolo@gmail.com>
# Copyright © 2010-2012 Matteo Boscariol <boscarim@hotmail.com>
# Copyright © 2013-2018 Luca Wehrstedt <luca.wehrstedt@gmail.com>
# Copyright © 2014-2018 William Di Luigi <williamdiluigi@gmail.com>
# Copyright © 2015 Luca Chiodini <luca@chiodini.org>
# Copyright © 2016 Andrea Cracco <guilucand@gmail.com>
# Copyright © 2018 Edoardo Morassutto <edoardo.morassutto@gmail.com>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

# Modified on 2023 by Carlos Miguel Soto <miguelsotocarlos@gmail.com>

# This file is meant to run inside a cms instance, so we have to diable
# all the missing import errors:
# flake8: noqa
# type: ignore

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function
from __future__ import unicode_literals

import io
import logging
import os
import os.path
import sys
import yaml
from datetime import timedelta
import re

from cms.db import Contest, User, Task, Statement, Attachment, Team, Dataset, \
    Manager, Testcase
import cms.db
from cmscommon.datetime import make_datetime
from cmscontrib import touch

import json
import subprocess

from .base_loader import ContestLoader, TaskLoader, UserLoader, TeamLoader


logger = logging.getLogger(__name__)


# Patch PyYAML to make it load all strings as unicode instead of str
# (see http://stackoverflow.com/questions/2890146).
def construct_yaml_str(self, node):
    return self.construct_scalar(node)


yaml.Loader.add_constructor("tag:yaml.org,2002:str", construct_yaml_str)
yaml.SafeLoader.add_constructor("tag:yaml.org,2002:str", construct_yaml_str)


def getmtime(fname):
    return os.stat(fname).st_mtime


def get_with_default(src, name, default):
    try:
        return src[name]
    except KeyError:
        return default


def load(dst, src, name, default, conv=lambda i: i):
    dst[name] = conv(get_with_default(src, name, default))


def make_timedelta(t):
    return timedelta(seconds=t)


def compile_checker(path):
    p = os.path.split(path)[0]
    subprocess.call([
        "g++",
        "-static",
        "--std=gnu++11",
        "-o",
        os.path.join(p, 'checker'),
        path,
    ])


def extract_cases(path, dest):
    subprocess.call([
        "unzip",
        "-u",
        path,
        "-d",
        dest,
    ])


class ArgLoader(ContestLoader, TaskLoader, UserLoader, TeamLoader):
    """
    TASK_NAME/
        TASK_NAME.pdf
        kits/
            TASK_NAME-cpp.zip
            TASK_NAME-java.zip
        graders/
            grader.cpp
            grader.java
        casos/
            casos.zip
        corrector.cpp
    """

    short_name = 'argentina'
    description = 'Argentinian format importer'

    def compat(self) -> bool:
        sys.version_info[0] < 3

    @staticmethod
    def detect(path):
        return False

    def get_task_loader(self, taskname):
        return ArgLoader(os.path.join(self.path, taskname), self.file_cacher)

    def get_contest(self):
        """See docstring in class ContestLoader."""
        args = {}

        name = "imported_contest"

        args["name"] = name
        args["description"] = name

        tasks = [f for f in os.listdir(self.path) if os.path.isdir(f)]

        logger.info("Contest parameters loaded.")

        return Contest(**args), tasks, []

    def get_user(self):
        logger.critical("user loading not supported")

    def get_team(self):
        logger.critical("team loading not supported")

    def get_graders(self, task):
        res = []
        if os.path.exists(os.path.join(self.path, "graders")):
            for lang in [("C++", ".cpp"), ("Java", ".java")]:
                extension = lang[1]
                grader_filename = os.path.join(
                    self.path, "graders", "grader%s" % extension)
                if os.path.exists(grader_filename):
                    digest = self.file_cacher.put_file_from_path(
                        grader_filename,
                        "Grader for task %s and language %s" %
                        (task.name, lang[0]))
                    res += [
                        Manager("grader%s" % extension, digest)]
                else:
                    logger.warning("Grader for language %s not found ", lang)
            return res, "grader"
        else:
            return [], "alone"

    def get_checker(self, task):
        checker_src = os.path.join(self.path, "corrector.cpp")
        if os.path.exists(checker_src):
            compile_checker(checker_src)
            checker_dst = os.path.join(self.path, "checker")
            digest = self.file_cacher.put_file_from_path(
                checker_dst,
                "Manager for task %s" % task.name)
            return [Manager("checker", digest)], "comparator"
        else:
            return [], "diff"

    def get_task_object(self, config, get_statement):
        name = get_with_default(config, "name", "imported_task")

        args = {}

        args["name"] = name
        args["title"] = get_with_default(config, 'title', name)

        logger.info("Loading parameters for task %s.", name)

        if get_statement:
            primary_language = "ar"

            paths = [os.path.join(self.path, name+".pdf")]

            for path in paths:
                if os.path.exists(path):
                    digest = self.file_cacher.put_file_from_path(
                        path,
                        "Statement for task %s (lang: %s)" %
                        (name, primary_language))
                    break
            else:
                logger.critical("Couldn't find any task statement, aborting.")
                sys.exit(1)
            args["statements"] = {
                primary_language: Statement(primary_language, digest)
            }

            if self.compat():
                args["primary_statements"] = "[" + primary_language + "]"
            else:
                args["primary_statements"] = [primary_language]


        if self.compat():
            args["submission_format"] = [
                cms.db.SubmissionFormatElement("%s.%%l" % name)
            ]
        else:
            args["submission_format"] = [
                "%s.%%l" % name
            ]

        args["token_mode"] = "disabled"
        if not self.compat():
            args["score_mode"] = "max_subtask"

        # Attachments
        args["attachments"] = dict()
        if os.path.exists(os.path.join(self.path, "kits")):
            for filename in os.listdir(os.path.join(self.path, "kits")):
                if filename.endswith('.zip'):
                    digest = self.file_cacher.put_file_from_path(
                        os.path.join(self.path, "kits", filename),
                        "Attachment %s for task %s" % (filename, name))
                    args["attachments"][filename] = Attachment(filename, digest)

        task = Task(**args)
        return task

    def get_testcases(self, config, task):
        testcases = os.path.join(self.path, "testcases")
        if os.path.exists(testcases):
            os.remove(testcases)

        cases_path = os.path.join(self.path, "casos", "casos.zip")
        if not os.path.exists(cases_path):
            cases_path = os.path.join(self.path, "casos.zip")
            if not os.path.exists(cases_path):
                logger.critical('no testcases found')
        extract_cases(cases_path, testcases)
        filenames = set(os.listdir(testcases))

        inputs = set([os.path.splitext(f)[0] for f in filenames if f.endswith('.in')])
        outputs = set([os.path.splitext(f)[0] for f in filenames if f.endswith('.dat')])

        if inputs != outputs:
            logger.critical("inputs and outputs don't match")

        res = []
        nuevo = True
        for basename in inputs:
            if re.match('S0[1-9]E.*', basename):
                nuevo = False
            input_filename = basename + ".in"
            output_filename = basename + ".dat"
            if output_filename not in filenames:
                logger.warning("no output for input")
            input_digest = self.file_cacher.put_file_from_path(
                os.path.join(testcases, input_filename),
                "Input %s for task %s" % (basename, task.name))
            output_digest = self.file_cacher.put_file_from_path(
                os.path.join(testcases, output_filename),
                "Output %s for task %s" % (basename, task.name))
            res += [
                Testcase(basename, False, input_digest, output_digest)]

        if 'score_parameters' in config:
            return res, config['score_parameters']

        subtasks = dict()
        for basename in inputs:
            match = re.search('S([0-9]+)E', basename).group(1)
            if not nuevo:
                subtasks[int(match)] = match
            else:
                for c in match:
                    subtasks[int(c)] = c

        scores = config["subtask_scores"]
        scores = {int(k): v for k, v in scores.items()}

        if subtasks.keys() != scores.keys():
            logger.critical("subtasks from config and testcases don't match")

        score_parameters = []
        if nuevo:
            for i, s in subtasks.items():
                score_parameters.append([scores[i], 'S.*%s.*E.*' % s])
        else:
            dependencies = get_with_default(config, "subtask_dependencies", dict())
            dependencies = {int(k): v for k, v in dependencies.items()}
            for i, s in subtasks.items():
                ors = s
                if i in dependencies:
                    if dependencies[i] == 'all':
                        ors = '.*'
                    else:
                        for subsubtask in dependencies[i]:
                            ors += '|' + subtasks[int(subsubtask)]
                score_parameters.append([scores[i], 'S(%s)E.*' % ors])
        return res, score_parameters

    def get_task(self, get_statement=True):
        """See docstring in class TaskLoader."""
        config = json.load(open(os.path.join(self.path, 'config.json')))

        task = self.get_task_object(config, get_statement)

        args = {}
        args["task"] = task
        args["description"] = "Default"
        args["autojudge"] = True

        args["time_limit"] = 1.0
        args["memory_limit"] = 512

        # Builds the parameters that depend on the task type
        args["managers"] = []
        graders, compilation_param = self.get_graders(task)
        checker, evaluation_param = self.get_checker(task)

        args["managers"] += graders
        args["managers"] += checker

        infile_param = ""  # stdin
        outfile_param = ""  # stdout
        args["task_type"] = "Batch"

        if self.compat():
            args["task_type_parameters"] = '["%s", ["%s", "%s"], "%s"]' % (compilation_param, infile_param, outfile_param, evaluation_param)
        else:
            args["task_type_parameters"] = [
                compilation_param,
                [infile_param, outfile_param],
                evaluation_param,
            ]

        args["testcases"], score_config = self.get_testcases(config, task)
        for t in args["testcases"]:
            t.public = True

        args["score_type"] = get_with_default(config, "score_type", "GroupMin")

        if self.compat():
            args["score_type_parameters"] = json.dumps(score_config)
        else:
            args["score_type_parameters"] = score_config

        args["testcases"] = dict((tc.codename, tc) for tc in args["testcases"])
        args["managers"] = dict((mg.filename, mg) for mg in args["managers"])

        dataset = Dataset(**args)
        task.active_dataset = dataset

        logger.info("Task parameters loaded.")

        return task

    def contest_has_changed(self):
        return True

    def user_has_changed(self):
        return True

    def team_has_changed(self):
        return True

    def task_has_changed(self):
        return True
