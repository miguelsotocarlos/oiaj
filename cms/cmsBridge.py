#!/usr/bin/env python
# -*- coding: utf-8 -*-

# Contest Management System - http://cms-dev.github.io/
# Copyright © 2015-2018 Stefano Maggiolo <s.maggiolo@gmail.com>
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

"""Utility to submit a solution for a user.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function
from __future__ import unicode_literals
import base64
import http
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import os
import time
from future.builtins.disabled import *  # noqa
from future.builtins import *  # noqa
from six import iteritems

import logging
import sys

from cms import ServiceCoord
from cms.db import File, Participation, SessionGen, Submission, Task, User
from cms.db.filecacher import FileCacher
from cms.io import RemoteServiceClient
from cmscommon.datetime import make_datetime
import re


logger = logging.getLogger(__name__)


def maybe_send_notification(submission_id):
    """Non-blocking attempt to notify a running ES of the submission"""
    rs = RemoteServiceClient(ServiceCoord("EvaluationService", 0))
    rs.connect()
    rs.new_submission(submission_id=submission_id)
    rs.disconnect()


class OiaInternalError(Exception):
    def __init__(self, msg):
        self.msg = msg


class OiaUserError(Exception):
    def __init__(self, msg):
        self.msg = msg


def add_submission(user_id, task_id, timestamp, files, language):
    file_cacher = FileCacher()
    with SessionGen() as session:

        participation = session.query(Participation)\
            .join(Participation.user)\
            .filter(User.id == user_id)\
            .first()
        if participation is None:
            raise OiaUserError(f"User `{user_id}' does not exists or does not participate in the contest.")
        task = session.query(Task)\
            .filter(Task.id == task_id)\
            .first()
        if task is None:
            raise OiaUserError(f"Unable to find task `{task_id}'.")

        elements = set(task.submission_format)

        for file_ in files:
            if file_ not in elements:
                raise OiaUserError(f"File `{file_}' is not in the submission format "
                                 "for the task.")

        # Store all files from the arguments, and obtain their digests..
        file_digests = {}
        try:
            for file_ in files:
                digest = file_cacher.put_file_content(
                    files[file_],
                    "Submission file %s sent by %s at %d."
                    % (file_, user_id, timestamp))
                file_digests[file_] = digest
        except Exception as e:
            raise OiaInternalError(f"Error while storing submission's file: {e}.")

        # Create objects in the DB.
        submission = Submission(make_datetime(timestamp), language,
                                participation=participation, task=task)
        for filename, digest in iteritems(file_digests):
            session.add(File(filename, digest, submission=submission))
        session.add(submission)
        session.commit()
        maybe_send_notification(submission.id)

    return True


def submit(user_id, task_id, timestamp, files, language):
    add_submission(user_id, task_id, timestamp, files, language)


class HTTPRequestHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if re.search('/health', self.path):
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"Hello, world!")

    def do_POST(self):
        if re.search('/submit', self.path):
            try:
                content_length = int(self.headers.get('Content-Length', 0))
                body = self.rfile.read(content_length)
                req = json.loads(body.decode('utf-8'))
                task_id = req['task_id']
                user_id = req['user_id']
                language = req['language']
                files = {}

                for f in req['files']:
                    files[f] = base64.b64decode(req['files'][f])

                submit(user_id, task_id, time.time(), files, language)
                self.send_response(200)
                self.end_headers()
                self.wfile.write(b"Hello, world!")
            except OiaUserError as e:
                self.send_error(code=http.HTTPStatus.BAD_REQUEST, message=e.msg)
            except OiaInternalError as e:
                print(e)
                self.send_error(code=http.HTTPStatus.INTERNAL_SERVER_ERROR, message=e.msg)
            except Exception as e:
                print(e)
                self.send_error(code=http.HTTPStatus.INTERNAL_SERVER_ERROR)


def main():
    server_address = ('', int(os.getenv('OIAJ_SUBMITTER_PORT')))
    httpd = HTTPServer(server_address, HTTPRequestHandler)
    httpd.serve_forever()


if __name__ == "__main__":
    sys.exit(main())
