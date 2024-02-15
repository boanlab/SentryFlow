# SPDX-License-Identifier: Apache-2.0

# Dockerfile
FROM ubuntu:latest

RUN apt-get update && apt-get -y install python3 python3-pip wget git 

RUN git clone https://github.com/adobe/stringlifier

RUN pip3 install ./stringlifier pymongo Flask

WORKDIR /app

COPY . .

CMD ["python3", "ai-engine.py"]
