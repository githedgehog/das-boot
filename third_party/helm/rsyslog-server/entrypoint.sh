#!/bin/bash
set -eu

if [ "${DO_LOG_ALL}" = true ]; then
  echo "Will log all to /var/log/all.log"
else
  echo "Removing config for /var/log/all.log"
  rm /etc/rsyslog.d/all.conf
  rm /etc/logrotate.d/all.conf
  echo "Logging:"
  ls -l /etc/rsyslog.d
  echo "Rotating:"
  ls -l /etc/logrotate.d
fi

if [ "${DO_DUMP_TO_STDOUT}" = true ]; then
  echo "Logs follow"
else
  echo "Does not dump logs to stdout"
  rm /etc/rsyslog.d/stdout.conf
fi

echo "${ROTATE_SCHEDULE} /usr/sbin/logrotate /etc/logrotate.conf" | crontab -
crond -l "${CRON_LOG_LEVEL}"

PIDFILE="/var/run/rsyslogd.pid"
rm -f "${PIDFILE}"
exec rsyslogd -n -f /etc/rsyslogd.conf -i "${PIDFILE}"