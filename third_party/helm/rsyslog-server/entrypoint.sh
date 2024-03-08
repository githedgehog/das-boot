#!/bin/bash
# Copyright 2023 Hedgehog
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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