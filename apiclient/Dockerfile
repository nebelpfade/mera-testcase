#I'm using ubuntu here due to testcase spec, otherwise I prefer alpine
FROM ubuntu:16.04
MAINTAINER Ivan Gaas <ivan.gaas@gmail.com>

#Install crontab and curl as it missed inside default image
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && apt-get install -y cron curl && \
#Prepare cron.log as symlink to stdout of PID1
    ln -sf /proc/1/fd/1 /var/log/cron.log && \
#Install cron job
    (crontab -l ; echo "*/1 * * * * /apiclient >> /var/log/cron.log 2>&1\n") | crontab && \
#Do clean up
    apt-get clean && rm -rf /var/tmp/* /tmp/*

#Copy apiclient script to the image
COPY apiclient /
#Start crond as PID1
ENTRYPOINT ["cron", "-f"]
#Allow to change log level
CMD ["-l", "8"]
