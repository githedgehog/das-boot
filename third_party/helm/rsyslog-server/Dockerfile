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

FROM alpine:3.17

RUN apk add --no-cache rsyslog logrotate bash

EXPOSE 514/tcp
EXPOSE 514/udp

ENV ROTATE_SCHEDULE='0 * * * *' \
    DO_LOG_ALL=true \
    DO_DUMP_TO_STDOUT=true \
    CRON_LOG_LEVEL=8

COPY entrypoint.sh /
COPY etc/ /etc/

RUN echo "cd /var/log; echo \"Current Logs:\"; ls -lth" > /root/.bashrc

ENTRYPOINT ["/entrypoint.sh"]
