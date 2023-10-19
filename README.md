# OIAJ backend
## Prerequisites
The only prerequisite is a working `docker` installation

## Running
1. Build the image: `docker compose build`
2. Start the containers: `docker compose up -d`
3. Enter the running container `docker exec -it oiaj bash`
4. Start the services (inside the container): `oia up`

## API
The API can be accessed at `localhost:1367` after starting the services

## Logs
To access the logs run `screen -r log` inside the container

## Troubleshooting
### Cgroups
If submission evaluation is not working, and you get `isolate` errors in the logs, then cgroups v1 might not be enabled in your system. To to enable it, add the line
```
GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=0 cgroup_enable=memory,cpu,cpuacct,blkio,devices,freezer,net_cls,net_prio,pids"
```
to the boot configuration using `sudo nano /etc/default/grub`. If a line with `GRUB_CMDLINE_LINUX` is already present in the file, just add the options
```
systemd.unified_cgroup_hierarchy=0 cgroup_enable=memory,cpu,cpuacct,blkio,devices,freezer,net_cls,net_prio,pids
```
to that line instead.

To apply the changes run `sudo update-grub` and then `sudo reboot`.
