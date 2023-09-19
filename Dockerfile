FROM ubuntu:18.04

ENV PYTHONUNBUFFERED 1

# Install tzdata first to avoid interactive questions
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && \
    apt-get -y install tzdata

# Install prerequisites
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && \
    apt-get -y install build-essential g++ openjdk-8-jdk-headless \
    postgresql-client python3.6 python3-pip cppreference-doc-en-html \
    cgroup-lite libcap-dev zip wget curl python3.6-dev libpq-dev \
    libcups2-dev libyaml-dev libffi-dev locales screen postgresql-common

# Postgres (just the client)
RUN sh -c "yes | /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh"
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && apt-get -y install postgresql-client-15

# Set locale
RUN locale-gen en_US.UTF-8
ENV LANG en_US.UTF-8
ENV LANGUAGE en_US:en
ENV LC_ALL en_US.UTF-8

# Get CMS
WORKDIR /home
RUN wget https://github.com/cms-dev/cms/releases/download/v1.4.rc1/v1.4.rc1.tar.gz
RUN tar xvf v1.4.rc1.tar.gz

# Install dependencies
WORKDIR /home/cms
RUN pip3 install -r requirements.txt

# Add custom laoder to the CMS
RUN sed -i '30 i from .argentina_loader import ArgLoader' /home/cms/cmscontrib/loaders/__init__.py
RUN sed -i '40 i LOADERS[ArgLoader.short_name] = ArgLoader' /home/cms/cmscontrib/loaders/__init__.py
COPY cms/argentina_loader.py /home/cms/cmscontrib/loaders/

# Build and install CMS
RUN python3 prerequisites.py --as-root build
RUN python3 prerequisites.py --as-root install
RUN python3 setup.py install

# Copy helper scripts
WORKDIR /root/install/oia-scripts
COPY scripts/setup.py /root/install/oia-scripts/setup.py
RUN touch /root/install/oia-scripts/README.md
COPY scripts/src/oia/main.py /root/install/oia-scripts/src/oia/main.py

RUN python3 -m pip install -e /root/install/oia-scripts

# Install golang
RUN wget https://go.dev/dl/go1.20.2.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.20.2.linux-amd64.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive && \
    apt-get -y install git

RUN git config --global --add safe.directory /workspaces/oiajudge

USER root
WORKDIR /workspaces/oiajudge
CMD sleep infinity
