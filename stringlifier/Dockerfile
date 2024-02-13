# SPDX-License-Identifier: Apache-2.0

# Dockerfile
FROM ubuntu:latest

RUN apt-get update && apt-get install python3 python3-pip wget git -y

RUN git clone https://github.com/isu-kim/stringlifier.git

RUN pip3 install ./stringlifier pymongo Flask

WORKDIR /app

COPY . .

CMD ["python3", "run.py"]